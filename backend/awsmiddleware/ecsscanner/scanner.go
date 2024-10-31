package ecsscanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"github.com/aws/aws-sdk-go/service/ecs"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func (scanner *ECSScanner) ScanECS(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("ecsscanner/ScanECS")
	ctx, span := tracer.Start(ctx, "ScanECS")
	defer span.End()

	var allFindings []string

	clusters, err := scanner.ListClusters(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	// Categorize findings by type
	var iamRoleFindings, networkModeFindings, loggingConfigFindings, sensitiveEnvVarFindings, resourceLimitsFindings []string

	for _, clusterArn := range clusters {
		// List task definitions for each cluster
		taskDefs, err := scanner.ECSClient.ListTaskDefinitions(&ecs.ListTaskDefinitionsInput{})
		if err != nil {
			span.RecordError(err)
			allFindings = append(allFindings, fmt.Sprintf("Error listing task definitions for cluster %s: %v", *clusterArn, err))
			continue
		}

		for _, taskDefArn := range taskDefs.TaskDefinitionArns {
			// Check IAM roles
			iamFindings, err := scanner.CheckTaskDefinitionIAMRoles(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				allFindings = append(allFindings, fmt.Sprintf("Error checking IAM roles for task definition %s: %v", *taskDefArn, err))
			} else {
				iamRoleFindings = append(iamRoleFindings, iamFindings...)
			}

			// Check network mode
			networkFindings, err := scanner.CheckTaskNetworkMode(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				allFindings = append(allFindings, fmt.Sprintf("Error checking network mode for task definition %s: %v", *taskDefArn, err))
			} else {
				networkModeFindings = append(networkModeFindings, networkFindings...)
			}

			// Check logging configuration
			logFindings, err := scanner.CheckLoggingConfiguration(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				allFindings = append(allFindings, fmt.Sprintf("Error checking logging configuration for task definition %s: %v", *taskDefArn, err))
			} else {
				loggingConfigFindings = append(loggingConfigFindings, logFindings...)
			}

			// Check for sensitive environment variables
			sensitiveFindings, err := scanner.CheckSensitiveEnvironmentVariables(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				allFindings = append(allFindings, fmt.Sprintf("Error checking sensitive environment variables for task definition %s: %v", *taskDefArn, err))
			} else {
				sensitiveEnvVarFindings = append(sensitiveEnvVarFindings, sensitiveFindings...)
			}

			// Check resource limits
			resourceFindings, err := scanner.CheckResourceLimits(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				allFindings = append(allFindings, fmt.Sprintf("Error checking resource limits for task definition %s: %v", *taskDefArn, err))
			} else {
				resourceLimitsFindings = append(resourceLimitsFindings, resourceFindings...)
			}
		}
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.ProcessAndSummarizeFindings(
		ctx,
		"ecs",
		region,
		iamRoleFindings,
		networkModeFindings,
		loggingConfigFindings,
		sensitiveEnvVarFindings,
		resourceLimitsFindings,
	)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, iamRoleFindings...)
	allFindings = append(allFindings, networkModeFindings...)
	allFindings = append(allFindings, loggingConfigFindings...)
	allFindings = append(allFindings, sensitiveEnvVarFindings...)
	allFindings = append(allFindings, resourceLimitsFindings...)

	return allFindings, scanItems, nil
}

func (scanner *ECSScanner) ProcessAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
	iamRoles []string,
	networkModes []string,
	loggingConfigs []string,
	sensitiveEnvVars []string,
	resourceLimits []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("ecsscanner/ProcessAndSummarizeFindings")
	ctx, span := tracer.Start(ctx, "ProcessAndSummarizeFindings")
	defer span.End()

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Define prompts for each type of finding and call the LLM
	if len(iamRoles) > 0 {
		iamRoleSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, iamRoles)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing IAM roles: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: iamRoles,
			Title:    "IAM Roles",
			Summary:  iamRoleSummary.Summary,
			Remedy:   iamRoleSummary.Remedies,
			Commands: iamRoleSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("IAM Roles", zap.Any("IAM Roles", iamRoleSummary))
	}

	if len(networkModes) > 0 {
		networkModeSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, networkModes)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing network modes: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: networkModes,
			Title:    "Network Modes",
			Summary:  networkModeSummary.Summary,
			Remedy:   networkModeSummary.Remedies,
			Commands: networkModeSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Network Modes", zap.Any("Network Modes", networkModeSummary))
	}

	if len(loggingConfigs) > 0 {
		loggingSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, loggingConfigs)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing logging configuration: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: loggingConfigs,
			Title:    "Logging Configurations",
			Summary:  loggingSummary.Summary,
			Remedy:   loggingSummary.Remedies,
			Commands: loggingSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Logging Configurations", zap.Any("Logging Configurations", loggingSummary))
	}

	if len(sensitiveEnvVars) > 0 {
		sensitiveEnvSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, sensitiveEnvVars)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing sensitive environment variables: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: sensitiveEnvVars,
			Title:    "Sensitive Environment Variables",
			Summary:  sensitiveEnvSummary.Summary,
			Remedy:   sensitiveEnvSummary.Remedies,
			Commands: sensitiveEnvSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Sensitive Environment Variables", zap.Any("Sensitive Environment Variables", sensitiveEnvSummary))
	}

	if len(resourceLimits) > 0 {
		resourceLimitsSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, resourceLimits)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing resource limits: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: resourceLimits,
			Title:    "Resource Limits",
			Summary:  resourceLimitsSummary.Summary,
			Remedy:   resourceLimitsSummary.Remedies,
			Commands: resourceLimitsSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Resource Limits", zap.Any("Resource Limits", resourceLimitsSummary))
	}

	return scanItems, nil
}
