package dynamodbscanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (scanner *DynamoDBScanner) ScanDynamoDB(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("dynamodbscanner/ScanDynamoDB")
	ctx, span := tracer.Start(ctx, "ScanDynamoDB")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting DynamoDB scan",
		zap.String("region", region))

	var allFindings []string

	tables, err := scanner.ListTables(ctx)
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error listing DynamoDB tables: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("region", region),
			zap.Error(err))
		return nil, nil, fmt.Errorf(errorMsg)
	}

	scanner.logger.Logger(ctx).Info("Retrieved DynamoDB tables",
		zap.String("region", region),
		zap.Int("table_count", len(tables)))

	// Categorize findings by type
	var encryptionFindings, pitrFindings, autoScalingFindings, ttlFindings, unusedIndexesFindings []string

	for _, tableName := range tables {
		scanner.logger.Logger(ctx).Info("Scanning table",
			zap.String("table_name", *tableName),
			zap.String("region", region))

		// Check Table Encryption
		encryptionResult, err := scanner.CheckTableEncryption(ctx, *tableName)
		if err != nil {
			errorMsg := fmt.Sprintf("Error checking encryption for table %s: %v", *tableName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("table_name", *tableName),
				zap.Error(err))
			encryptionFindings = append(encryptionFindings, errorMsg)
		} else {
			encryptionFindings = append(encryptionFindings, encryptionResult...)
		}

		// Check Point-in-Time Recovery (PITR)
		pitrResult, err := scanner.CheckPointInTimeRecovery(ctx, *tableName)
		if err != nil {
			errorMsg := fmt.Sprintf("Error checking point-in-time recovery for table %s: %v", *tableName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("table_name", *tableName),
				zap.Error(err))
			pitrFindings = append(pitrFindings, errorMsg)
		} else {
			pitrFindings = append(pitrFindings, pitrResult...)
		}

		// Check Auto Scaling
		autoScalingResult, err := scanner.CheckAutoScaling(ctx, *tableName)
		if err != nil {
			errorMsg := fmt.Sprintf("Error checking auto scaling for table %s: %v", *tableName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("table_name", *tableName),
				zap.Error(err))
			autoScalingFindings = append(autoScalingFindings, errorMsg)
		} else {
			autoScalingFindings = append(autoScalingFindings, autoScalingResult...)
		}

		// Check TTL Settings
		ttlResult, err := scanner.CheckTTLStatus(ctx, *tableName)
		if err != nil {
			errorMsg := fmt.Sprintf("Error checking TTL for table %s: %v", *tableName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("table_name", *tableName),
				zap.Error(err))
			ttlFindings = append(ttlFindings, errorMsg)
		} else {
			ttlFindings = append(ttlFindings, ttlResult...)
		}

		// Check Unused Indexes
		unusedIndexesResult, err := scanner.CheckUnusedIndexes(ctx, *tableName)
		if err != nil {
			errorMsg := fmt.Sprintf("Error checking unused indexes for table %s: %v", *tableName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("table_name", *tableName),
				zap.Error(err))
			unusedIndexesFindings = append(unusedIndexesFindings, errorMsg)
		} else {
			unusedIndexesFindings = append(unusedIndexesFindings, unusedIndexesResult...)
		}

		scanner.logger.Logger(ctx).Info("Completed table scan",
			zap.String("table_name", *tableName),
			zap.Int("encryption_findings", len(encryptionResult)),
			zap.Int("pitr_findings", len(pitrResult)),
			zap.Int("auto_scaling_findings", len(autoScalingResult)),
			zap.Int("ttl_findings", len(ttlResult)),
			zap.Int("unused_indexes_findings", len(unusedIndexesResult)))
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.processAndSummarizeDynamoDBFindings(
		ctx,
		"dynamodb",
		region,
		encryptionFindings,
		pitrFindings,
		autoScalingFindings,
		ttlFindings,
		unusedIndexesFindings,
	)
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error processing and summarizing findings: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("region", region),
			zap.Error(err))
		return nil, nil, fmt.Errorf(errorMsg)
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, encryptionFindings...)
	allFindings = append(allFindings, pitrFindings...)
	allFindings = append(allFindings, autoScalingFindings...)
	allFindings = append(allFindings, ttlFindings...)
	allFindings = append(allFindings, unusedIndexesFindings...)

	scanner.logger.Logger(ctx).Info("Completed DynamoDB scan",
		zap.String("region", region),
		zap.Int("total_findings", len(allFindings)),
		zap.Int("scan_items", len(scanItems)))

	span.SetAttributes(
		attribute.String("region", region),
		attribute.Int("total_findings", len(allFindings)),
		attribute.Int("scan_items", len(scanItems)),
	)

	return allFindings, scanItems, nil
}

func (scanner *DynamoDBScanner) processAndSummarizeDynamoDBFindings(
	ctx context.Context,
	service string,
	region string,
	encryptionFindings []string,
	pitrFindings []string,
	autoScalingFindings []string,
	ttlFindings []string,
	unusedIndexesFindings []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("dynamodbscanner/processAndSummarizeDynamoDBFindings")
	ctx, span := tracer.Start(ctx, "processAndSummarizeDynamoDBFindings")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Processing and summarizing findings",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", scanner.accountId))

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	if len(encryptionFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing encryption findings",
			zap.Int("finding_count", len(encryptionFindings)))

		encryptionSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, scanner.accountId, encryptionFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing encryption findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: encryptionFindings,
			Title:    "Encryption Settings",
			Summary:  encryptionSummary.Summary,
			Remedy:   encryptionSummary.Remedies,
			Commands: encryptionSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Encryption findings summarized",
			zap.Int("finding_count", len(encryptionFindings)),
			zap.Any("summary", encryptionSummary))
	}

	if len(pitrFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing PITR findings",
			zap.Int("finding_count", len(pitrFindings)))

		pitrSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, scanner.accountId, pitrFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing PITR findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: pitrFindings,
			Title:    "Point-in-Time Recovery (PITR)",
			Summary:  pitrSummary.Summary,
			Remedy:   pitrSummary.Remedies,
			Commands: pitrSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("PITR findings summarized",
			zap.Int("finding_count", len(pitrFindings)),
			zap.Any("summary", pitrSummary))
	}

	if len(autoScalingFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing auto scaling findings",
			zap.Int("finding_count", len(autoScalingFindings)))

		autoScalingSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, scanner.accountId, autoScalingFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing auto scaling findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: autoScalingFindings,
			Title:    "Auto Scaling Settings",
			Summary:  autoScalingSummary.Summary,
			Remedy:   autoScalingSummary.Remedies,
			Commands: autoScalingSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Auto scaling findings summarized",
			zap.Int("finding_count", len(autoScalingFindings)),
			zap.Any("summary", autoScalingSummary))
	}

	if len(ttlFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing TTL findings",
			zap.Int("finding_count", len(ttlFindings)))

		ttlSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, scanner.accountId, ttlFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing TTL findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: ttlFindings,
			Title:    "Time-to-Live (TTL) Settings",
			Summary:  ttlSummary.Summary,
			Remedy:   ttlSummary.Remedies,
			Commands: ttlSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("TTL findings summarized",
			zap.Int("finding_count", len(ttlFindings)),
			zap.Any("summary", ttlSummary))
	}

	if len(unusedIndexesFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing unused indexes findings",
			zap.Int("finding_count", len(unusedIndexesFindings)))

		unusedIndexesSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, scanner.accountId, unusedIndexesFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing unused indexes findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: unusedIndexesFindings,
			Title:    "Unused Indexes",
			Summary:  unusedIndexesSummary.Summary,
			Remedy:   unusedIndexesSummary.Remedies,
			Commands: unusedIndexesSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Unused indexes findings summarized",
			zap.Int("finding_count", len(unusedIndexesFindings)),
			zap.Any("summary", unusedIndexesSummary))
	}

	scanner.logger.Logger(ctx).Info("Completed processing and summarizing findings",
		zap.String("service", service),
		zap.String("region", region),
		zap.Int("total_scan_items", len(scanItems)))

	span.SetAttributes(
		attribute.String("service", service),
		attribute.String("region", region),
		attribute.Int("total_scan_items", len(scanItems)),
	)

	return scanItems, nil
}
