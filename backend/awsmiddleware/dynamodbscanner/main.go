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
	"go.uber.org/zap"
)

type DynamoDBScanner struct {
	dynamodbClient *dynamodb.DynamoDB
	logger         *logger.LogMiddleware
	Gemini         *geminiapi.Gemini
	accountId      string
}

// Initialize a DynamoDB scanner with assumed credentials
func NewDynamoDBScanner(accessKey, secretKey, sessionToken, region string, accountId string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *DynamoDBScanner {
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
		accountId:      accountId,
	}
}

// List all DynamoDB tables
func (scanner *DynamoDBScanner) ListTables(ctx context.Context) ([]*string, error) {
	tracer := otel.Tracer("dynamodbscanner/ListTables")
	ctx, span := tracer.Start(ctx, "ListTables")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Listing DynamoDB tables")

	// List DynamoDB tables
	result, err := scanner.dynamodbClient.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error listing DynamoDB tables: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	scanner.logger.Logger(ctx).Info("DynamoDB tables retrieved",
		zap.Int("table_count", len(result.TableNames)))

	span.SetAttributes(attribute.Int("table_count", len(result.TableNames)))
	return result.TableNames, nil
}

// Check if table encryption is enabled
func (scanner *DynamoDBScanner) CheckTableEncryption(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckTableEncryption")
	ctx, span := tracer.Start(ctx, "CheckTableEncryption")
	defer span.End()

	var findings []string

	scanner.logger.Logger(ctx).Info("Checking table encryption",
		zap.String("table_name", tableName))

	result, err := scanner.dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing table %s: %v", tableName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("table_name", tableName),
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	if result.Table.SSEDescription == nil || aws.StringValue(result.Table.SSEDescription.Status) != "ENABLED" {
		finding := fmt.Sprintf("WARNING: DynamoDB table %s does not have encryption enabled.", tableName)
		scanner.logger.Logger(ctx).Warn("Table encryption not enabled",
			zap.String("table_name", tableName))
		findings = append(findings, finding)
	} else {
		scanner.logger.Logger(ctx).Info("Table encryption is enabled",
			zap.String("table_name", tableName),
			zap.String("encryption_status", aws.StringValue(result.Table.SSEDescription.Status)))
	}

	if result.Table.SSEDescription != nil {
		span.SetAttributes(
			attribute.String("table_name", tableName),
			attribute.String("table_encryption", aws.StringValue(result.Table.SSEDescription.Status)),
		)
	}
	return findings, nil
}

// Check if Point-in-Time Recovery (PITR) is enabled
func (scanner *DynamoDBScanner) CheckPointInTimeRecovery(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckPointInTimeRecovery")
	ctx, span := tracer.Start(ctx, "CheckPointInTimeRecovery")
	defer span.End()

	var findings []string

	scanner.logger.Logger(ctx).Info("Checking point-in-time recovery",
		zap.String("table_name", tableName))

	result, err := scanner.dynamodbClient.DescribeContinuousBackups(&dynamodb.DescribeContinuousBackupsInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing continuous backups for table %s: %v", tableName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("table_name", tableName),
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	if result.ContinuousBackupsDescription.PointInTimeRecoveryDescription == nil ||
		aws.StringValue(result.ContinuousBackupsDescription.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus) != "ENABLED" {
		finding := fmt.Sprintf("WARNING: Point-in-time recovery is not enabled for DynamoDB table %s.", tableName)
		scanner.logger.Logger(ctx).Warn("Point-in-time recovery not enabled",
			zap.String("table_name", tableName))
		findings = append(findings, finding)
	} else {
		scanner.logger.Logger(ctx).Info("Point-in-time recovery is enabled",
			zap.String("table_name", tableName),
			zap.String("pitr_status", aws.StringValue(result.ContinuousBackupsDescription.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus)))
	}

	if result.ContinuousBackupsDescription.PointInTimeRecoveryDescription != nil {
		span.SetAttributes(
			attribute.String("table_name", tableName),
			attribute.String("pitr_status", aws.StringValue(result.ContinuousBackupsDescription.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus)),
		)
	}
	return findings, nil
}

// Check if Auto Scaling is enabled for a table
func (scanner *DynamoDBScanner) CheckAutoScaling(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckAutoScaling")
	ctx, span := tracer.Start(ctx, "CheckAutoScaling")
	defer span.End()

	var findings []string

	scanner.logger.Logger(ctx).Info("Checking auto scaling",
		zap.String("table_name", tableName))

	// Describe auto scaling settings for the table
	result, err := scanner.dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing table %s: %v", tableName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("table_name", tableName),
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	if result.Table.ProvisionedThroughput != nil && aws.Int64Value(result.Table.ProvisionedThroughput.ReadCapacityUnits) > 0 {
		finding := fmt.Sprintf("Auto Scaling is enabled for DynamoDB table %s with Read Capacity Units: %d, Write Capacity Units: %d.",
			tableName,
			aws.Int64Value(result.Table.ProvisionedThroughput.ReadCapacityUnits),
			aws.Int64Value(result.Table.ProvisionedThroughput.WriteCapacityUnits))
		scanner.logger.Logger(ctx).Info("Auto scaling is enabled",
			zap.String("table_name", tableName),
			zap.Int64("read_capacity_units", aws.Int64Value(result.Table.ProvisionedThroughput.ReadCapacityUnits)),
			zap.Int64("write_capacity_units", aws.Int64Value(result.Table.ProvisionedThroughput.WriteCapacityUnits)))
		findings = append(findings, finding)
	} else {
		finding := fmt.Sprintf("WARNING: Auto Scaling is not enabled for DynamoDB table %s.", tableName)
		scanner.logger.Logger(ctx).Warn("Auto scaling not enabled",
			zap.String("table_name", tableName))
		findings = append(findings, finding)
	}

	span.SetAttributes(
		attribute.String("table_name", tableName),
		attribute.Bool("auto_scaling_enabled", result.Table.ProvisionedThroughput != nil &&
			aws.Int64Value(result.Table.ProvisionedThroughput.ReadCapacityUnits) > 0),
	)
	return findings, nil
}

// Check for unused indexes in a DynamoDB table
func (scanner *DynamoDBScanner) CheckUnusedIndexes(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckUnusedIndexes")
	ctx, span := tracer.Start(ctx, "CheckUnusedIndexes")
	defer span.End()

	var findings []string

	scanner.logger.Logger(ctx).Info("Checking for unused indexes",
		zap.String("table_name", tableName))

	result, err := scanner.dynamodbClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing table %s: %v", tableName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("table_name", tableName),
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	if len(result.Table.GlobalSecondaryIndexes) == 0 {
		finding := fmt.Sprintf("No Global Secondary Indexes found for DynamoDB table %s.", tableName)
		scanner.logger.Logger(ctx).Info("No global secondary indexes found",
			zap.String("table_name", tableName))
		findings = append(findings, finding)
	} else {
		scanner.logger.Logger(ctx).Info("Found global secondary indexes",
			zap.String("table_name", tableName),
			zap.Int("index_count", len(result.Table.GlobalSecondaryIndexes)))

		for _, gsi := range result.Table.GlobalSecondaryIndexes {
			indexName := aws.StringValue(gsi.IndexName)
			if aws.StringValue(gsi.IndexStatus) != "ACTIVE" {
				finding := fmt.Sprintf("WARNING: Global Secondary Index %s on table %s is not active or unused.", indexName, tableName)
				scanner.logger.Logger(ctx).Warn("Inactive or unused global secondary index",
					zap.String("table_name", tableName),
					zap.String("index_name", indexName),
					zap.String("index_status", aws.StringValue(gsi.IndexStatus)))
				findings = append(findings, finding)
			} else {
				scanner.logger.Logger(ctx).Info("Global secondary index is active",
					zap.String("table_name", tableName),
					zap.String("index_name", indexName))
			}
		}
	}

	span.SetAttributes(
		attribute.String("table_name", tableName),
		attribute.Int("gsi_count", len(result.Table.GlobalSecondaryIndexes)),
	)
	return findings, nil
}

// Check TTL (Time to Live) status on a table
func (scanner *DynamoDBScanner) CheckTTLStatus(ctx context.Context, tableName string) ([]string, error) {
	tracer := otel.Tracer("dynamodbscanner/CheckTTLStatus")
	ctx, span := tracer.Start(ctx, "CheckTTLStatus")
	defer span.End()

	var findings []string

	scanner.logger.Logger(ctx).Info("Checking TTL status",
		zap.String("table_name", tableName))

	result, err := scanner.dynamodbClient.DescribeTimeToLive(&dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing TTL for table %s: %v", tableName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("table_name", tableName),
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	if result.TimeToLiveDescription == nil || aws.StringValue(result.TimeToLiveDescription.TimeToLiveStatus) != "ENABLED" {
		finding := fmt.Sprintf("WARNING: TTL is not enabled for DynamoDB table %s.", tableName)
		scanner.logger.Logger(ctx).Warn("TTL not enabled",
			zap.String("table_name", tableName))
		findings = append(findings, finding)
	} else {
		scanner.logger.Logger(ctx).Info("TTL is enabled",
			zap.String("table_name", tableName),
			zap.String("ttl_status", aws.StringValue(result.TimeToLiveDescription.TimeToLiveStatus)))
	}

	if result.TimeToLiveDescription != nil {
		span.SetAttributes(
			attribute.String("table_name", tableName),
			attribute.String("ttl_status", aws.StringValue(result.TimeToLiveDescription.TimeToLiveStatus)),
		)
	}
	return findings, nil
}
