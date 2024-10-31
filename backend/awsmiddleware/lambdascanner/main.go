package lambdascanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"guarddev/logger"
	"guarddev/modelapi/geminiapi"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type LambdaScanner struct {
	lambdaClient *lambda.Lambda
	iamClient    *iam.IAM
	logger       *logger.LogMiddleware
	Gemini       *geminiapi.Gemini
}

// Initialize a Lambda scanner with assumed credentials
func NewLambdaScanner(accessKey, secretKey, sessionToken, region string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *LambdaScanner {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, sessionToken),
	}))
	return &LambdaScanner{
		lambdaClient: lambda.New(sess),
		iamClient:    iam.New(sess),
		logger:       logger,
		Gemini:       gemini,
	}
}

// List all Lambda functions
func (scanner *LambdaScanner) ListFunctions(ctx context.Context) ([]*lambda.FunctionConfiguration, error) {
	tracer := otel.Tracer("lambdascanner/ListFunctions")
	ctx, span := tracer.Start(ctx, "ListFunctions")
	defer span.End()

	result, err := scanner.lambdaClient.ListFunctions(&lambda.ListFunctionsInput{})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error listing Lambda functions: %v", err)
	}

	span.SetAttributes(attribute.Int("function_count", len(result.Functions)))
	return result.Functions, nil
}

// Check if Lambda function has overly permissive IAM role
func (scanner *LambdaScanner) CheckFunctionIAMRole(ctx context.Context, function *lambda.FunctionConfiguration) ([]string, error) {
	tracer := otel.Tracer("lambdascanner/CheckFunctionIAMRole")
	ctx, span := tracer.Start(ctx, "CheckFunctionIAMRole")
	defer span.End()

	var findings []string

	roleArn := aws.StringValue(function.Role)
	if roleArn == "" {
		findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s does not have a valid IAM role assigned.", aws.StringValue(function.FunctionName)))
		return findings, nil
	}

	roleParts := strings.Split(roleArn, "/")
	if len(roleParts) < 2 {
		findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s has an invalid IAM role ARN: %s", aws.StringValue(function.FunctionName), roleArn))
		return findings, nil
	}
	roleName := roleParts[1]

	_, err := scanner.iamClient.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			findings = append(findings, fmt.Sprintf("WARNING: IAM role %s for Lambda function %s does not exist.", roleName, aws.StringValue(function.FunctionName)))
		} else {
			span.RecordError(err)
			return nil, fmt.Errorf("error getting IAM role %s: %v", roleName, err)
		}
		return findings, nil
	}

	policies, err := scanner.iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: aws.String(roleName)})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error listing attached policies for role %s: %v", roleName, err)
	}

	for _, policy := range policies.AttachedPolicies {
		policyDetails, err := scanner.iamClient.GetPolicy(&iam.GetPolicyInput{PolicyArn: policy.PolicyArn})
		if err != nil {
			findings = append(findings, fmt.Sprintf("Error retrieving policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		policyVersion, err := scanner.iamClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetails.Policy.DefaultVersionId,
		})
		if err != nil {
			findings = append(findings, fmt.Sprintf("Error retrieving policy version: %v", err))
			continue
		}

		// Decode the policy document
		decodedDocument, err := url.QueryUnescape(*policyVersion.PolicyVersion.Document)
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error decoding policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		var policyDoc map[string]interface{}
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			findings = append(findings, fmt.Sprintf("Error parsing policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		statements, ok := policyDoc["Statement"].([]interface{})
		if !ok {
			findings = append(findings, fmt.Sprintf("Invalid policy document structure for policy %s", aws.StringValue(policy.PolicyName)))
			continue
		}

		for _, stmt := range statements {
			statement, ok := stmt.(map[string]interface{})
			if !ok {
				findings = append(findings, fmt.Sprintf("Invalid statement structure in policy %s", aws.StringValue(policy.PolicyName)))
				continue
			}
			if action, ok := statement["Action"].(interface{}); ok {
				if action == "*" {
					findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s has overly permissive actions in IAM role %s.", aws.StringValue(function.FunctionName), roleName))
				}
			}
		}
	}

	return findings, nil
}

// Check Lambda function environment variables for sensitive information
func (scanner *LambdaScanner) CheckSensitiveEnvironmentVariables(ctx context.Context, function *lambda.FunctionConfiguration) []string {
	tracer := otel.Tracer("lambdascanner/CheckSensitiveEnvironmentVariables")
	ctx, span := tracer.Start(ctx, "CheckSensitiveEnvironmentVariables")
	defer span.End()

	var findings []string

	envVars := function.Environment
	if envVars != nil && envVars.Variables != nil {
		for key, _ := range envVars.Variables {
			if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "secret") {
				findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s has potentially sensitive environment variable: %s=REDACTED", aws.StringValue(function.FunctionName), key))
			}
		}
	} else {
		findings = append(findings, fmt.Sprintf("No environment variables found for Lambda function %s.", aws.StringValue(function.FunctionName)))
	}

	return findings
}

// Check if Lambda function has CloudWatch Logs enabled
func (scanner *LambdaScanner) CheckCloudWatchLogs(ctx context.Context, function *lambda.FunctionConfiguration) ([]string, error) {
	tracer := otel.Tracer("lambdascanner/CheckCloudWatchLogs")
	ctx, span := tracer.Start(ctx, "CheckCloudWatchLogs")
	defer span.End()

	var findings []string

	roleArn := aws.StringValue(function.Role)
	if roleArn == "" {
		findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s does not have a valid IAM role assigned.", aws.StringValue(function.FunctionName)))
		return findings, nil
	}

	roleParts := strings.Split(roleArn, "/")
	if len(roleParts) < 2 {
		findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s has an invalid IAM role ARN: %s", aws.StringValue(function.FunctionName), roleArn))
		return findings, nil
	}
	roleName := roleParts[1]

	_, err := scanner.iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			findings = append(findings, fmt.Sprintf("WARNING: IAM role %s for Lambda function %s does not exist.", roleName, aws.StringValue(function.FunctionName)))
		} else {
			span.RecordError(err)
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
		return nil, fmt.Errorf("error listing attached policies for role %s: %v", roleName, err)
	}

	for _, policy := range policies.AttachedPolicies {
		policyDetail, err := scanner.iamClient.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			span.RecordError(err)
			continue
		}

		version, err := scanner.iamClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetail.Policy.DefaultVersionId,
		})
		if err != nil {
			span.RecordError(err)
			continue
		}

		// Decode the policy document
		decodedDocument, err := url.QueryUnescape(*version.PolicyVersion.Document)
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error decoding policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		var policyDoc map[string]interface{}
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			span.RecordError(err)
			continue
		}

		statements, ok := policyDoc["Statement"].([]interface{})
		if !ok {
			findings = append(findings, fmt.Sprintf("Invalid policy document structure for policy %s", aws.StringValue(policy.PolicyName)))
			continue
		}

		for _, stmt := range statements {
			statement, ok := stmt.(map[string]interface{})
			if !ok {
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
		findings = append(findings, fmt.Sprintf("CloudWatch logging is enabled for Lambda function %s.", aws.StringValue(function.FunctionName)))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: CloudWatch logging is not enabled for Lambda function %s.", aws.StringValue(function.FunctionName)))
	}

	return findings, nil
}

// Check Lambda function timeout configuration
func (scanner *LambdaScanner) CheckFunctionTimeout(ctx context.Context, function *lambda.FunctionConfiguration) []string {
	tracer := otel.Tracer("lambdascanner/CheckFunctionTimeout")
	ctx, span := tracer.Start(ctx, "CheckFunctionTimeout")
	defer span.End()

	var findings []string

	timeout := aws.Int64Value(function.Timeout)
	if timeout > 60 {
		findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s has a timeout set to %d seconds, which may be too high.", aws.StringValue(function.FunctionName), timeout))
	}

	return findings
}

// Check if Lambda function is in a VPC
func (scanner *LambdaScanner) CheckVpcConfiguration(ctx context.Context, function *lambda.FunctionConfiguration) []string {
	tracer := otel.Tracer("lambdascanner/CheckVpcConfiguration")
	ctx, span := tracer.Start(ctx, "CheckVpcConfiguration")
	defer span.End()

	var findings []string

	vpcConfig := function.VpcConfig
	if vpcConfig == nil || len(vpcConfig.SubnetIds) == 0 {
		findings = append(findings, fmt.Sprintf("WARNING: Lambda function %s is not configured in a VPC.", aws.StringValue(function.FunctionName)))
	}

	return findings
}
