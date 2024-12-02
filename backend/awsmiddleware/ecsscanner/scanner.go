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

	scanner.logger.Logger(ctx).Info("Starting ECS scan",
		zap.String("region", region))

	var allFindings []string

	clusters, err := scanner.ListClusters(ctx)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list ECS clusters",
			zap.String("region", region),
			zap.Error(err))
		return nil, nil, err
	}

	scanner.logger.Logger(ctx).Info("Retrieved ECS clusters",
		zap.Int("cluster_count", len(clusters)))

	// Categorize findings by type
	var iamRoleFindings, networkModeFindings, loggingConfigFindings, sensitiveEnvVarFindings, resourceLimitsFindings []string

	for _, clusterArn := range clusters {
		scanner.logger.Logger(ctx).Debug("Processing cluster",
			zap.String("cluster_arn", *clusterArn))

		// List task definitions for each cluster
		taskDefs, err := scanner.ECSClient.ListTaskDefinitions(&ecs.ListTaskDefinitionsInput{})
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to list task definitions",
				zap.String("cluster_arn", *clusterArn),
				zap.Error(err))
			allFindings = append(allFindings, fmt.Sprintf("Error listing task definitions for cluster %s: %v", *clusterArn, err))
			continue
		}

		scanner.logger.Logger(ctx).Info("Retrieved task definitions",
			zap.String("cluster_arn", *clusterArn),
			zap.Int("task_definition_count", len(taskDefs.TaskDefinitionArns)))

		for _, taskDefArn := range taskDefs.TaskDefinitionArns {
			scanner.logger.Logger(ctx).Debug("Processing task definition",
				zap.String("task_definition_arn", *taskDefArn))

			// Check IAM roles
			iamFindings, err := scanner.CheckTaskDefinitionIAMRoles(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				scanner.logger.Logger(ctx).Error("Failed to check IAM roles",
					zap.String("task_definition_arn", *taskDefArn),
					zap.Error(err))
				allFindings = append(allFindings, fmt.Sprintf("Error checking IAM roles for task definition %s: %v", *taskDefArn, err))
			} else {
				iamRoleFindings = append(iamRoleFindings, iamFindings...)
			}

			// Check network mode
			networkFindings, err := scanner.CheckTaskNetworkMode(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				scanner.logger.Logger(ctx).Error("Failed to check network mode",
					zap.String("task_definition_arn", *taskDefArn),
					zap.Error(err))
				allFindings = append(allFindings, fmt.Sprintf("Error checking network mode for task definition %s: %v", *taskDefArn, err))
			} else {
				networkModeFindings = append(networkModeFindings, networkFindings...)
			}

			// Check logging configuration
			logFindings, err := scanner.CheckLoggingConfiguration(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				scanner.logger.Logger(ctx).Error("Failed to check logging configuration",
					zap.String("task_definition_arn", *taskDefArn),
					zap.Error(err))
				allFindings = append(allFindings, fmt.Sprintf("Error checking logging configuration for task definition %s: %v", *taskDefArn, err))
			} else {
				loggingConfigFindings = append(loggingConfigFindings, logFindings...)
			}

			// Check for sensitive environment variables
			sensitiveFindings, err := scanner.CheckSensitiveEnvironmentVariables(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				scanner.logger.Logger(ctx).Error("Failed to check sensitive environment variables",
					zap.String("task_definition_arn", *taskDefArn),
					zap.Error(err))
				allFindings = append(allFindings, fmt.Sprintf("Error checking sensitive environment variables for task definition %s: %v", *taskDefArn, err))
			} else {
				sensitiveEnvVarFindings = append(sensitiveEnvVarFindings, sensitiveFindings...)
			}

			// Check resource limits
			resourceFindings, err := scanner.CheckResourceLimits(ctx, *taskDefArn)
			if err != nil {
				span.RecordError(err)
				scanner.logger.Logger(ctx).Error("Failed to check resource limits",
					zap.String("task_definition_arn", *taskDefArn),
					zap.Error(err))
				allFindings = append(allFindings, fmt.Sprintf("Error checking resource limits for task definition %s: %v", *taskDefArn, err))
			} else {
				resourceLimitsFindings = append(resourceLimitsFindings, resourceFindings...)
			}
		}
	}

	scanner.logger.Logger(ctx).Info("Completed scanning task definitions",
		zap.Int("iam_findings", len(iamRoleFindings)),
		zap.Int("network_findings", len(networkModeFindings)),
		zap.Int("logging_findings", len(loggingConfigFindings)),
		zap.Int("sensitive_env_findings", len(sensitiveEnvVarFindings)),
		zap.Int("resource_limit_findings", len(resourceLimitsFindings)))

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.ProcessAndSummarizeFindings(
		ctx,
		"ecs",
		region,
		scanner.accountId,
		iamRoleFindings,
		networkModeFindings,
		loggingConfigFindings,
		sensitiveEnvVarFindings,
		resourceLimitsFindings,
	)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to process and summarize findings",
			zap.Error(err))
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, iamRoleFindings...)
	allFindings = append(allFindings, networkModeFindings...)
	allFindings = append(allFindings, loggingConfigFindings...)
	allFindings = append(allFindings, sensitiveEnvVarFindings...)
	allFindings = append(allFindings, resourceLimitsFindings...)

	scanner.logger.Logger(ctx).Info("Completed ECS scan",
		zap.String("region", region),
		zap.Int("total_findings", len(allFindings)),
		zap.Int("scan_items", len(scanItems)))

	return allFindings, scanItems, nil
}

func (scanner *ECSScanner) ProcessAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
	accountId string,
	iamRoles []string,
	networkModes []string,
	loggingConfigs []string,
	sensitiveEnvVars []string,
	resourceLimits []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("ecsscanner/ProcessAndSummarizeFindings")
	ctx, span := tracer.Start(ctx, "ProcessAndSummarizeFindings")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting to process and summarize findings",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", accountId))

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Define prompts for each type of finding and call the LLM
	if len(iamRoles) > 0 {
		scanner.logger.Logger(ctx).Debug("Processing IAM role findings",
			zap.Int("finding_count", len(iamRoles)))

		iamRoleSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, iamRoles)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize IAM role findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing IAM roles: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: iamRoles,
			Title:    "IAM Roles",
			Summary:  iamRoleSummary.Summary,
			Remedy:   iamRoleSummary.Remedies,
			Commands: iamRoleSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed IAM role findings",
			zap.Any("summary", iamRoleSummary))
	}

	if len(networkModes) > 0 {
		scanner.logger.Logger(ctx).Debug("Processing network mode findings",
			zap.Int("finding_count", len(networkModes)))

		networkModeSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, networkModes)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize network mode findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing network modes: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: networkModes,
			Title:    "Network Modes",
			Summary:  networkModeSummary.Summary,
			Remedy:   networkModeSummary.Remedies,
			Commands: networkModeSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed network mode findings",
			zap.Any("summary", networkModeSummary))
	}

	if len(loggingConfigs) > 0 {
		scanner.logger.Logger(ctx).Debug("Processing logging configuration findings",
			zap.Int("finding_count", len(loggingConfigs)))

		loggingSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, loggingConfigs)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize logging configuration findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing logging configuration: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: loggingConfigs,
			Title:    "Logging Configurations",
			Summary:  loggingSummary.Summary,
			Remedy:   loggingSummary.Remedies,
			Commands: loggingSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed logging configuration findings",
			zap.Any("summary", loggingSummary))
	}

	if len(sensitiveEnvVars) > 0 {
		scanner.logger.Logger(ctx).Debug("Processing sensitive environment variable findings",
			zap.Int("finding_count", len(sensitiveEnvVars)))

		sensitiveEnvSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, sensitiveEnvVars)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize sensitive environment variable findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing sensitive environment variables: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: sensitiveEnvVars,
			Title:    "Sensitive Environment Variables",
			Summary:  sensitiveEnvSummary.Summary,
			Remedy:   sensitiveEnvSummary.Remedies,
			Commands: sensitiveEnvSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed sensitive environment variable findings",
			zap.Any("summary", sensitiveEnvSummary))
	}

	if len(resourceLimits) > 0 {
		scanner.logger.Logger(ctx).Debug("Processing resource limit findings",
			zap.Int("finding_count", len(resourceLimits)))

		resourceLimitsSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, resourceLimits)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to summarize resource limit findings",
				zap.Error(err))
			return nil, fmt.Errorf("error summarizing resource limits: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: resourceLimits,
			Title:    "Resource Limits",
			Summary:  resourceLimitsSummary.Summary,
			Remedy:   resourceLimitsSummary.Remedies,
			Commands: resourceLimitsSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed resource limit findings",
			zap.Any("summary", resourceLimitsSummary))
	}

	scanner.logger.Logger(ctx).Info("Completed processing and summarizing findings",
		zap.Int("total_scan_items", len(scanItems)))

	return scanItems, nil
}
