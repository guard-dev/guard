package awsmiddleware

import (
	"context"
	"fmt"
	"guarddev/logger"
	"guarddev/modelapi/geminiapi"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type AWSMiddleware struct {
	stsClient *sts.STS
	logger    *logger.LogMiddleware
	gemini    *geminiapi.Gemini
}

type AssumeRoleProps struct {
	AWSAccountID string
	ExternalID   string
}

type AssumeRoleResult struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func Connect(logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *AWSMiddleware {
	tracer := otel.Tracer("awsmiddleware/Connect")
	_, span := tracer.Start(context.Background(), "Connect")
	defer span.End()

	AWS_ACCESS_KEY_ID := os.Getenv("AWS_ACCESS_KEY_ID")
	AWS_SECRET_ACCESS_KEY := os.Getenv("AWS_SECRET_ACCESS_KEY")
	AWS_REGION := os.Getenv("AWS_REGION")

	span.SetAttributes(
		attribute.String("aws.region", AWS_REGION),
	)

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(AWS_REGION),
		Credentials: credentials.NewStaticCredentials(AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, ""),
	}))
	stsClient := sts.New(sess)
	return &AWSMiddleware{stsClient: stsClient, logger: logger, gemini: gemini}
}

func (a *AWSMiddleware) AssumeRole(ctx context.Context, props AssumeRoleProps) (*AssumeRoleResult, error) {
	tracer := otel.Tracer("awsmiddleware/AssumeRole")
	ctx, span := tracer.Start(ctx, "AssumeRole")
	defer span.End()

	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/GuardSecurityScanRole", props.AWSAccountID)

	span.SetAttributes(
		attribute.String("aws.account_id", props.AWSAccountID),
		attribute.String("aws.role_arn", roleArn),
	)

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String("GuardSession"),
		ExternalId:      aws.String(props.ExternalID),
	}

	result, err := a.stsClient.AssumeRoleWithContext(ctx, input)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error assuming role: %v", err)
	}

	return &AssumeRoleResult{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
	}, nil
}
