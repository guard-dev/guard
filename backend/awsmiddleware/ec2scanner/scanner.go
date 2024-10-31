package ec2scanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"go.opentelemetry.io/otel"
)

func (scanner *EC2Scanner) ScanEC2(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("ec2scanner/ScanEC2")
	ctx, span := tracer.Start(ctx, "ScanEC2")
	defer span.End()

	var allFindings []string

	instances, err := scanner.ListInstances(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	// Categorize findings by type
	var publicIPFindings, volumeEncryptionFindings, securityGroupFindings, iamRoleFindings, cloudWatchMonitoringFindings, metadataFindings, elasticIPFindings []string

	for _, instance := range instances {
		// Public IP
		hasPublicIP, publicIPFinding := scanner.HasPublicIP(ctx, instance)
		if hasPublicIP {
			publicIPFindings = append(publicIPFindings, publicIPFinding)
		}

		// Volume encryption
		volumesEncrypted, volumeFindings := scanner.AreVolumesEncrypted(ctx, instance)
		if !volumesEncrypted {
			volumeEncryptionFindings = append(volumeEncryptionFindings, volumeFindings...)
		}

		// Security group settings
		securityGroupFinding := scanner.CheckSecurityGroups(ctx, instance)
		securityGroupFindings = append(securityGroupFindings, securityGroupFinding...)

		// IAM role
		iamRoleFinding := scanner.CheckInstanceIAMRole(ctx, instance)
		iamRoleFindings = append(iamRoleFindings, iamRoleFinding...)

		// CloudWatch monitoring
		cloudWatchFinding := scanner.CheckCloudWatchMonitoring(ctx, instance)
		cloudWatchMonitoringFindings = append(cloudWatchMonitoringFindings, cloudWatchFinding...)

		// Metadata version
		metadataFinding := scanner.CheckMetadataVersion(ctx, instance)
		metadataFindings = append(metadataFindings, metadataFinding...)
	}

	// Check for unassociated Elastic IPs
	elasticIPFinding, err := scanner.CheckUnassociatedElasticIPs(ctx)
	if err != nil {
		span.RecordError(err)
		elasticIPFindings = append(elasticIPFindings, fmt.Sprintf("Error checking unassociated Elastic IPs: %v", err))
	} else {
		elasticIPFindings = append(elasticIPFindings, elasticIPFinding...)
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.processAndSummarizeFindings(
		ctx,
		region,
		"ec2",
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
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, publicIPFindings...)
	allFindings = append(allFindings, volumeEncryptionFindings...)
	allFindings = append(allFindings, securityGroupFindings...)
	allFindings = append(allFindings, iamRoleFindings...)
	allFindings = append(allFindings, cloudWatchMonitoringFindings...)
	allFindings = append(allFindings, metadataFindings...)
	allFindings = append(allFindings, elasticIPFindings...)

	return allFindings, scanItems, nil
}

func (scanner *EC2Scanner) processAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
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

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	if len(publicIPFindings) > 0 {
		publicIPSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, publicIPFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing public IP findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: publicIPFindings,
			Title:    "Public IP Findings",
			Summary:  publicIPSummary.Summary,
			Remedy:   publicIPSummary.Remedies,
			Commands: publicIPSummary.Commands,
		})
	}

	if len(volumeEncryptionFindings) > 0 {
		volumeEncryptionSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, volumeEncryptionFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing volume encryption findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: volumeEncryptionFindings,
			Title:    "Volume Encryption Findings",
			Summary:  volumeEncryptionSummary.Summary,
			Remedy:   volumeEncryptionSummary.Remedies,
			Commands: volumeEncryptionSummary.Commands,
		})
	}

	if len(securityGroupFindings) > 0 {
		securityGroupSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, securityGroupFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing security group findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: securityGroupFindings,
			Title:    "Security Group Findings",
			Summary:  securityGroupSummary.Summary,
			Remedy:   securityGroupSummary.Remedies,
			Commands: securityGroupSummary.Commands,
		})
	}

	if len(iamRoleFindings) > 0 {
		iamRoleSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, iamRoleFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing IAM role findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: iamRoleFindings,
			Title:    "IAM Role Findings",
			Summary:  iamRoleSummary.Summary,
			Remedy:   iamRoleSummary.Remedies,
			Commands: iamRoleSummary.Commands,
		})
	}

	if len(cloudWatchMonitoringFindings) > 0 {
		cloudWatchMonitoringSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, cloudWatchMonitoringFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing CloudWatch monitoring findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: cloudWatchMonitoringFindings,
			Title:    "CloudWatch Monitoring Findings",
			Summary:  cloudWatchMonitoringSummary.Summary,
			Remedy:   cloudWatchMonitoringSummary.Remedies,
			Commands: cloudWatchMonitoringSummary.Commands,
		})
	}

	if len(metadataFindings) > 0 {
		metadataSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, metadataFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing metadata findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: metadataFindings,
			Title:    "Metadata Findings",
			Summary:  metadataSummary.Summary,
			Remedy:   metadataSummary.Remedies,
			Commands: metadataSummary.Commands,
		})
	}

	if len(elasticIPFindings) > 0 {
		elasticIPSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, elasticIPFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing Elastic IP findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: elasticIPFindings,
			Title:    "Elastic IP Findings",
			Summary:  elasticIPSummary.Summary,
			Remedy:   elasticIPSummary.Remedies,
			Commands: elasticIPSummary.Commands,
		})
	}

	return scanItems, nil
}
