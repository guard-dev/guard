package dynamodbscanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func (scanner *DynamoDBScanner) ScanDynamoDB(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("dynamodbscanner/ScanDynamoDB")
	ctx, span := tracer.Start(ctx, "ScanDynamoDB")
	defer span.End()

	var allFindings []string

	tables, err := scanner.ListTables(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	// Categorize findings by type
	var encryptionFindings, pitrFindings, autoScalingFindings, ttlFindings, unusedIndexesFindings []string

	for _, tableName := range tables {
		// Check Table Encryption
		encryptionResult, err := scanner.CheckTableEncryption(ctx, *tableName)
		if err != nil {
			encryptionFindings = append(encryptionFindings, fmt.Sprintf("Error checking encryption for table %s: %v", *tableName, err))
		} else {
			encryptionFindings = append(encryptionFindings, encryptionResult...)
		}

		// Check Point-in-Time Recovery (PITR)
		pitrResult, err := scanner.CheckPointInTimeRecovery(ctx, *tableName)
		if err != nil {
			pitrFindings = append(pitrFindings, fmt.Sprintf("Error checking point-in-time recovery for table %s: %v", *tableName, err))
		} else {
			pitrFindings = append(pitrFindings, pitrResult...)
		}

		// Check Auto Scaling
		autoScalingResult, err := scanner.CheckAutoScaling(ctx, *tableName)
		if err != nil {
			autoScalingFindings = append(autoScalingFindings, fmt.Sprintf("Error checking auto scaling for table %s: %v", *tableName, err))
		} else {
			autoScalingFindings = append(autoScalingFindings, autoScalingResult...)
		}

		// Check TTL Settings
		ttlResult, err := scanner.CheckTTLStatus(ctx, *tableName)
		if err != nil {
			ttlFindings = append(ttlFindings, fmt.Sprintf("Error checking TTL for table %s: %v", *tableName, err))
		} else {
			ttlFindings = append(ttlFindings, ttlResult...)
		}

		// Check Unused Indexes
		unusedIndexesResult, err := scanner.CheckUnusedIndexes(ctx, *tableName)
		if err != nil {
			unusedIndexesFindings = append(unusedIndexesFindings, fmt.Sprintf("Error checking unused indexes for table %s: %v", *tableName, err))
		} else {
			unusedIndexesFindings = append(unusedIndexesFindings, unusedIndexesResult...)
		}
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
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, encryptionFindings...)
	allFindings = append(allFindings, pitrFindings...)
	allFindings = append(allFindings, autoScalingFindings...)
	allFindings = append(allFindings, ttlFindings...)
	allFindings = append(allFindings, unusedIndexesFindings...)

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

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	if len(encryptionFindings) > 0 {
		encryptionSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, encryptionFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing encryption findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: encryptionFindings,
			Title:    "Encryption Settings",
			Summary:  encryptionSummary.Summary,
			Remedy:   encryptionSummary.Remedies,
			Commands: encryptionSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Encryption Settings", zap.Any("Encryption Summary", encryptionSummary))
	}

	if len(pitrFindings) > 0 {
		pitrSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, pitrFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing PITR findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: pitrFindings,
			Title:    "Point-in-Time Recovery (PITR)",
			Summary:  pitrSummary.Summary,
			Remedy:   pitrSummary.Remedies,
			Commands: pitrSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Point-in-Time Recovery (PITR)", zap.Any("PITR Summary", pitrSummary))
	}

	if len(autoScalingFindings) > 0 {
		autoScalingSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, autoScalingFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing auto scaling findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: autoScalingFindings,
			Title:    "Auto Scaling Settings",
			Summary:  autoScalingSummary.Summary,
			Remedy:   autoScalingSummary.Remedies,
			Commands: autoScalingSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Auto Scaling Settings", zap.Any("Auto Scaling Summary", autoScalingSummary))
	}

	if len(ttlFindings) > 0 {
		ttlSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, ttlFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing TTL findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: ttlFindings,
			Title:    "Time-to-Live (TTL) Settings",
			Summary:  ttlSummary.Summary,
			Remedy:   ttlSummary.Remedies,
			Commands: ttlSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Time-to-Live (TTL) Settings", zap.Any("TTL Summary", ttlSummary))
	}

	if len(unusedIndexesFindings) > 0 {
		unusedIndexesSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, unusedIndexesFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing unused indexes findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: unusedIndexesFindings,
			Title:    "Unused Indexes",
			Summary:  unusedIndexesSummary.Summary,
			Remedy:   unusedIndexesSummary.Remedies,
			Commands: unusedIndexesSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Unused Indexes", zap.Any("Unused Indexes Summary", unusedIndexesSummary))
	}

	return scanItems, nil
}
