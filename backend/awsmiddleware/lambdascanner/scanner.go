package lambdascanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"github.com/aws/aws-sdk-go/aws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (scanner *LambdaScanner) ScanLambda(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("lambdascanner/ScanLambda")
	ctx, span := tracer.Start(ctx, "ScanLambda")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting Lambda security scan",
		zap.String("region", region))

	var allFindings []string

	functions, err := scanner.ListFunctions(ctx)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list Lambda functions",
			zap.String("region", region),
			zap.Error(err))
		return nil, nil, err
	}

	span.SetAttributes(attribute.Int("function_count", len(functions)))
	scanner.logger.Logger(ctx).Info("Retrieved Lambda functions",
		zap.String("region", region),
		zap.Int("function_count", len(functions)))

	// Categorize findings by type
	var iamRoleFindings, envVariableFindings, cloudWatchLogFindings, timeoutFindings, vpcFindings []string

	for _, function := range functions {
		funcName := aws.StringValue(function.FunctionName)
		scanner.logger.Logger(ctx).Debug("Scanning Lambda function",
			zap.String("function_name", funcName))

		allFindings = append(allFindings, fmt.Sprintf("Found Lambda Function: %s", funcName))

		// Check IAM role attached to Lambda function
		iamFindings, err := scanner.CheckFunctionIAMRole(ctx, function)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to check IAM role",
				zap.String("function_name", funcName),
				zap.Error(err))
			iamRoleFindings = append(iamRoleFindings, fmt.Sprintf("Error checking IAM role for function %s: %v", funcName, err))
		} else {
			iamRoleFindings = append(iamRoleFindings, iamFindings...)
		}

		// Check for sensitive environment variables in Lambda function
		envFindings := scanner.CheckSensitiveEnvironmentVariables(ctx, function)
		envVariableFindings = append(envVariableFindings, envFindings...)

		// Check if CloudWatch Logs is properly configured
		logFindings, err := scanner.CheckCloudWatchLogs(ctx, function)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to check CloudWatch logs",
				zap.String("function_name", funcName),
				zap.Error(err))
			cloudWatchLogFindings = append(cloudWatchLogFindings, fmt.Sprintf("Error checking CloudWatch logs for function %s: %v", funcName, err))
		} else {
			cloudWatchLogFindings = append(cloudWatchLogFindings, logFindings...)
		}

		// Check Lambda function timeout configuration
		timeoutConfigFindings := scanner.CheckFunctionTimeout(ctx, function)
		timeoutFindings = append(timeoutFindings, timeoutConfigFindings...)

		// Check if Lambda function is configured in a VPC
		vpcConfigFindings := scanner.CheckVpcConfiguration(ctx, function)
		vpcFindings = append(vpcFindings, vpcConfigFindings...)

		scanner.logger.Logger(ctx).Debug("Completed function scan",
			zap.String("function_name", funcName),
			zap.Int("iam_findings", len(iamFindings)),
			zap.Int("env_findings", len(envFindings)),
			zap.Int("log_findings", len(logFindings)),
			zap.Int("timeout_findings", len(timeoutConfigFindings)),
			zap.Int("vpc_findings", len(vpcConfigFindings)))
	}

	// Process and summarize categorized findings using LLM
	scanner.logger.Logger(ctx).Info("Processing and summarizing findings",
		zap.String("region", region),
		zap.Int("iam_findings", len(iamRoleFindings)),
		zap.Int("env_findings", len(envVariableFindings)),
		zap.Int("log_findings", len(cloudWatchLogFindings)),
		zap.Int("timeout_findings", len(timeoutFindings)),
		zap.Int("vpc_findings", len(vpcFindings)))

	scanItems, err := scanner.processAndSummarizeFindings(
		ctx,
		"lambda",
		region,
		scanner.accountId,
		iamRoleFindings,
		envVariableFindings,
		cloudWatchLogFindings,
		timeoutFindings,
		vpcFindings,
	)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to process and summarize findings",
			zap.String("region", region),
			zap.Error(err))
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, iamRoleFindings...)
	allFindings = append(allFindings, envVariableFindings...)
	allFindings = append(allFindings, cloudWatchLogFindings...)
	allFindings = append(allFindings, timeoutFindings...)
	allFindings = append(allFindings, vpcFindings...)

	span.SetAttributes(attribute.Int("total_findings", len(allFindings)))
	scanner.logger.Logger(ctx).Info("Completed Lambda security scan",
		zap.String("region", region),
		zap.Int("total_findings", len(allFindings)),
		zap.Int("scan_items", len(scanItems)))

	return allFindings, scanItems, nil
}

func (scanner *LambdaScanner) processAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
	accountId string,
	iamRoleFindings []string,
	envVariableFindings []string,
	cloudWatchLogFindings []string,
	timeoutFindings []string,
	vpcFindings []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("lambdascanner/processAndSummarizeFindings")
	ctx, span := tracer.Start(ctx, "processAndSummarizeFindings")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting findings summarization",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", accountId))

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Summarize IAM Role findings
	if len(iamRoleFindings) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing IAM role findings",
			zap.Int("finding_count", len(iamRoleFindings)))

		iamRoleSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, iamRoleFindings)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize IAM role findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing IAM role findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: iamRoleFindings,
			Title:    "Lambda IAM Role Findings",
			Summary:  iamRoleSummary.Summary,
			Remedy:   iamRoleSummary.Remedies,
			Commands: iamRoleSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Lambda IAM Role Findings", zap.Any("Summary", iamRoleSummary))
	}

	// Summarize Environment Variable findings
	if len(envVariableFindings) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing environment variable findings",
			zap.Int("finding_count", len(envVariableFindings)))

		envVarSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, envVariableFindings)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize environment variable findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing environment variable findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: envVariableFindings,
			Title:    "Sensitive Environment Variable Findings",
			Summary:  envVarSummary.Summary,
			Remedy:   envVarSummary.Remedies,
			Commands: envVarSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Sensitive Environment Variable Findings", zap.Any("Summary", envVarSummary))
	}

	// Summarize CloudWatch Log findings
	if len(cloudWatchLogFindings) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing CloudWatch log findings",
			zap.Int("finding_count", len(cloudWatchLogFindings)))

		cloudWatchSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, cloudWatchLogFindings)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize CloudWatch log findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing CloudWatch log findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: cloudWatchLogFindings,
			Title:    "CloudWatch Log Findings",
			Summary:  cloudWatchSummary.Summary,
			Remedy:   cloudWatchSummary.Remedies,
			Commands: cloudWatchSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("CloudWatch Log Findings", zap.Any("Summary", cloudWatchSummary))
	}

	// Summarize Timeout Configuration findings
	if len(timeoutFindings) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing timeout configuration findings",
			zap.Int("finding_count", len(timeoutFindings)))

		timeoutSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, timeoutFindings)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize timeout configuration findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing timeout configuration findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: timeoutFindings,
			Title:    "Timeout Configuration Findings",
			Summary:  timeoutSummary.Summary,
			Remedy:   timeoutSummary.Remedies,
			Commands: timeoutSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Timeout Configuration Findings", zap.Any("Summary", timeoutSummary))
	}

	// Summarize VPC Configuration findings
	if len(vpcFindings) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing VPC configuration findings",
			zap.Int("finding_count", len(vpcFindings)))

		vpcSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, vpcFindings)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize VPC configuration findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing VPC configuration findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: vpcFindings,
			Title:    "VPC Configuration Findings",
			Summary:  vpcSummary.Summary,
			Remedy:   vpcSummary.Remedies,
			Commands: vpcSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("VPC Configuration Findings", zap.Any("Summary", vpcSummary))
	}

	scanner.logger.Logger(ctx).Info("Completed findings summarization",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", accountId),
		zap.Int("total_scan_items", len(scanItems)))

	return scanItems, nil
}
