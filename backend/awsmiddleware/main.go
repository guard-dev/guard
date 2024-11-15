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
	stsClient  *sts.STS
	logger     *logger.LogMiddleware
	gemini     *geminiapi.Gemini
	selfHosted bool
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

	AWS_REGION := os.Getenv("AWS_REGION")
	selfHosting := os.Getenv("SELF_HOSTING") != ""

	span.SetAttributes(
		attribute.String("aws.region", AWS_REGION),
	)

	var sess *session.Session
	var stsClient *sts.STS

	if selfHosting {
		// Self-Hosting: Use provided credentials directly
		AWS_ACCESS_KEY_ID := os.Getenv("AWS_ACCESS_KEY_ID")
		AWS_SECRET_ACCESS_KEY := os.Getenv("AWS_SECRET_ACCESS_KEY")

		sess = session.Must(session.NewSession(&aws.Config{
			Region:      aws.String(AWS_REGION),
			Credentials: credentials.NewStaticCredentials(AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, ""),
		}))
	} else {
		// Production: Create a session that will use STS
		sess = session.Must(session.NewSession(&aws.Config{
			Region: aws.String(AWS_REGION),
		}))
		stsClient = sts.New(sess)
	}

	return &AWSMiddleware{
		stsClient:  stsClient,
		logger:     logger,
		gemini:     gemini,
		selfHosted: selfHosting,
	}
}

func (a *AWSMiddleware) AssumeRole(ctx context.Context, props AssumeRoleProps) (*AssumeRoleResult, error) {
	tracer := otel.Tracer("awsmiddleware/AssumeRole")
	ctx, span := tracer.Start(ctx, "AssumeRole")
	defer span.End()

	if a.selfHosted {
		// Self-Hosting: Direct access is used, no role assumption necessary
		return nil, fmt.Errorf("AssumeRole is not applicable in self-hosting mode")
	}

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

// GetAccountID retrieves the AWS account ID for the given access key and secret key
func (a *AWSMiddleware) GetAccountID(accessKey, secretKey string) (string, error) {
	// Create a new AWS session with provided credentials
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:      aws.String("us-west-2"), // Region is required, but it doesn’t matter for STS calls
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session, %w", err)
	}

	// Create an STS client
	svc := sts.New(sess)

	// Call GetCallerIdentity to get account information
	result, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity, %w", err)
	}

	// Return the Account ID
	return *result.Account, nil
}
