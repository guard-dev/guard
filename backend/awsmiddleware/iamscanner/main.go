package iamscanner

import (
	"context"
	"encoding/json"
	"fmt"
	"guarddev/logger"
	"guarddev/modelapi/geminiapi"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type IAMScanner struct {
	IAMClient *iam.IAM
	logger    *logger.LogMiddleware
	Gemini    *geminiapi.Gemini
	accountId string
}

type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

type Statement struct {
	Effect    string      `json:"Effect"`
	Action    interface{} `json:"Action"`
	Resource  interface{} `json:"Resource"`
	Principal *Principal  `json:"Principal,omitempty"`
}

type Principal struct {
	AWS string `json:"AWS"`
}

// Initialize an IAM scanner with assumed credentials
func NewIAMScanner(accessKey, secretKey, sessionToken, region string, accountId string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *IAMScanner {
	// Create a new AWS session with the provided credentials
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			sessionToken,
		),
	}))
	// Return a new IAMScanner instance
	return &IAMScanner{
		IAMClient: iam.New(sess),
		logger:    logger,
		Gemini:    gemini,
		accountId: accountId,
	}
}

// List all IAM users
func (scanner *IAMScanner) ListUsers(ctx context.Context) ([]*iam.User, error) {
	tracer := otel.Tracer("iamscanner/ListUsers")
	ctx, span := tracer.Start(ctx, "ListUsers")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting IAM user listing")

	result, err := scanner.IAMClient.ListUsers(nil)
	if err != nil {
		errorMsg := fmt.Sprintf("Error listing IAM users: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		span.RecordError(err)
		return nil, fmt.Errorf(errorMsg)
	}

	userCount := len(result.Users)
	scanner.logger.Logger(ctx).Info("Successfully listed IAM users",
		zap.Int("user_count", userCount))
	span.SetAttributes(attribute.Int("user_count", userCount))
	return result.Users, nil
}

// Check for unused IAM access keys
func (scanner *IAMScanner) CheckUnusedAccessKeys(ctx context.Context, user *iam.User) []string {
	tracer := otel.Tracer("iamscanner/CheckUnusedAccessKeys")
	ctx, span := tracer.Start(ctx, "CheckUnusedAccessKeys")
	defer span.End()

	var findings []string
	userName := aws.StringValue(user.UserName)

	scanner.logger.Logger(ctx).Info("Checking unused access keys",
		zap.String("user_name", userName))

	keys, err := scanner.IAMClient.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: user.UserName,
	})
	if err != nil {
		errorMsg := fmt.Sprintf("Error listing access keys for user %s: %v", userName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("user_name", userName),
			zap.Error(err))
		span.RecordError(err)
		findings = append(findings, errorMsg)
		return findings
	}

	for _, key := range keys.AccessKeyMetadata {
		keyId := aws.StringValue(key.AccessKeyId)
		scanner.logger.Logger(ctx).Debug("Checking access key last usage",
			zap.String("user_name", userName),
			zap.String("access_key_id", keyId))

		lastUsed, err := scanner.IAMClient.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			errorMsg := fmt.Sprintf("Error checking last used time for access key %s: %v", keyId, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("user_name", userName),
				zap.String("access_key_id", keyId),
				zap.Error(err))
			span.RecordError(err)
			findings = append(findings, errorMsg)
			continue
		}

		if lastUsed.AccessKeyLastUsed.LastUsedDate == nil || time.Since(*lastUsed.AccessKeyLastUsed.LastUsedDate) > 90*24*time.Hour {
			finding := fmt.Sprintf("WARNING: Access key %s for user %s has not been used for over 90 days.", keyId, userName)
			scanner.logger.Logger(ctx).Warn("Found unused access key",
				zap.String("user_name", userName),
				zap.String("access_key_id", keyId),
				zap.Time("last_used", *lastUsed.AccessKeyLastUsed.LastUsedDate))
			findings = append(findings, finding)
		}
	}

	scanner.logger.Logger(ctx).Info("Completed unused access key check",
		zap.String("user_name", userName),
		zap.Int("finding_count", len(findings)))
	span.SetAttributes(attribute.Int("finding_count", len(findings)))
	return findings
}

// Check for overly permissive IAM policies (wildcards "*")
func (scanner *IAMScanner) CheckOverlyPermissivePolicies(ctx context.Context, user *iam.User) []string {
	tracer := otel.Tracer("iamscanner/CheckOverlyPermissivePolicies")
	ctx, span := tracer.Start(ctx, "CheckOverlyPermissivePolicies")
	defer span.End()

	var findings []string
	userName := aws.StringValue(user.UserName)

	scanner.logger.Logger(ctx).Info("Checking for overly permissive policies",
		zap.String("user_name", userName))

	policies, err := scanner.IAMClient.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: user.UserName,
	})
	if err != nil {
		errorMsg := fmt.Sprintf("Error listing policies for user %s: %v", userName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("user_name", userName),
			zap.Error(err))
		span.RecordError(err)
		findings = append(findings, errorMsg)
		return findings
	}

	for _, policy := range policies.AttachedPolicies {
		policyName := aws.StringValue(policy.PolicyName)
		scanner.logger.Logger(ctx).Debug("Analyzing policy",
			zap.String("user_name", userName),
			zap.String("policy_name", policyName))

		policyDetail, err := scanner.IAMClient.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			errorMsg := fmt.Sprintf("Error retrieving policy %s for user %s: %v", policyName, userName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("user_name", userName),
				zap.String("policy_name", policyName),
				zap.Error(err))
			span.RecordError(err)
			findings = append(findings, errorMsg)
			continue
		}

		version, err := scanner.IAMClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetail.Policy.DefaultVersionId,
		})
		if err != nil {
			errorMsg := fmt.Sprintf("Error retrieving policy version %s: %v", aws.StringValue(policyDetail.Policy.DefaultVersionId), err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("user_name", userName),
				zap.String("policy_name", policyName),
				zap.String("version_id", aws.StringValue(policyDetail.Policy.DefaultVersionId)),
				zap.Error(err))
			span.RecordError(err)
			findings = append(findings, errorMsg)
			continue
		}

		decodedDocument, err := url.QueryUnescape(*version.PolicyVersion.Document)
		if err != nil {
			errorMsg := fmt.Sprintf("Error decoding policy document for policy %s: %v", policyName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("user_name", userName),
				zap.String("policy_name", policyName),
				zap.Error(err))
			span.RecordError(err)
			findings = append(findings, errorMsg)
			continue
		}

		var policyDoc PolicyDocument
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			errorMsg := fmt.Sprintf("Error parsing policy document for policy %s: %v", policyName, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("user_name", userName),
				zap.String("policy_name", policyName),
				zap.Error(err))
			span.RecordError(err)
			findings = append(findings, errorMsg)
			continue
		}

		for _, statement := range policyDoc.Statement {
			var actions []string
			switch v := statement.Action.(type) {
			case string:
				actions = []string{v}
			case []interface{}:
				for _, a := range v {
					if actionStr, ok := a.(string); ok {
						actions = append(actions, actionStr)
					}
				}
			}

			for _, action := range actions {
				if action == "*" {
					finding := fmt.Sprintf("WARNING: Policy %s attached to user %s allows overly permissive actions (wildcard \"*\").", policyName, userName)
					scanner.logger.Logger(ctx).Warn("Found overly permissive policy",
						zap.String("user_name", userName),
						zap.String("policy_name", policyName),
						zap.String("action", action))
					findings = append(findings, finding)
				}
			}
		}
	}

	scanner.logger.Logger(ctx).Info("Completed overly permissive policy check",
		zap.String("user_name", userName),
		zap.Int("finding_count", len(findings)))
	span.SetAttributes(attribute.Int("finding_count", len(findings)))
	return findings
}

// Check if root account has been used
func (scanner *IAMScanner) CheckRootAccountUsage(ctx context.Context) ([]string, error) {
	tracer := otel.Tracer("iamscanner/CheckRootAccountUsage")
	ctx, span := tracer.Start(ctx, "CheckRootAccountUsage")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Checking root account usage",
		zap.String("account_id", scanner.accountId))

	var findings []string

	// Get credential report
	_, err := scanner.IAMClient.GenerateCredentialReport(&iam.GenerateCredentialReportInput{})
	if err != nil {
		errorMsg := fmt.Sprintf("Error generating credential report: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		span.RecordError(err)
		return nil, fmt.Errorf(errorMsg)
	}

	report, err := scanner.IAMClient.GetCredentialReport(&iam.GetCredentialReportInput{})
	if err != nil {
		errorMsg := fmt.Sprintf("Error retrieving credential report: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		span.RecordError(err)
		return nil, fmt.Errorf(errorMsg)
	}

	// Check for root account usage in last 24 hours
	if report.Content != nil {
		scanner.logger.Logger(ctx).Debug("Analyzing credential report")
		// Parse and check the report content
		// Add specific checks for root account usage
		findings = append(findings, "Root account usage analysis completed - implement specific checks")
	}

	scanner.logger.Logger(ctx).Info("Completed root account usage check",
		zap.Int("finding_count", len(findings)))
	span.SetAttributes(attribute.Int("finding_count", len(findings)))
	return findings, nil
}

// Check for overly permissive role trust policies
func (scanner *IAMScanner) CheckRoleTrustPolicies(ctx context.Context, role *iam.Role) []string {
	tracer := otel.Tracer("iamscanner/CheckRoleTrustPolicies")
	ctx, span := tracer.Start(ctx, "CheckRoleTrustPolicies")
	defer span.End()

	var findings []string
	roleName := aws.StringValue(role.RoleName)

	scanner.logger.Logger(ctx).Info("Checking role trust policy",
		zap.String("role_name", roleName))

	if role.AssumeRolePolicyDocument == nil {
		scanner.logger.Logger(ctx).Debug("No trust policy document found",
			zap.String("role_name", roleName))
		return findings
	}

	decodedDocument, err := url.QueryUnescape(*role.AssumeRolePolicyDocument)
	if err != nil {
		errorMsg := fmt.Sprintf("Error decoding trust policy document for role %s: %v", roleName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("role_name", roleName),
			zap.Error(err))
		span.RecordError(err)
		findings = append(findings, errorMsg)
		return findings
	}

	var policyDoc PolicyDocument
	err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
	if err != nil {
		errorMsg := fmt.Sprintf("Error parsing trust policy document for role %s: %v", roleName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("role_name", roleName),
			zap.Error(err))
		span.RecordError(err)
		findings = append(findings, errorMsg)
		return findings
	}

	for _, statement := range policyDoc.Statement {
		if statement.Principal != nil && statement.Principal.AWS == "*" {
			finding := fmt.Sprintf("WARNING: Role %s has an overly permissive trust policy allowing any AWS account.", roleName)
			scanner.logger.Logger(ctx).Warn("Found overly permissive trust policy",
				zap.String("role_name", roleName),
				zap.String("principal", statement.Principal.AWS))
			findings = append(findings, finding)
		}
	}

	scanner.logger.Logger(ctx).Info("Completed role trust policy check",
		zap.String("role_name", roleName),
		zap.Int("finding_count", len(findings)))
	span.SetAttributes(attribute.Int("finding_count", len(findings)))
	return findings
}

// Check for privilege escalation risks
func (scanner *IAMScanner) CheckPrivilegeEscalation(ctx context.Context, user *iam.User) []string {
	tracer := otel.Tracer("iamscanner/CheckPrivilegeEscalation")
	ctx, span := tracer.Start(ctx, "CheckPrivilegeEscalation")
	defer span.End()

	var findings []string
	userName := aws.StringValue(user.UserName)

	scanner.logger.Logger(ctx).Info("Checking for privilege escalation risks",
		zap.String("user_name", userName))

	policies, err := scanner.IAMClient.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: user.UserName,
	})
	if err != nil {
		errorMsg := fmt.Sprintf("Error listing policies for user %s: %v", userName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("user_name", userName),
			zap.Error(err))
		span.RecordError(err)
		findings = append(findings, errorMsg)
		return findings
	}

	for _, policy := range policies.AttachedPolicies {
		policyName := aws.StringValue(policy.PolicyName)
		scanner.logger.Logger(ctx).Debug("Analyzing policy for privilege escalation",
			zap.String("user_name", userName),
			zap.String("policy_name", policyName))

		// Check for policies that could lead to privilege escalation
		// This is a simplified check - implement more comprehensive checks based on your security requirements
		if policyName == "AdministratorAccess" {
			finding := fmt.Sprintf("WARNING: User %s has administrator access which could lead to privilege escalation.", userName)
			scanner.logger.Logger(ctx).Warn("Found potential privilege escalation risk",
				zap.String("user_name", userName),
				zap.String("policy_name", policyName))
			findings = append(findings, finding)
		}
	}

	scanner.logger.Logger(ctx).Info("Completed privilege escalation check",
		zap.String("user_name", userName),
		zap.Int("finding_count", len(findings)))
	span.SetAttributes(attribute.Int("finding_count", len(findings)))
	return findings
}

// Check MFA configuration for all users
func (scanner *IAMScanner) CheckUserMFAStatus(ctx context.Context, user *iam.User) []string {
	tracer := otel.Tracer("iamscanner/CheckUserMFAStatus")
	ctx, span := tracer.Start(ctx, "CheckUserMFAStatus")
	defer span.End()

	var findings []string
	userName := aws.StringValue(user.UserName)

	scanner.logger.Logger(ctx).Info("Checking MFA status",
		zap.String("user_name", userName))

	mfaDevices, err := scanner.IAMClient.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: user.UserName,
	})
	if err != nil {
		errorMsg := fmt.Sprintf("Error checking MFA devices for user %s: %v", userName, err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("user_name", userName),
			zap.Error(err))
		span.RecordError(err)
		findings = append(findings, errorMsg)
		return findings
	}

	if len(mfaDevices.MFADevices) == 0 {
		finding := fmt.Sprintf("WARNING: User %s does not have MFA enabled.", userName)
		scanner.logger.Logger(ctx).Warn("MFA not enabled",
			zap.String("user_name", userName))
		findings = append(findings, finding)
	}

	scanner.logger.Logger(ctx).Info("Completed MFA status check",
		zap.String("user_name", userName),
		zap.Int("finding_count", len(findings)),
		zap.Int("mfa_device_count", len(mfaDevices.MFADevices)))
	span.SetAttributes(
		attribute.Int("finding_count", len(findings)),
		attribute.Int("mfa_device_count", len(mfaDevices.MFADevices)))
	return findings
}
