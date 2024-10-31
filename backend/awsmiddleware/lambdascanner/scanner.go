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

	var allFindings []string

	functions, err := scanner.ListFunctions(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	span.SetAttributes(attribute.Int("function_count", len(functions)))

	// Categorize findings by type
	var iamRoleFindings, envVariableFindings, cloudWatchLogFindings, timeoutFindings, vpcFindings []string

	for _, function := range functions {
		allFindings = append(allFindings, fmt.Sprintf("Found Lambda Function: %s", aws.StringValue(function.FunctionName)))

		// Check IAM role attached to Lambda function
		iamFindings, err := scanner.CheckFunctionIAMRole(ctx, function)
		if err != nil {
			span.RecordError(err)
			iamRoleFindings = append(iamRoleFindings, fmt.Sprintf("Error checking IAM role for function %s: %v", aws.StringValue(function.FunctionName), err))
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
			cloudWatchLogFindings = append(cloudWatchLogFindings, fmt.Sprintf("Error checking CloudWatch logs for function %s: %v", aws.StringValue(function.FunctionName), err))
		} else {
			cloudWatchLogFindings = append(cloudWatchLogFindings, logFindings...)
		}

		// Check Lambda function timeout configuration
		timeoutConfigFindings := scanner.CheckFunctionTimeout(ctx, function)
		timeoutFindings = append(timeoutFindings, timeoutConfigFindings...)

		// Check if Lambda function is configured in a VPC
		vpcConfigFindings := scanner.CheckVpcConfiguration(ctx, function)
		vpcFindings = append(vpcFindings, vpcConfigFindings...)
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.processAndSummarizeFindings(
		ctx,
		"lambda",
		region,
		iamRoleFindings,
		envVariableFindings,
		cloudWatchLogFindings,
		timeoutFindings,
		vpcFindings,
	)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, iamRoleFindings...)
	allFindings = append(allFindings, envVariableFindings...)
	allFindings = append(allFindings, cloudWatchLogFindings...)
	allFindings = append(allFindings, timeoutFindings...)
	allFindings = append(allFindings, vpcFindings...)

	span.SetAttributes(attribute.Int("total_findings", len(allFindings)))

	return allFindings, scanItems, nil
}

func (scanner *LambdaScanner) processAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
	iamRoleFindings []string,
	envVariableFindings []string,
	cloudWatchLogFindings []string,
	timeoutFindings []string,
	vpcFindings []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("lambdascanner/processAndSummarizeFindings")
	ctx, span := tracer.Start(ctx, "processAndSummarizeFindings")
	defer span.End()

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Summarize IAM Role findings
	if len(iamRoleFindings) > 0 {
		iamRoleSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, iamRoleFindings)
		if err != nil {
			span.RecordError(err)
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
		envVarSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, envVariableFindings)
		if err != nil {
			span.RecordError(err)
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
		cloudWatchSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, cloudWatchLogFindings)
		if err != nil {
			span.RecordError(err)
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
		timeoutSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, timeoutFindings)
		if err != nil {
			span.RecordError(err)
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
		vpcSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, vpcFindings)
		if err != nil {
			span.RecordError(err)
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

	span.SetAttributes(attribute.Int("scan_items_count", len(scanItems)))

	return scanItems, nil
}
