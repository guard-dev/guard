package awsmiddleware

import (
	"context"
	"fmt"

	"guarddev/awsmiddleware/dynamodbscanner"
	"guarddev/awsmiddleware/ec2scanner"
	"guarddev/awsmiddleware/ecsscanner"
	"guarddev/awsmiddleware/iamscanner"
	"guarddev/awsmiddleware/lambdascanner"
	"guarddev/awsmiddleware/s3scanner"
	"guarddev/database/postgres"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type ScanResults struct {
	ScanItem        postgres.ScanItem
	ScanItemEntries []postgres.CreateNewScanItemEntryParams
}

func (a *AWSMiddleware) StartScan(ctx context.Context, accessKey, secretKey, sessionToken string, regions, services []string) ([]ScanResults, error) {
	tracer := otel.Tracer("awsmiddleware/StartScan")
	ctx, span := tracer.Start(ctx, "StartScan")
	defer span.End()

	span.SetAttributes(
		attribute.StringSlice("aws.regions", regions),
		attribute.StringSlice("aws.services", services),
	)

	var results []ScanResults

	logger := a.logger.Logger(ctx)

	for _, region := range regions {
		for _, service := range services {
			logger.Info("[AWSMiddleware/StartScan] Scanning Service", zap.String("Service", service), zap.String("Region", region))
			result, scanItems, err := a.scanService(ctx, accessKey, secretKey, sessionToken, region, service)
			if err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("error scanning %s in %s: %v", service, region, err)
			}
			results = append(results, ScanResults{
				ScanItem:        result,
				ScanItemEntries: scanItems,
			})
		}
	}

	return results, nil
}

func (a *AWSMiddleware) scanService(ctx context.Context, accessKey, secretKey, sessionToken, region, service string) (postgres.ScanItem, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("awsmiddleware/scanService")
	ctx, span := tracer.Start(ctx, "scanService")
	defer span.End()

	span.SetAttributes(
		attribute.String("aws.region", region),
		attribute.String("aws.service", service),
	)

	result := postgres.ScanItem{
		Service: service,
		Region:  region,
	}

	var scanItemEntries []postgres.CreateNewScanItemEntryParams = []postgres.CreateNewScanItemEntryParams{}

	var err error
	var findings []string

	switch service {
	case "s3":
		scanner := s3scanner.NewS3Scanner(accessKey, secretKey, sessionToken, region, a.logger, a.gemini)
		findings, scanItemEntries, err = scanner.ScanS3(ctx, region)
	case "ec2":
		scanner := ec2scanner.NewEC2Scanner(accessKey, secretKey, sessionToken, region, a.logger, a.gemini)
		findings, scanItemEntries, err = scanner.ScanEC2(ctx, region)
	case "ecs":
		scanner := ecsscanner.NewECSScanner(accessKey, secretKey, sessionToken, region, a.logger, a.gemini)
		findings, scanItemEntries, err = scanner.ScanECS(ctx, region)
	case "lambda":
		scanner := lambdascanner.NewLambdaScanner(accessKey, secretKey, sessionToken, region, a.logger, a.gemini)
		findings, scanItemEntries, err = scanner.ScanLambda(ctx, region)
	case "dynamodb":
		scanner := dynamodbscanner.NewDynamoDBScanner(accessKey, secretKey, sessionToken, region, a.logger, a.gemini)
		findings, scanItemEntries, err = scanner.ScanDynamoDB(ctx, region)
	case "iam":
		scanner := iamscanner.NewIAMScanner(accessKey, secretKey, sessionToken, region, a.logger, a.gemini)
		findings, scanItemEntries, err = scanner.ScanIAM(ctx, region)
	default:
		err = fmt.Errorf("unsupported service: %s", service)
	}

	if err != nil {
		span.RecordError(err)
		return result, nil, err
	}

	result.Findings = findings
	span.SetAttributes(attribute.Int("findings.count", len(findings)))

	return result, scanItemEntries, nil
}
