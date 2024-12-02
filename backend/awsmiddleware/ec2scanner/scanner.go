package ec2scanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"
	"strings"

	"github.com/aws/aws-sdk-go/aws"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (scanner *EC2Scanner) ScanEC2(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("ec2scanner/ScanEC2")
	ctx, span := tracer.Start(ctx, "ScanEC2")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting EC2 scan",
		zap.String("region", region),
		zap.String("account_id", scanner.accountId))

	var allFindings []string

	instances, err := scanner.ListInstances(ctx)
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Failed to list EC2 instances: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg)
		return nil, nil, fmt.Errorf(errorMsg)
	}

	scanner.logger.Logger(ctx).Info("Found EC2 instances",
		zap.Int("instance_count", len(instances)))

	// Categorize findings by type
	var publicIPFindings, volumeEncryptionFindings, securityGroupFindings, iamRoleFindings, cloudWatchMonitoringFindings, metadataFindings, elasticIPFindings []string

	for _, instance := range instances {
		instanceID := aws.StringValue(instance.InstanceId)
		scanner.logger.Logger(ctx).Info("Scanning instance",
			zap.String("instance_id", instanceID))

		// Public IP
		hasPublicIP, publicIPFinding := scanner.HasPublicIP(ctx, instance)
		if hasPublicIP {
			scanner.logger.Logger(ctx).Warn("Instance has public IP",
				zap.String("instance_id", instanceID),
				zap.String("finding", publicIPFinding))
			publicIPFindings = append(publicIPFindings, publicIPFinding)
		}

		// Volume encryption
		volumesEncrypted, volumeFindings := scanner.AreVolumesEncrypted(ctx, instance)
		if !volumesEncrypted {
			for _, finding := range volumeFindings {
				scanner.logger.Logger(ctx).Warn("Volume encryption issue",
					zap.String("instance_id", instanceID),
					zap.String("finding", finding))
			}
			volumeEncryptionFindings = append(volumeEncryptionFindings, volumeFindings...)
		}

		// Security group settings
		securityGroupFinding := scanner.CheckSecurityGroups(ctx, instance)
		if len(securityGroupFinding) > 0 {
			for _, finding := range securityGroupFinding {
				scanner.logger.Logger(ctx).Warn("Security group issue",
					zap.String("instance_id", instanceID),
					zap.String("finding", finding))
			}
			securityGroupFindings = append(securityGroupFindings, securityGroupFinding...)
		}

		// IAM role
		iamRoleFinding := scanner.CheckInstanceIAMRole(ctx, instance)
		if len(iamRoleFinding) > 0 {
			for _, finding := range iamRoleFinding {
				if strings.Contains(finding, "WARNING") {
					scanner.logger.Logger(ctx).Warn("IAM role issue",
						zap.String("instance_id", instanceID),
						zap.String("finding", finding))
				} else {
					scanner.logger.Logger(ctx).Info("IAM role check",
						zap.String("instance_id", instanceID),
						zap.String("finding", finding))
				}
			}
			iamRoleFindings = append(iamRoleFindings, iamRoleFinding...)
		}

		// CloudWatch monitoring
		cloudWatchFinding := scanner.CheckCloudWatchMonitoring(ctx, instance)
		if len(cloudWatchFinding) > 0 {
			for _, finding := range cloudWatchFinding {
				if strings.Contains(finding, "WARNING") {
					scanner.logger.Logger(ctx).Warn("CloudWatch monitoring issue",
						zap.String("instance_id", instanceID),
						zap.String("finding", finding))
				} else {
					scanner.logger.Logger(ctx).Info("CloudWatch monitoring check",
						zap.String("instance_id", instanceID),
						zap.String("finding", finding))
				}
			}
			cloudWatchMonitoringFindings = append(cloudWatchMonitoringFindings, cloudWatchFinding...)
		}

		// Metadata version
		metadataFinding := scanner.CheckMetadataVersion(ctx, instance)
		if len(metadataFinding) > 0 {
			for _, finding := range metadataFinding {
				if strings.Contains(finding, "WARNING") {
					scanner.logger.Logger(ctx).Warn("Metadata version issue",
						zap.String("instance_id", instanceID),
						zap.String("finding", finding))
				} else {
					scanner.logger.Logger(ctx).Info("Metadata version check",
						zap.String("instance_id", instanceID),
						zap.String("finding", finding))
				}
			}
			metadataFindings = append(metadataFindings, metadataFinding...)
		}
	}

	// Check for unassociated Elastic IPs
	elasticIPFinding, err := scanner.CheckUnassociatedElasticIPs(ctx)
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error checking unassociated Elastic IPs: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg)
		elasticIPFindings = append(elasticIPFindings, errorMsg)
	} else if len(elasticIPFinding) > 0 {
		for _, finding := range elasticIPFinding {
			scanner.logger.Logger(ctx).Warn("Unassociated Elastic IP",
				zap.String("finding", finding))
		}
		elasticIPFindings = append(elasticIPFindings, elasticIPFinding...)
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.processAndSummarizeFindings(
		ctx,
		region,
		"ec2",
		scanner.accountId,
		publicIPFindings,
		volumeEncryptionFindings,
		securityGroupFindings,
		iamRoleFindings,
		cloudWatchMonitoringFindings,
		metadataFindings,
		elasticIPFindings,
	)
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Failed to process and summarize findings: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg)
		return nil, nil, fmt.Errorf(errorMsg)
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, publicIPFindings...)
	allFindings = append(allFindings, volumeEncryptionFindings...)
	allFindings = append(allFindings, securityGroupFindings...)
	allFindings = append(allFindings, iamRoleFindings...)
	allFindings = append(allFindings, cloudWatchMonitoringFindings...)
	allFindings = append(allFindings, metadataFindings...)
	allFindings = append(allFindings, elasticIPFindings...)

	scanner.logger.Logger(ctx).Info("EC2 scan completed",
		zap.Int("total_findings", len(allFindings)),
		zap.Int("scan_items", len(scanItems)))

	span.SetAttributes(
		attribute.Int("total_findings", len(allFindings)),
		attribute.Int("scan_items", len(scanItems)),
	)

	return allFindings, scanItems, nil
}

func (scanner *EC2Scanner) processAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
	accountId string,
	publicIPFindings []string,
	volumeEncryptionFindings []string,
	securityGroupFindings []string,
	iamRoleFindings []string,
	cloudWatchMonitoringFindings []string,
	metadataFindings []string,
	elasticIPFindings []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("ec2scanner/processAndSummarizeFindings")
	ctx, span := tracer.Start(ctx, "processAndSummarizeFindings")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Processing and summarizing EC2 findings",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", accountId))

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	if len(publicIPFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing public IP findings",
			zap.Int("finding_count", len(publicIPFindings)))
		publicIPSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, publicIPFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing public IP findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: publicIPFindings,
			Title:    "Public IP Findings",
			Summary:  publicIPSummary.Summary,
			Remedy:   publicIPSummary.Remedies,
			Commands: publicIPSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Public IP findings processed",
			zap.Any("summary", publicIPSummary))
	}

	if len(volumeEncryptionFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing volume encryption findings",
			zap.Int("finding_count", len(volumeEncryptionFindings)))
		volumeEncryptionSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, volumeEncryptionFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing volume encryption findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: volumeEncryptionFindings,
			Title:    "Volume Encryption Findings",
			Summary:  volumeEncryptionSummary.Summary,
			Remedy:   volumeEncryptionSummary.Remedies,
			Commands: volumeEncryptionSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Volume encryption findings processed",
			zap.Any("summary", volumeEncryptionSummary))
	}

	if len(securityGroupFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing security group findings",
			zap.Int("finding_count", len(securityGroupFindings)))
		securityGroupSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, securityGroupFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing security group findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: securityGroupFindings,
			Title:    "Security Group Configuration",
			Summary:  securityGroupSummary.Summary,
			Remedy:   securityGroupSummary.Remedies,
			Commands: securityGroupSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Security group findings processed",
			zap.Any("summary", securityGroupSummary))
	}

	if len(iamRoleFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing IAM role findings",
			zap.Int("finding_count", len(iamRoleFindings)))
		iamRoleSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, iamRoleFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing IAM role findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: iamRoleFindings,
			Title:    "IAM Role Findings",
			Summary:  iamRoleSummary.Summary,
			Remedy:   iamRoleSummary.Remedies,
			Commands: iamRoleSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("IAM role findings processed",
			zap.Any("summary", iamRoleSummary))
	}

	if len(cloudWatchMonitoringFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing CloudWatch monitoring findings",
			zap.Int("finding_count", len(cloudWatchMonitoringFindings)))
		cloudWatchMonitoringSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, cloudWatchMonitoringFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing CloudWatch monitoring findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: cloudWatchMonitoringFindings,
			Title:    "CloudWatch Monitoring Findings",
			Summary:  cloudWatchMonitoringSummary.Summary,
			Remedy:   cloudWatchMonitoringSummary.Remedies,
			Commands: cloudWatchMonitoringSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("CloudWatch monitoring findings processed",
			zap.Any("summary", cloudWatchMonitoringSummary))
	}

	if len(metadataFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing metadata findings",
			zap.Int("finding_count", len(metadataFindings)))
		metadataSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, metadataFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing metadata findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: metadataFindings,
			Title:    "Metadata Findings",
			Summary:  metadataSummary.Summary,
			Remedy:   metadataSummary.Remedies,
			Commands: metadataSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Metadata findings processed",
			zap.Any("summary", metadataSummary))
	}

	if len(elasticIPFindings) > 0 {
		scanner.logger.Logger(ctx).Info("Summarizing Elastic IP findings",
			zap.Int("finding_count", len(elasticIPFindings)))
		elasticIPSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, elasticIPFindings)
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error summarizing Elastic IP findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg)
			return nil, fmt.Errorf(errorMsg)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: elasticIPFindings,
			Title:    "Elastic IP Findings",
			Summary:  elasticIPSummary.Summary,
			Remedy:   elasticIPSummary.Remedies,
			Commands: elasticIPSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Elastic IP findings processed",
			zap.Any("summary", elasticIPSummary))
	}

	scanner.logger.Logger(ctx).Info("Completed processing all findings",
		zap.Int("total_scan_items", len(scanItems)))

	span.SetAttributes(attribute.Int("scan_items_count", len(scanItems)))
	return scanItems, nil
}
