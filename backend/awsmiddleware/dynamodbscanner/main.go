package dynamodbscanner

import (
	"context"
	"fmt"

	"guarddev/logger"
	"guarddev/modelapi/geminiapi"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type DynamoDBScanner struct {
	dynamodbClient *dynamodb.DynamoDB
	logger         *logger.LogMiddleware
	Gemini         *geminiapi.Gemini
}

// Initialize a DynamoDB scanner with assumed credentials
func NewDynamoDBScanner(accessKey, secretKey, sessionToken, region string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *DynamoDBScanner {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			sessionToken,
		),
	}))
	return &DynamoDBScanner{
		dynamodbClient: dynamodb.New(sess),
		logger:         logger,
		Gemini:         gemini,
	}
}

// List all DynamoDB tables
func (scanner *DynamoDBScanner) ListTables(ctx context.Context) ([]*string, error) {
	tracer := otel.Tracer("dynamodbscanner/ListTables")
	ctx, span := tracer.Start(ctx, "ListTables")
	defer span.End()

	// List DynamoDB tables
	result, err := scanner.dynamodbClient.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error listing DynamoDB tables: %v", err)
	}

	span.SetAttributes(attribute.Int("table_count", len(result.TableNames)))
	return result.TableNames, nil
}

// Check if table encryption is enabled
func (scanner *DynamoDBScanner) CheckTableEncryption(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckTableEncryption")
	ctx, span := tracer.Start(ctx, "CheckTableEncryption")
	defer span.End()

	var findings []string

	result, err := scanner.dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing table %s: %v", tableName, err)
	}

	if result.Table.SSEDescription == nil || aws.StringValue(result.Table.SSEDescription.Status) != "ENABLED" {
		findings = append(findings, fmt.Sprintf("WARNING: DynamoDB table %s does not have encryption enabled.", tableName))
	}

	if result.Table.SSEDescription != nil {
		span.SetAttributes(attribute.String("table_encryption", aws.StringValue(result.Table.SSEDescription.Status)))
	}
	return findings, nil
}

// Check if Point-in-Time Recovery (PITR) is enabled
func (scanner *DynamoDBScanner) CheckPointInTimeRecovery(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckPointInTimeRecovery")
	ctx, span := tracer.Start(ctx, "CheckPointInTimeRecovery")
	defer span.End()

	var findings []string

	result, err := scanner.dynamodbClient.DescribeContinuousBackups(&dynamodb.DescribeContinuousBackupsInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing continuous backups for table %s: %v", tableName, err)
	}

	if result.ContinuousBackupsDescription.PointInTimeRecoveryDescription == nil || aws.StringValue(result.ContinuousBackupsDescription.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus) != "ENABLED" {
		findings = append(findings, fmt.Sprintf("WARNING: Point-in-time recovery is not enabled for DynamoDB table %s.", tableName))
	}

	if result.ContinuousBackupsDescription.PointInTimeRecoveryDescription != nil {
		span.SetAttributes(attribute.String("pitr_status", aws.StringValue(result.ContinuousBackupsDescription.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus)))
	}
	return findings, nil
}

// Check if Auto Scaling is enabled for a table
func (scanner *DynamoDBScanner) CheckAutoScaling(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckAutoScaling")
	ctx, span := tracer.Start(ctx, "CheckAutoScaling")
	defer span.End()

	var findings []string

	// Describe auto scaling settings for the table
	result, err := scanner.dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing table %s: %v", tableName, err)
	}

	if result.Table.ProvisionedThroughput != nil && aws.Int64Value(result.Table.ProvisionedThroughput.ReadCapacityUnits) > 0 {
		findings = append(findings, fmt.Sprintf("Auto Scaling is enabled for DynamoDB table %s with Read Capacity Units: %d, Write Capacity Units: %d.", tableName, aws.Int64Value(result.Table.ProvisionedThroughput.ReadCapacityUnits), aws.Int64Value(result.Table.ProvisionedThroughput.WriteCapacityUnits)))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: Auto Scaling is not enabled for DynamoDB table %s.", tableName))
	}

	span.SetAttributes(attribute.Bool("auto_scaling_enabled", len(findings) == 0))
	return findings, nil
}

// Check for unused indexes in a DynamoDB table
func (scanner *DynamoDBScanner) CheckUnusedIndexes(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckUnusedIndexes")
	ctx, span := tracer.Start(ctx, "CheckUnusedIndexes")
	defer span.End()

	var findings []string

	result, err := scanner.dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing table %s: %v", tableName, err)
	}

	if len(result.Table.GlobalSecondaryIndexes) == 0 {
		findings = append(findings, fmt.Sprintf("No Global Secondary Indexes found for DynamoDB table %s.", tableName))
	} else {
		for _, gsi := range result.Table.GlobalSecondaryIndexes {
			if aws.StringValue(gsi.IndexStatus) != "ACTIVE" {
				findings = append(findings, fmt.Sprintf("WARNING: Global Secondary Index %s on table %s is not active or unused.", aws.StringValue(gsi.IndexName), tableName))
			}
		}
	}

	span.SetAttributes(attribute.Int("gsi_count", len(result.Table.GlobalSecondaryIndexes)))
	return findings, nil
}

// Check TTL (Time to Live) status on a table
func (scanner *DynamoDBScanner) CheckTTLStatus(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckTTLStatus")
	ctx, span := tracer.Start(ctx, "CheckTTLStatus")
	defer span.End()

	var findings []string

	result, err := scanner.dynamodbClient.DescribeTimeToLive(&dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing TTL for table %s: %v", tableName, err)
	}

	if result.TimeToLiveDescription == nil || aws.StringValue(result.TimeToLiveDescription.TimeToLiveStatus) != "ENABLED" {
		findings = append(findings, fmt.Sprintf("WARNING: TTL is not enabled for DynamoDB table %s.", tableName))
	}

	if result.TimeToLiveDescription != nil {
		span.SetAttributes(attribute.String("ttl_status", aws.StringValue(result.TimeToLiveDescription.TimeToLiveStatus)))
	}
	return findings, nil
}
