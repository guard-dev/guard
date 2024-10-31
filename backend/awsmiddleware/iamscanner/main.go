package iamscanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"guarddev/logger"
	"guarddev/modelapi/geminiapi"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type IAMScanner struct {
	IAMClient *iam.IAM
	logger    *logger.LogMiddleware
	Gemini    *geminiapi.Gemini
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
func NewIAMScanner(accessKey, secretKey, sessionToken, region string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *IAMScanner {
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
	}
}

// List all IAM users
func (scanner *IAMScanner) ListUsers(ctx context.Context) ([]*iam.User, error) {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/ListUsers")
	ctx, span := tracer.Start(ctx, "ListUsers")
	defer span.End()

	// List all IAM users in the account
	result, err := scanner.IAMClient.ListUsers(nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error listing IAM users: %v", err)
	}

	// Record the number of users found
	span.SetAttributes(attribute.Int("user_count", len(result.Users)))
	return result.Users, nil
}

// Check for unused IAM access keys
func (scanner *IAMScanner) CheckUnusedAccessKeys(ctx context.Context, user *iam.User) []string {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/CheckUnusedAccessKeys")
	ctx, span := tracer.Start(ctx, "CheckUnusedAccessKeys")
	defer span.End()

	var findings []string

	// List all access keys for the given user
	keys, err := scanner.IAMClient.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: user.UserName,
	})
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error listing access keys for user %s: %v", aws.StringValue(user.UserName), err))
		return findings
	}

	// Check each access key to see if it has been used recently
	for _, key := range keys.AccessKeyMetadata {
		lastUsed, err := scanner.IAMClient.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error checking last used time for access key %s: %v", aws.StringValue(key.AccessKeyId), err))
			continue
		}

		// If the key has not been used in over 90 days, add a warning
		if lastUsed.AccessKeyLastUsed.LastUsedDate == nil || time.Since(*lastUsed.AccessKeyLastUsed.LastUsedDate) > 90*24*time.Hour {
			findings = append(findings, fmt.Sprintf("WARNING: Access key %s for user %s has not been used for over 90 days.", aws.StringValue(key.AccessKeyId), aws.StringValue(user.UserName)))
		}
	}

	return findings
}

// Check for overly permissive IAM policies (wildcards "*")
func (scanner *IAMScanner) CheckOverlyPermissivePolicies(ctx context.Context, user *iam.User) []string {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/CheckOverlyPermissivePolicies")
	ctx, span := tracer.Start(ctx, "CheckOverlyPermissivePolicies")
	defer span.End()

	var findings []string

	// List all attached user policies
	policies, err := scanner.IAMClient.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: user.UserName,
	})
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error listing policies for user %s: %v", aws.StringValue(user.UserName), err))
		return findings
	}

	// Iterate through each policy to check for overly permissive actions
	for _, policy := range policies.AttachedPolicies {
		// Retrieve the policy details
		policyDetail, err := scanner.IAMClient.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error retrieving policy %s for user %s: %v", aws.StringValue(policy.PolicyName), aws.StringValue(user.UserName), err))
			continue
		}

		// Retrieve the default policy version
		version, err := scanner.IAMClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetail.Policy.DefaultVersionId,
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error retrieving policy version %s: %v", aws.StringValue(policyDetail.Policy.DefaultVersionId), err))
			continue
		}

		// Decode the policy document
		decodedDocument, err := url.QueryUnescape(*version.PolicyVersion.Document)
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error decoding policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		// Parse the policy document
		var policyDoc PolicyDocument
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error parsing policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		// Check each statement for overly permissive actions (wildcards "*")
		for _, statement := range policyDoc.Statement {
			// Handle Action as string or []string
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
					findings = append(findings, fmt.Sprintf("WARNING: Policy %s attached to user %s allows overly permissive actions (wildcard \"*\").", aws.StringValue(policy.PolicyName), aws.StringValue(user.UserName)))
				}
			}
		}
	}

	return findings
}

// Check if root account has been used
func (scanner *IAMScanner) CheckRootAccountUsage(ctx context.Context) ([]string, error) {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/CheckRootAccountUsage")
	ctx, span := tracer.Start(ctx, "CheckRootAccountUsage")
	defer span.End()

	var findings []string

	// Retrieve account summary information
	result, err := scanner.IAMClient.GetAccountSummary(nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error retrieving account summary: %v", err)
	}

	summaryMap := result.SummaryMap

	// Check if the root account has active access keys
	if accountAccessKeysPresent, ok := summaryMap["AccountAccessKeysPresent"]; ok && *accountAccessKeysPresent > 0 {
		findings = append(findings, "WARNING: Root account has active access keys.")
	}

	// Check if MFA is enabled for the root account
	if accountMFAEnabled, ok := summaryMap["AccountMFAEnabled"]; ok && *accountMFAEnabled == 0 {
		findings = append(findings, "WARNING: MFA is not enabled for the root account.")
	}

	// If no warnings, add a message indicating root account security appears adequate
	if len(findings) == 0 {
		findings = append(findings, "Root account usage appears secure based on available information.")
	}

	return findings, nil
}

// Check for overly permissive role trust policies
func (scanner *IAMScanner) CheckRoleTrustPolicies(ctx context.Context, role *iam.Role) []string {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/CheckRoleTrustPolicies")
	ctx, span := tracer.Start(ctx, "CheckRoleTrustPolicies")
	defer span.End()

	var findings []string

	// Retrieve the trust policy for the given role
	roleTrustPolicy, err := scanner.IAMClient.GetRole(&iam.GetRoleInput{
		RoleName: role.RoleName,
	})
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error retrieving trust policy for role %s: %v", aws.StringValue(role.RoleName), err))
		return findings
	}

	// Decode the trust policy document
	decodedDocument, err := url.QueryUnescape(*roleTrustPolicy.Role.AssumeRolePolicyDocument)
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error decoding trust policy document for role %s: %v", aws.StringValue(role.RoleName), err))
		return findings
	}

	// Parse the trust policy document
	var policyDoc PolicyDocument
	err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error parsing trust policy document for role %s: %v", aws.StringValue(role.RoleName), err))
		return findings
	}

	// Check if the trust policy allows any AWS account to assume the role
	for _, statement := range policyDoc.Statement {
		if statement.Principal != nil && statement.Principal.AWS == "*" {
			findings = append(findings, fmt.Sprintf("WARNING: Role %s has a trust policy allowing ANY AWS account to assume it. Cross-account access should be restricted.", aws.StringValue(role.RoleName)))
		}
	}

	return findings
}

// Check for privilege escalation risks
func (scanner *IAMScanner) CheckPrivilegeEscalation(ctx context.Context, user *iam.User) []string {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/CheckPrivilegeEscalation")
	ctx, span := tracer.Start(ctx, "CheckPrivilegeEscalation")
	defer span.End()

	var findings []string

	// List all attached user policies
	policies, err := scanner.IAMClient.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: user.UserName,
	})
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error listing policies for user %s: %v", aws.StringValue(user.UserName), err))
		return findings
	}

	// Iterate through each policy to check for privilege escalation risks
	for _, policy := range policies.AttachedPolicies {
		// Retrieve the policy details
		policyDetail, err := scanner.IAMClient.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error retrieving policy %s for user %s: %v", aws.StringValue(policy.PolicyName), aws.StringValue(user.UserName), err))
			continue
		}

		// Retrieve the default policy version
		version, err := scanner.IAMClient.GetPolicyVersion(&iam.GetPolicyVersionInput{
			PolicyArn: policy.PolicyArn,
			VersionId: policyDetail.Policy.DefaultVersionId,
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error retrieving policy version %s: %v", aws.StringValue(policyDetail.Policy.DefaultVersionId), err))
			continue
		}

		// Decode the policy document
		decodedDocument, err := url.QueryUnescape(*version.PolicyVersion.Document)
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error decoding policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		// Parse the policy document
		var policyDoc PolicyDocument
		err = json.Unmarshal([]byte(decodedDocument), &policyDoc)
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error parsing policy document for policy %s: %v", aws.StringValue(policy.PolicyName), err))
			continue
		}

		// Check each statement for actions that may lead to privilege escalation
		for _, statement := range policyDoc.Statement {
			// Handle Action as string or []string
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
				if action == "iam:PassRole" || action == "sts:AssumeRole" {
					findings = append(findings, fmt.Sprintf("WARNING: User %s has the ability to escalate privileges using %s action.", aws.StringValue(user.UserName), action))
				}
			}
		}
	}

	return findings
}

// Check MFA configuration for all users
func (scanner *IAMScanner) CheckUserMFAStatus(ctx context.Context, user *iam.User) []string {
	// Start a new OpenTelemetry span for tracing
	tracer := otel.Tracer("iamscanner/CheckUserMFAStatus")
	ctx, span := tracer.Start(ctx, "CheckUserMFAStatus")
	defer span.End()

	var findings []string

	// List MFA devices for the given user
	mfaDevices, err := scanner.IAMClient.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: user.UserName,
	})
	if err != nil {
		span.RecordError(err)
		findings = append(findings, fmt.Sprintf("Error listing MFA devices for user %s: %v", aws.StringValue(user.UserName), err))
		return findings
	}

	// If no MFA devices are found, add a warning
	if len(mfaDevices.MFADevices) == 0 {
		findings = append(findings, fmt.Sprintf("WARNING: MFA is not enabled for user %s.", aws.StringValue(user.UserName)))
	}

	return findings
}
