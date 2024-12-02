package ecsscanner

import (
	"context"
	"fmt"
	"strings"

	"guarddev/logger"
	"guarddev/modelapi/geminiapi"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"go.uber.org/zap"
)

type ECSScanner struct {
	ECSClient *ecs.ECS
	iamClient *iam.IAM
	logger    *logger.LogMiddleware
	Gemini    *geminiapi.Gemini
	accountId string
}

// Initialize an ECS scanner with assumed credentials
func NewECSScanner(accessKey, secretKey, sessionToken, region string, accountId string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *ECSScanner {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			sessionToken,
		),
	}))
	return &ECSScanner{
		ECSClient: ecs.New(sess),
		iamClient: iam.New(sess),
		logger:    logger,
		Gemini:    gemini,
		accountId: accountId,
	}
}

// List ECS clusters
func (scanner *ECSScanner) ListClusters(ctx context.Context) ([]*string, error) {
	tracer := otel.Tracer("ecsscanner/ListClusters")
	ctx, span := tracer.Start(ctx, "ListClusters")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting ECS cluster listing")

	result, err := scanner.ECSClient.ListClusters(nil)
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list ECS clusters",
			zap.Error(err))
		return nil, fmt.Errorf("error listing ECS clusters: %v", err)
	}

	span.SetAttributes(attribute.Int("cluster_count", len(result.ClusterArns)))
	scanner.logger.Logger(ctx).Info("Successfully listed ECS clusters",
		zap.Int("cluster_count", len(result.ClusterArns)))

	return result.ClusterArns, nil
}

// Check IAM roles for ECS task definitions
func (scanner *ECSScanner) CheckTaskDefinitionIAMRoles(ctx context.Context, taskDefArn string) ([]string, error) {
	tracer := otel.Tracer("ecsscanner/CheckTaskDefinitionIAMRoles")
	ctx, span := tracer.Start(ctx, "CheckTaskDefinitionIAMRoles")
	defer span.End()

	scanner.logger.Logger(ctx).Debug("Checking IAM roles for task definition",
		zap.String("task_definition_arn", taskDefArn))

	var findings []string

	taskDef, err := scanner.ECSClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefArn),
	})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to describe task definition",
			zap.String("task_definition_arn", taskDefArn),
			zap.Error(err))
		return nil, fmt.Errorf("error describing task definition %s: %v", taskDefArn, err)
	}

	if taskDef.TaskDefinition.ExecutionRoleArn != nil {
		roleArn := aws.StringValue(taskDef.TaskDefinition.ExecutionRoleArn)
		arnParts := strings.Split(roleArn, "/")
		if len(arnParts) < 2 {
			scanner.logger.Logger(ctx).Error("Invalid ARN format for IAM role",
				zap.String("role_arn", roleArn))
			return nil, fmt.Errorf("invalid ARN format for IAM role: %s", roleArn)
		}
		roleName := arnParts[len(arnParts)-1]

		_, err := scanner.iamClient.GetRole(&iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to get IAM role",
				zap.String("role_name", roleName),
				zap.Error(err))
			return nil, fmt.Errorf("error getting IAM role %s: %v", roleName, err)
		}

		findings = append(findings, fmt.Sprintf("Found IAM role for task definition %s: %s", taskDefArn, roleName))
		scanner.logger.Logger(ctx).Info("Found IAM role for task definition",
			zap.String("task_definition_arn", taskDefArn),
			zap.String("role_name", roleName))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: No IAM execution role found for task definition %s", taskDefArn))
		scanner.logger.Logger(ctx).Warn("No IAM execution role found for task definition",
			zap.String("task_definition_arn", taskDefArn))
	}

	return findings, nil
}

// Check network mode for ECS tasks
func (scanner *ECSScanner) CheckTaskNetworkMode(ctx context.Context, taskDefArn string) ([]string, error) {
	tracer := otel.Tracer("ecsscanner/CheckTaskNetworkMode")
	ctx, span := tracer.Start(ctx, "CheckTaskNetworkMode")
	defer span.End()

	scanner.logger.Logger(ctx).Debug("Checking network mode for task definition",
		zap.String("task_definition_arn", taskDefArn))

	var findings []string
	taskDef, err := scanner.ECSClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefArn),
	})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to describe task definition",
			zap.String("task_definition_arn", taskDefArn),
			zap.Error(err))
		return nil, fmt.Errorf("error describing task definition %s: %v", taskDefArn, err)
	}

	networkMode := aws.StringValue(taskDef.TaskDefinition.NetworkMode)
	if networkMode == "awsvpc" {
		findings = append(findings, fmt.Sprintf("Task definition %s is using network mode: %s", taskDefArn, networkMode))
		scanner.logger.Logger(ctx).Info("Task definition using awsvpc network mode",
			zap.String("task_definition_arn", taskDefArn),
			zap.String("network_mode", networkMode))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: Task definition %s is not using the 'awsvpc' network mode. Current mode: %s", taskDefArn, networkMode))
		scanner.logger.Logger(ctx).Warn("Task definition not using awsvpc network mode",
			zap.String("task_definition_arn", taskDefArn),
			zap.String("network_mode", networkMode))
	}

	return findings, nil
}

// Check if logging is enabled for ECS task definitions
func (scanner *ECSScanner) CheckLoggingConfiguration(ctx context.Context, taskDefArn string) ([]string, error) {
	tracer := otel.Tracer("ecsscanner/CheckLoggingConfiguration")
	ctx, span := tracer.Start(ctx, "CheckLoggingConfiguration")
	defer span.End()

	scanner.logger.Logger(ctx).Debug("Checking logging configuration for task definition",
		zap.String("task_definition_arn", taskDefArn))

	var findings []string
	taskDef, err := scanner.ECSClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefArn),
	})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to describe task definition",
			zap.String("task_definition_arn", taskDefArn),
			zap.Error(err))
		return nil, fmt.Errorf("error describing task definition %s: %v", taskDefArn, err)
	}

	for _, containerDef := range taskDef.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(containerDef.Name)
		if containerDef.LogConfiguration == nil {
			findings = append(findings, fmt.Sprintf("WARNING: No logging configuration found for container %s in task definition %s", containerName, taskDefArn))
			scanner.logger.Logger(ctx).Warn("No logging configuration found for container",
				zap.String("task_definition_arn", taskDefArn),
				zap.String("container_name", containerName))
		} else {
			findings = append(findings, fmt.Sprintf("Logging is configured for container %s in task definition %s", containerName, taskDefArn))
			scanner.logger.Logger(ctx).Info("Logging is configured for container",
				zap.String("task_definition_arn", taskDefArn),
				zap.String("container_name", containerName))
		}
	}

	return findings, nil
}

// Check for sensitive environment variables in ECS task definitions
func (scanner *ECSScanner) CheckSensitiveEnvironmentVariables(ctx context.Context, taskDefArn string) ([]string, error) {
	tracer := otel.Tracer("ecsscanner/CheckSensitiveEnvironmentVariables")
	ctx, span := tracer.Start(ctx, "CheckSensitiveEnvironmentVariables")
	defer span.End()

	scanner.logger.Logger(ctx).Debug("Checking sensitive environment variables for task definition",
		zap.String("task_definition_arn", taskDefArn))

	var findings []string
	taskDef, err := scanner.ECSClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefArn),
	})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to describe task definition",
			zap.String("task_definition_arn", taskDefArn),
			zap.Error(err))
		return nil, fmt.Errorf("error describing task definition %s: %v", taskDefArn, err)
	}

	sensitiveKeywords := []string{"password", "secret", "key", "token"}
	for _, containerDef := range taskDef.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(containerDef.Name)
		for _, envVar := range containerDef.Environment {
			envVarName := aws.StringValue(envVar.Name)
			for _, keyword := range sensitiveKeywords {
				if strings.Contains(strings.ToLower(envVarName), keyword) {
					findings = append(findings, fmt.Sprintf("WARNING: Sensitive information found in environment variable %s for container %s in task definition %s", envVarName, containerName, taskDefArn))
					scanner.logger.Logger(ctx).Warn("Sensitive information found in environment variable",
						zap.String("task_definition_arn", taskDefArn),
						zap.String("container_name", containerName),
						zap.String("env_var_name", envVarName))
				}
			}
		}
	}

	return findings, nil
}

// Check resource limits for ECS task definitions
func (scanner *ECSScanner) CheckResourceLimits(ctx context.Context, taskDefArn string) ([]string, error) {
	tracer := otel.Tracer("ecsscanner/CheckResourceLimits")
	ctx, span := tracer.Start(ctx, "CheckResourceLimits")
	defer span.End()

	scanner.logger.Logger(ctx).Debug("Checking resource limits for task definition",
		zap.String("task_definition_arn", taskDefArn))

	var findings []string
	taskDef, err := scanner.ECSClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefArn),
	})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to describe task definition",
			zap.String("task_definition_arn", taskDefArn),
			zap.Error(err))
		return nil, fmt.Errorf("error describing task definition %s: %v", taskDefArn, err)
	}

	for _, containerDef := range taskDef.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(containerDef.Name)
		if containerDef.Cpu == nil || containerDef.Memory == nil {
			findings = append(findings, fmt.Sprintf("WARNING: Resource limits (CPU/Memory) not set for container %s in task definition %s", containerName, taskDefArn))
			scanner.logger.Logger(ctx).Warn("Resource limits not set for container",
				zap.String("task_definition_arn", taskDefArn),
				zap.String("container_name", containerName))
		} else {
			findings = append(findings, fmt.Sprintf("Resource limits are set for container %s in task definition %s", containerName, taskDefArn))
			scanner.logger.Logger(ctx).Info("Resource limits are set for container",
				zap.String("task_definition_arn", taskDefArn),
				zap.String("container_name", containerName),
				zap.Int64("cpu", aws.Int64Value(containerDef.Cpu)),
				zap.Int64("memory", aws.Int64Value(containerDef.Memory)))
		}
	}

	return findings, nil
}
