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

	var allFindings []string

	buckets, err := scanner.ListBuckets(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	span.SetAttributes(attribute.Int("bucket_count", len(buckets)))

	// Categorize findings by type
	var publicAccessFindings, encryptionFindings, versioningFindings, loggingFindings, blockPublicAccessFindings, lifecyclePolicyFindings, mfaDeleteFindings []string

	for _, bucket := range buckets {
		bucketName := aws.StringValue(bucket.Name)
		allFindings = append(allFindings, fmt.Sprintf("Found S3 Bucket: %s", bucketName))

		// Public access findings
		_, publicFinding, err := scanner.IsBucketPublic(ctx, bucketName)
		if err != nil {
			span.RecordError(err)
			publicAccessFindings = append(publicAccessFindings, fmt.Sprintf("Error checking public access for bucket %s: %v", bucketName, err))
		} else {
			publicAccessFindings = append(publicAccessFindings, publicFinding)
		}

		// Encryption findings
		_, encryptionFinding, err := scanner.IsEncryptionEnabled(ctx, bucketName)
		if err != nil {
			encryptionFindings = append(encryptionFindings, fmt.Sprintf("Error checking encryption for bucket %s: %v", bucketName, err))
		} else {
			encryptionFindings = append(encryptionFindings, encryptionFinding)
		}

		// Versioning findings
		_, versioningFinding, err := scanner.IsVersioningEnabled(ctx, bucketName)
		if err != nil {
			versioningFindings = append(versioningFindings, fmt.Sprintf("Error checking versioning for bucket %s: %v", bucketName, err))
		} else {
			versioningFindings = append(versioningFindings, versioningFinding)
		}

		// Logging findings
		_, loggingFinding, err := scanner.IsLoggingEnabled(ctx, bucketName)
		if err != nil {
			loggingFindings = append(loggingFindings, fmt.Sprintf("Error checking logging for bucket %s: %v", bucketName, err))
		} else {
			loggingFindings = append(loggingFindings, loggingFinding)
		}

		// Block public access findings
		_, blockPublicAccessFinding, err := scanner.CheckBlockPublicAccess(ctx, bucketName)
		if err != nil {
			blockPublicAccessFindings = append(blockPublicAccessFindings, fmt.Sprintf("Error checking block public access settings for bucket %s: %v", bucketName, err))
		} else {
			blockPublicAccessFindings = append(blockPublicAccessFindings, blockPublicAccessFinding)
		}

		// Lifecycle policy findings
		_, lifecycleFinding, err := scanner.IsLifecyclePolicyEnabled(ctx, bucketName)
		if err != nil {
			lifecyclePolicyFindings = append(lifecyclePolicyFindings, fmt.Sprintf("Error checking lifecycle policy for bucket %s: %v", bucketName, err))
		} else {
			lifecyclePolicyFindings = append(lifecyclePolicyFindings, lifecycleFinding)
		}

		// MFA delete findings
		_, mfaDeleteFinding, err := scanner.IsMFADeleteEnabled(ctx, bucketName)
		if err != nil {
			mfaDeleteFindings = append(mfaDeleteFindings, fmt.Sprintf("Error checking MFA delete for bucket %s: %v", bucketName, err))
		} else {
			mfaDeleteFindings = append(mfaDeleteFindings, mfaDeleteFinding)
		}
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.processAndSummarizeS3Findings(
		ctx,
		"s3",
		region,
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

	span.SetAttributes(attribute.Int("total_findings", len(allFindings)))

	return allFindings, scanItems, nil
}

func (scanner *S3Scanner) processAndSummarizeS3Findings(
	ctx context.Context,
	service string,
	region string,
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

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Define prompts for each type of finding and call the LLM
	if len(publicAccess) > 0 {
		publicAccessSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, publicAccess)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing public access findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: publicAccess,
			Title:    "Public Access",
			Summary:  publicAccessSummary.Summary,
			Remedy:   publicAccessSummary.Remedies,
			Commands: publicAccessSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Public Access Findings", zap.Any("Public Access", publicAccessSummary))
	}

	if len(encryption) > 0 {
		encryptionSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, encryption)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing encryption findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: encryption,
			Title:    "Encryption",
			Summary:  encryptionSummary.Summary,
			Remedy:   encryptionSummary.Remedies,
			Commands: encryptionSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Encryption Findings", zap.Any("Encryption", encryptionSummary))
	}

	if len(versioning) > 0 {
		versioningSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, versioning)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing versioning findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: versioning,
			Title:    "Versioning",
			Summary:  versioningSummary.Summary,
			Remedy:   versioningSummary.Remedies,
			Commands: versioningSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Versioning Findings", zap.Any("Versioning", versioningSummary))
	}

	if len(logging) > 0 {
		loggingSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, logging)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing logging findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: logging,
			Title:    "Logging",
			Summary:  loggingSummary.Summary,
			Remedy:   loggingSummary.Remedies,
			Commands: loggingSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Logging Findings", zap.Any("Logging", loggingSummary))
	}

	if len(blockPublicAccess) > 0 {
		blockPublicAccessSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, blockPublicAccess)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing block public access findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: blockPublicAccess,
			Title:    "Block Public Access",
			Summary:  blockPublicAccessSummary.Summary,
			Remedy:   blockPublicAccessSummary.Remedies,
			Commands: blockPublicAccessSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Block Public Access Findings", zap.Any("Block Public Access", blockPublicAccessSummary))
	}

	if len(lifecyclePolicy) > 0 {
		lifecyclePolicySummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, lifecyclePolicy)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing lifecycle policy findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: lifecyclePolicy,
			Title:    "Lifecycle Policy",
			Summary:  lifecyclePolicySummary.Summary,
			Remedy:   lifecyclePolicySummary.Remedies,
			Commands: lifecyclePolicySummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Lifecycle Policy Findings", zap.Any("Lifecycle Policy", lifecyclePolicySummary))
	}

	if len(mfaDelete) > 0 {
		mfaDeleteSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, mfaDelete)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing MFA delete findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: mfaDelete,
			Title:    "MFA Delete",
			Summary:  mfaDeleteSummary.Summary,
			Remedy:   mfaDeleteSummary.Remedies,
			Commands: mfaDeleteSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("MFA Delete Findings", zap.Any("MFA Delete", mfaDeleteSummary))
	}

	span.SetAttributes(attribute.Int("scan_items_count", len(scanItems)))

	return scanItems, nil
}
