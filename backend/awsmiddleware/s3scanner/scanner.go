package s3scanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"github.com/aws/aws-sdk-go/aws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (scanner *S3Scanner) ScanS3(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("s3scanner/ScanS3")
	ctx, span := tracer.Start(ctx, "ScanS3")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting S3 scan",
		zap.String("region", region),
		zap.String("account_id", scanner.accountId))

	span.SetAttributes(
		attribute.String("aws.region", region),
		attribute.String("aws.account_id", scanner.accountId),
	)

	var allFindings []string

	buckets, err := scanner.ListBuckets(ctx)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list buckets",
			zap.String("region", region),
			zap.Error(err))
		return nil, nil, err
	}

	scanner.logger.Logger(ctx).Info("Found S3 buckets",
		zap.Int("bucket_count", len(buckets)))
	span.SetAttributes(attribute.Int("bucket_count", len(buckets)))

	// Categorize findings by type
	var publicAccessFindings, encryptionFindings, versioningFindings, loggingFindings, blockPublicAccessFindings, lifecyclePolicyFindings, mfaDeleteFindings []string

	for _, bucket := range buckets {
		bucketName := aws.StringValue(bucket.Name)
		allFindings = append(allFindings, fmt.Sprintf("Found S3 Bucket: %s", bucketName))

		scanner.logger.Logger(ctx).Debug("Scanning bucket",
			zap.String("bucket", bucketName))

		// Public access findings
		_, publicFinding, err := scanner.IsBucketPublic(ctx, bucketName)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to check public access",
				zap.String("bucket", bucketName),
				zap.Error(err))
			publicAccessFindings = append(publicAccessFindings, fmt.Sprintf("Error checking public access for bucket %s: %v", bucketName, err))
		} else {
			publicAccessFindings = append(publicAccessFindings, publicFinding)
		}

		// Encryption findings
		_, encryptionFinding, err := scanner.IsEncryptionEnabled(ctx, bucketName)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to check encryption",
				zap.String("bucket", bucketName),
				zap.Error(err))
			encryptionFindings = append(encryptionFindings, fmt.Sprintf("Error checking encryption for bucket %s: %v", bucketName, err))
		} else {
			encryptionFindings = append(encryptionFindings, encryptionFinding)
		}

		// Versioning findings
		_, versioningFinding, err := scanner.IsVersioningEnabled(ctx, bucketName)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to check versioning",
				zap.String("bucket", bucketName),
				zap.Error(err))
			versioningFindings = append(versioningFindings, fmt.Sprintf("Error checking versioning for bucket %s: %v", bucketName, err))
		} else {
			versioningFindings = append(versioningFindings, versioningFinding)
		}

		// Logging findings
		_, loggingFinding, err := scanner.IsLoggingEnabled(ctx, bucketName)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to check logging",
				zap.String("bucket", bucketName),
				zap.Error(err))
			loggingFindings = append(loggingFindings, fmt.Sprintf("Error checking logging for bucket %s: %v", bucketName, err))
		} else {
			loggingFindings = append(loggingFindings, loggingFinding)
		}

		// Block public access findings
		_, blockPublicAccessFinding, err := scanner.CheckBlockPublicAccess(ctx, bucketName)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to check block public access",
				zap.String("bucket", bucketName),
				zap.Error(err))
			blockPublicAccessFindings = append(blockPublicAccessFindings, fmt.Sprintf("Error checking block public access for bucket %s: %v", bucketName, err))
		} else {
			blockPublicAccessFindings = append(blockPublicAccessFindings, blockPublicAccessFinding)
		}

		// Lifecycle policy findings
		_, lifecyclePolicyFinding, err := scanner.IsLifecyclePolicyEnabled(ctx, bucketName)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to check lifecycle policy",
				zap.String("bucket", bucketName),
				zap.Error(err))
			lifecyclePolicyFindings = append(lifecyclePolicyFindings, fmt.Sprintf("Error checking lifecycle policy for bucket %s: %v", bucketName, err))
		} else {
			lifecyclePolicyFindings = append(lifecyclePolicyFindings, lifecyclePolicyFinding)
		}

		// MFA Delete findings
		_, mfaDeleteFinding, err := scanner.IsMFADeleteEnabled(ctx, bucketName)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to check MFA delete",
				zap.String("bucket", bucketName),
				zap.Error(err))
			mfaDeleteFindings = append(mfaDeleteFindings, fmt.Sprintf("Error checking MFA Delete for bucket %s: %v", bucketName, err))
		} else {
			mfaDeleteFindings = append(mfaDeleteFindings, mfaDeleteFinding)
		}

		scanner.logger.Logger(ctx).Debug("Completed bucket scan",
			zap.String("bucket", bucketName))
	}

	scanner.logger.Logger(ctx).Info("Processing and summarizing findings")
	scanItems, err := scanner.processAndSummarizeS3Findings(
		ctx,
		"s3",
		region,
		scanner.accountId,
		publicAccessFindings,
		encryptionFindings,
		versioningFindings,
		loggingFindings,
		blockPublicAccessFindings,
		lifecyclePolicyFindings,
		mfaDeleteFindings,
	)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to process and summarize findings",
			zap.Error(err))
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, publicAccessFindings...)
	allFindings = append(allFindings, encryptionFindings...)
	allFindings = append(allFindings, versioningFindings...)
	allFindings = append(allFindings, loggingFindings...)
	allFindings = append(allFindings, blockPublicAccessFindings...)
	allFindings = append(allFindings, lifecyclePolicyFindings...)
	allFindings = append(allFindings, mfaDeleteFindings...)

	scanner.logger.Logger(ctx).Info("Completed S3 scan",
		zap.Int("total_findings", len(allFindings)))
	span.SetAttributes(attribute.Int("total_findings", len(allFindings)))

	return allFindings, scanItems, nil
}

func (scanner *S3Scanner) processAndSummarizeS3Findings(
	ctx context.Context,
	service string,
	region string,
	accountId string,
	publicAccess []string,
	encryption []string,
	versioning []string,
	logging []string,
	blockPublicAccess []string,
	lifecyclePolicy []string,
	mfaDelete []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("s3scanner/processAndSummarizeS3Findings")
	ctx, span := tracer.Start(ctx, "processAndSummarizeS3Findings")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Processing findings",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", accountId))

	span.SetAttributes(
		attribute.String("service", service),
		attribute.String("region", region),
		attribute.String("accountId", accountId),
	)

	var scanItems []postgres.CreateNewScanItemEntryParams

	if len(publicAccess) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing public access findings",
			zap.Int("finding_count", len(publicAccess)))
		publicAccessSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, publicAccess)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize public access findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing public access findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: publicAccess,
			Title:    "Public Access",
			Summary:  publicAccessSummary.Summary,
			Remedy:   publicAccessSummary.Remedies,
			Commands: publicAccessSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Public Access Summary",
			zap.Any("summary", publicAccessSummary))
	}

	if len(encryption) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing encryption findings",
			zap.Int("finding_count", len(encryption)))
		encryptionSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, encryption)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize encryption findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing encryption findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: encryption,
			Title:    "Encryption",
			Summary:  encryptionSummary.Summary,
			Remedy:   encryptionSummary.Remedies,
			Commands: encryptionSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Encryption Summary",
			zap.Any("summary", encryptionSummary))
	}

	if len(versioning) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing versioning findings",
			zap.Int("finding_count", len(versioning)))
		versioningSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, versioning)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize versioning findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing versioning findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: versioning,
			Title:    "Versioning",
			Summary:  versioningSummary.Summary,
			Remedy:   versioningSummary.Remedies,
			Commands: versioningSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Versioning Summary",
			zap.Any("summary", versioningSummary))
	}

	if len(logging) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing logging findings",
			zap.Int("finding_count", len(logging)))
		loggingSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, logging)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize logging findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing logging findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: logging,
			Title:    "Logging",
			Summary:  loggingSummary.Summary,
			Remedy:   loggingSummary.Remedies,
			Commands: loggingSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Logging Summary",
			zap.Any("summary", loggingSummary))
	}

	if len(blockPublicAccess) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing block public access findings",
			zap.Int("finding_count", len(blockPublicAccess)))
		blockPublicAccessSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, blockPublicAccess)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize block public access findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing block public access findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: blockPublicAccess,
			Title:    "Block Public Access",
			Summary:  blockPublicAccessSummary.Summary,
			Remedy:   blockPublicAccessSummary.Remedies,
			Commands: blockPublicAccessSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Block Public Access Summary",
			zap.Any("summary", blockPublicAccessSummary))
	}

	if len(lifecyclePolicy) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing lifecycle policy findings",
			zap.Int("finding_count", len(lifecyclePolicy)))
		lifecyclePolicySummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, lifecyclePolicy)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize lifecycle policy findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing lifecycle policy findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: lifecyclePolicy,
			Title:    "Lifecycle Policy",
			Summary:  lifecyclePolicySummary.Summary,
			Remedy:   lifecyclePolicySummary.Remedies,
			Commands: lifecyclePolicySummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Lifecycle Policy Summary",
			zap.Any("summary", lifecyclePolicySummary))
	}

	if len(mfaDelete) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing MFA delete findings",
			zap.Int("finding_count", len(mfaDelete)))
		mfaDeleteSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, mfaDelete)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize MFA delete findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing MFA delete findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: mfaDelete,
			Title:    "MFA Delete",
			Summary:  mfaDeleteSummary.Summary,
			Remedy:   mfaDeleteSummary.Remedies,
			Commands: mfaDeleteSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("MFA Delete Summary",
			zap.Any("summary", mfaDeleteSummary))
	}

	span.SetAttributes(attribute.Int("scan_items_count", len(scanItems)))

	return scanItems, nil
}
