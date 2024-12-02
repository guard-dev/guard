package lambdascanner

import (
	"context"
	"encoding/json"
	"fmt"
	"guarddev/logger"
	"guarddev/modelapi/geminiapi"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type LambdaScanner struct {
	lambdaClient *lambda.Lambda
	iamClient    *iam.IAM
	logger       *logger.LogMiddleware
	Gemini       *geminiapi.Gemini
	accountId    string
}

// Initialize a Lambda scanner with assumed credentials
func NewLambdaScanner(accessKey, secretKey, sessionToken, region string, accountId string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *LambdaScanner {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			sessionToken,
		),
	}))
	return &LambdaScanner{
		lambdaClient: lambda.New(sess),
		iamClient:    iam.New(sess),
		logger:       logger,
		Gemini:       gemini,
		accountId:    accountId,
	}
}

// List all Lambda functions
func (scanner *LambdaScanner) ListFunctions(ctx context.Context) ([]*lambda.FunctionConfiguration, error) {
	tracer := otel.Tracer("lambdascanner/ListFunctions")
	ctx, span := tracer.Start(ctx, "ListFunctions")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting Lambda functions listing")

	result, err := scanner.lambdaClient.ListFunctions(&lambda.ListFunctionsInput{})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list Lambda functions",
			zap.Error(err))
		return nil, fmt.Errorf("error listing Lambda functions: %v", err)
	}

	span.SetAttributes(attribute.Int("function_count", len(result.Functions)))
	scanner.logger.Logger(ctx).Info("Successfully listed Lambda functions",
		zap.Int("function_count", len(result.Functions)))
	return result.Functions, nil
}

// Check if Lambda function has overly permissive IAM role
func (scanner *LambdaScanner) CheckFunctionIAMRole(ctx context.Context, function *lambda.FunctionConfiguration) ([]string, error) {
	tracer := otel.Tracer("lambdascanner/CheckFunctionIAMRole")
	ctx, span := tracer.Start(ctx, "CheckFunctionIAMRole")
	defer span.End()

	funcName := aws.StringValue(function.FunctionName)
	scanner.logger.Logger(ctx).Info("Checking IAM role for Lambda function",
		zap.String("function_name", funcName))

	var findings []string

	roleArn := aws.StringValue(function.Role)
	if roleArn == "" {
		msg := fmt.Sprintf("WARNING: Lambda function %s does not have a valid IAM role assigned.", funcName)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Warn("Lambda function missing IAM role",
			zap.String("function_name", funcName))
		return findings, nil
	}

	roleParts := strings.Split(roleArn, "/")
	if len(roleParts) < 2 {
		msg := fmt.Sprintf("WARNING: Lambda function %s has an invalid IAM role ARN: %s", funcName, roleArn)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Warn("Invalid IAM role ARN",
			zap.String("function_name", funcName),
			zap.String("role_arn", roleArn))
		return findings, nil
	}
	roleName := roleParts[1]

	_, err := scanner.iamClient.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			msg := fmt.Sprintf("WARNING: IAM role %s for Lambda function %s does not exist.", roleName, funcName)
			findings = append(findings, msg)
			scanner.logger.Logger(ctx).Warn("IAM role does not exist",
				zap.String("function_name", funcName),
				zap.String("role_name", roleName))
		} else {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to get IAM role",
				zap.String("function_name", funcName),
				zap.String("role_name", roleName),
				zap.Error(err))
			return nil, fmt.Errorf("error getting IAM role %s: %v", roleName, err)
		}
		return findings, nil
	}

	policies, err := scanner.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: aws.String(roleName)})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list attached role policies",
			zap.String("function_name", funcName),
			zap.String("role_name", roleName),
			zap.Error(err))
		return nil, fmt.Errorf("error listing attached policies for role %s: %v", roleName, err)
	}

	scanner.logger.Logger(ctx).Debug("Analyzing attached policies",
		zap.String("function_name", funcName),
		zap.String("role_name", roleName),
		zap.Int("policy_count", len(policies.AttachedPolicies)))

	for _, policy := range policies.AttachedPolicies {
		policyName := aws.StringValue(policy.PolicyName)
		policyDetails, err := scanner.iamClient.GetPolicy(&iam.GetPolicyInput{PolicyArn: policy.PolicyArn})
		if err != nil {
			msg := fmt.Sprintf("Error retrieving policy %s: %v", policyName, err)
			findings = append(findings, msg)
			scanner.logger.Logger(ctx).Error("Failed to get policy details",
				zap.String("policy_name", policyName),
				zap.Error(err))
			continue
		}

		policyVersion, err := scanner.iamClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetails.Policy.DefaultVersionId,
		})
		if err != nil {
			msg := fmt.Sprintf("Error retrieving policy version: %v", err)
			findings = append(findings, msg)
			scanner.logger.Logger(ctx).Error("Failed to get policy version",
				zap.String("policy_name", policyName),
				zap.Error(err))
			continue
		}

		decodedDocument, err := url.QueryUnescape(*policyVersion.PolicyVersion.Document)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to decode policy document",
				zap.String("policy_name", policyName),
				zap.Error(err))
			findings = append(findings, fmt.Sprintf("Error decoding policy document for policy %s: %v", policyName, err))
			continue
		}

		var policyDoc map[string]interface{}
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			scanner.logger.Logger(ctx).Error("Failed to parse policy document",
				zap.String("policy_name", policyName),
				zap.Error(err))
			findings = append(findings, fmt.Sprintf("Error parsing policy document for policy %s: %v", policyName, err))
			continue
		}

		statements, ok := policyDoc["Statement"].([]interface{})
		if !ok {
			scanner.logger.Logger(ctx).Warn("Invalid policy document structure",
				zap.String("policy_name", policyName))
			findings = append(findings, fmt.Sprintf("Invalid policy document structure for policy %s", policyName))
			continue
		}

		for _, stmt := range statements {
			statement, ok := stmt.(map[string]interface{})
			if !ok {
				scanner.logger.Logger(ctx).Warn("Invalid statement structure",
					zap.String("policy_name", policyName))
				findings = append(findings, fmt.Sprintf("Invalid statement structure in policy %s", policyName))
				continue
			}
			if action, ok := statement["Action"].(interface{}); ok {
				if action == "*" {
					msg := fmt.Sprintf("WARNING: Lambda function %s has overly permissive actions in IAM role %s.", funcName, roleName)
					findings = append(findings, msg)
					scanner.logger.Logger(ctx).Warn("Overly permissive IAM policy detected",
						zap.String("function_name", funcName),
						zap.String("role_name", roleName),
						zap.String("policy_name", policyName))
				}
			}
		}
	}

	scanner.logger.Logger(ctx).Info("Completed IAM role check",
		zap.String("function_name", funcName),
		zap.String("role_name", roleName),
		zap.Int("finding_count", len(findings)))
	return findings, nil
}

// Check Lambda function environment variables for sensitive information
func (scanner *LambdaScanner) CheckSensitiveEnvironmentVariables(ctx context.Context, function *lambda.FunctionConfiguration) []string {
	tracer := otel.Tracer("lambdascanner/CheckSensitiveEnvironmentVariables")
	ctx, span := tracer.Start(ctx, "CheckSensitiveEnvironmentVariables")
	defer span.End()

	funcName := aws.StringValue(function.FunctionName)
	scanner.logger.Logger(ctx).Info("Checking environment variables",
		zap.String("function_name", funcName))

	var findings []string

	envVars := function.Environment
	if envVars != nil && envVars.Variables != nil {
		scanner.logger.Logger(ctx).Debug("Analyzing environment variables",
			zap.String("function_name", funcName),
			zap.Int("variable_count", len(envVars.Variables)))

		for key := range envVars.Variables {
			if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "secret") {
				msg := fmt.Sprintf("WARNING: Lambda function %s has potentially sensitive environment variable: %s=REDACTED", funcName, key)
				findings = append(findings, msg)
				scanner.logger.Logger(ctx).Warn("Potentially sensitive environment variable detected",
					zap.String("function_name", funcName),
					zap.String("variable_name", key))
			}
		}
	} else {
		msg := fmt.Sprintf("No environment variables found for Lambda function %s.", funcName)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Info("No environment variables found",
			zap.String("function_name", funcName))
	}

	scanner.logger.Logger(ctx).Info("Completed environment variable check",
		zap.String("function_name", funcName),
		zap.Int("finding_count", len(findings)))
	return findings
}

// Check if Lambda function has CloudWatch Logs enabled
func (scanner *LambdaScanner) CheckCloudWatchLogs(ctx context.Context, function *lambda.FunctionConfiguration) ([]string, error) {
	tracer := otel.Tracer("lambdascanner/CheckCloudWatchLogs")
	ctx, span := tracer.Start(ctx, "CheckCloudWatchLogs")
	defer span.End()

	funcName := aws.StringValue(function.FunctionName)
	scanner.logger.Logger(ctx).Info("Checking CloudWatch Logs configuration",
		zap.String("function_name", funcName))

	var findings []string

	roleArn := aws.StringValue(function.Role)
	if roleArn == "" {
		msg := fmt.Sprintf("WARNING: Lambda function %s does not have a valid IAM role assigned.", funcName)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Warn("Lambda function missing IAM role",
			zap.String("function_name", funcName))
		return findings, nil
	}

	roleParts := strings.Split(roleArn, "/")
	if len(roleParts) < 2 {
		msg := fmt.Sprintf("WARNING: Lambda function %s has an invalid IAM role ARN: %s", funcName, roleArn)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Warn("Invalid IAM role ARN",
			zap.String("function_name", funcName),
			zap.String("role_arn", roleArn))
		return findings, nil
	}
	roleName := roleParts[1]

	_, err := scanner.iamClient.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			msg := fmt.Sprintf("WARNING: IAM role %s for Lambda function %s does not exist.", roleName, funcName)
			findings = append(findings, msg)
			scanner.logger.Logger(ctx).Warn("IAM role does not exist",
				zap.String("function_name", funcName),
				zap.String("role_name", roleName))
		} else {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to get IAM role",
				zap.String("function_name", funcName),
				zap.String("role_name", roleName),
				zap.Error(err))
			return nil, fmt.Errorf("error getting IAM role %s: %v", roleName, err)
		}
		return findings, nil
	}

	hasLoggingPermission := false
	policies, err := scanner.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		span.RecordError(err)
		scanner.logger.Logger(ctx).Error("Failed to list attached role policies",
			zap.String("function_name", funcName),
			zap.String("role_name", roleName),
			zap.Error(err))
		return nil, fmt.Errorf("error listing attached policies for role %s: %v", roleName, err)
	}

	scanner.logger.Logger(ctx).Debug("Analyzing attached policies",
		zap.String("function_name", funcName),
		zap.String("role_name", roleName),
		zap.Int("policy_count", len(policies.AttachedPolicies)))

	for _, policy := range policies.AttachedPolicies {
		policyDetail, err := scanner.iamClient.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to get policy details",
				zap.String("policy_name", aws.StringValue(policy.PolicyName)),
				zap.Error(err))
			continue
		}

		version, err := scanner.iamClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetail.Policy.DefaultVersionId,
		})
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to get policy version",
				zap.String("policy_name", aws.StringValue(policy.PolicyName)),
				zap.Error(err))
			continue
		}

		decodedDocument, err := url.QueryUnescape(*version.PolicyVersion.Document)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to decode policy document",
				zap.String("policy_name", aws.StringValue(policy.PolicyName)),
				zap.Error(err))
			findings = append(findings, fmt.Sprintf("Error decoding policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		var policyDoc map[string]interface{}
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			span.RecordError(err)
			scanner.logger.Logger(ctx).Error("Failed to parse policy document",
				zap.String("policy_name", aws.StringValue(policy.PolicyName)),
				zap.Error(err))
			continue
		}

		statements, ok := policyDoc["Statement"].([]interface{})
		if !ok {
			scanner.logger.Logger(ctx).Warn("Invalid policy document structure",
				zap.String("policy_name", aws.StringValue(policy.PolicyName)))
			findings = append(findings, fmt.Sprintf("Invalid policy document structure for policy %s", aws.StringValue(policy.PolicyName)))
			continue
		}

		for _, stmt := range statements {
			statement, ok := stmt.(map[string]interface{})
			if !ok {
				scanner.logger.Logger(ctx).Warn("Invalid statement structure",
					zap.String("policy_name", aws.StringValue(policy.PolicyName)))
				findings = append(findings, fmt.Sprintf("Invalid statement structure in policy %s", aws.StringValue(policy.PolicyName)))
				continue
			}
			actions := statement["Action"]
			if actionList, ok := actions.([]interface{}); ok {
				for _, action := range actionList {
					if action == "logs:CreateLogGroup" || action == "logs:CreateLogStream" || action == "logs:PutLogEvents" {
						hasLoggingPermission = true
						break
					}
				}
			} else if actionStr, ok := actions.(string); ok {
				if actionStr == "logs:CreateLogGroup" || actionStr == "logs:CreateLogStream" || actionStr == "logs:PutLogEvents" {
					hasLoggingPermission = true
					break
				}
			}
		}
	}

	if hasLoggingPermission {
		findings = append(findings, fmt.Sprintf("CloudWatch logging is enabled for Lambda function %s.", funcName))
		scanner.logger.Logger(ctx).Info("CloudWatch logging is enabled",
			zap.String("function_name", funcName))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: CloudWatch logging is not enabled for Lambda function %s.", funcName))
		scanner.logger.Logger(ctx).Warn("CloudWatch logging is not enabled",
			zap.String("function_name", funcName))
	}

	scanner.logger.Logger(ctx).Info("Completed CloudWatch logs check",
		zap.String("function_name", funcName),
		zap.Int("finding_count", len(findings)))
	return findings, nil
}

// Check Lambda function timeout configuration
func (scanner *LambdaScanner) CheckFunctionTimeout(ctx context.Context, function *lambda.FunctionConfiguration) []string {
	tracer := otel.Tracer("lambdascanner/CheckFunctionTimeout")
	ctx, span := tracer.Start(ctx, "CheckFunctionTimeout")
	defer span.End()

	funcName := aws.StringValue(function.FunctionName)
	scanner.logger.Logger(ctx).Info("Checking function timeout configuration",
		zap.String("function_name", funcName),
		zap.Int64("timeout", aws.Int64Value(function.Timeout)))

	var findings []string
	timeout := aws.Int64Value(function.Timeout)

	if timeout > 60 {
		msg := fmt.Sprintf("WARNING: Lambda function %s has a high timeout value of %d seconds", funcName, timeout)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Warn("High function timeout detected",
			zap.String("function_name", funcName),
			zap.Int64("timeout", timeout))
	}

	scanner.logger.Logger(ctx).Info("Completed timeout configuration check",
		zap.String("function_name", funcName),
		zap.Int("finding_count", len(findings)))
	return findings
}

// Check if Lambda function is in a VPC
func (scanner *LambdaScanner) CheckVpcConfiguration(ctx context.Context, function *lambda.FunctionConfiguration) []string {
	tracer := otel.Tracer("lambdascanner/CheckVpcConfiguration")
	ctx, span := tracer.Start(ctx, "CheckVpcConfiguration")
	defer span.End()

	funcName := aws.StringValue(function.FunctionName)
	scanner.logger.Logger(ctx).Info("Checking VPC configuration",
		zap.String("function_name", funcName))

	var findings []string

	if function.VpcConfig == nil || len(function.VpcConfig.SubnetIds) == 0 {
		msg := fmt.Sprintf("WARNING: Lambda function %s is not configured in a VPC", funcName)
		findings = append(findings, msg)
		scanner.logger.Logger(ctx).Warn("Function not in VPC",
			zap.String("function_name", funcName))
	} else {
		scanner.logger.Logger(ctx).Debug("Function VPC configuration",
			zap.String("function_name", funcName),
			zap.Strings("subnet_ids", aws.StringValueSlice(function.VpcConfig.SubnetIds)),
			zap.Strings("security_group_ids", aws.StringValueSlice(function.VpcConfig.SecurityGroupIds)))
	}

	scanner.logger.Logger(ctx).Info("Completed VPC configuration check",
		zap.String("function_name", funcName),
		zap.Int("finding_count", len(findings)))
	return findings
}
