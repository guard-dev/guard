package iamscanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/iam"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (scanner *IAMScanner) ScanIAM(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("iamscanner/ScanIAM")
	ctx, span := tracer.Start(ctx, "ScanIAM")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting IAM security scan",
		zap.String("region", region),
		zap.String("account_id", scanner.accountId))

	var allFindings []string

	users, err := scanner.ListUsers(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to list IAM users: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.String("region", region),
			zap.Error(err))
		span.RecordError(err)
		return nil, nil, fmt.Errorf(errorMsg)
	}

	scanner.logger.Logger(ctx).Info("Retrieved IAM users",
		zap.Int("user_count", len(users)))
	span.SetAttributes(attribute.Int("user_count", len(users)))

	// Categorize findings by type
	var unusedAccessKeysFindings, overlyPermissivePoliciesFindings, privilegeEscalationFindings, mfaFindings, rootAccountFindings []string

	for _, user := range users {
		userName := aws.StringValue(user.UserName)
		scanner.logger.Logger(ctx).Debug("Scanning user",
			zap.String("user_name", userName))

		// Unused access keys
		unusedKeyFindings := scanner.CheckUnusedAccessKeys(ctx, user)
		unusedAccessKeysFindings = append(unusedAccessKeysFindings, unusedKeyFindings...)

		// Overly permissive policies
		permissivePolicyFindings := scanner.CheckOverlyPermissivePolicies(ctx, user)
		overlyPermissivePoliciesFindings = append(overlyPermissivePoliciesFindings, permissivePolicyFindings...)

		// Privilege escalation risks
		privilegeEscalationRiskFindings := scanner.CheckPrivilegeEscalation(ctx, user)
		privilegeEscalationFindings = append(privilegeEscalationFindings, privilegeEscalationRiskFindings...)

		// MFA status
		mfaStatusFindings := scanner.CheckUserMFAStatus(ctx, user)
		mfaFindings = append(mfaFindings, mfaStatusFindings...)

		scanner.logger.Logger(ctx).Debug("Completed user scan",
			zap.String("user_name", userName),
			zap.Int("unused_keys_findings", len(unusedKeyFindings)),
			zap.Int("permissive_policy_findings", len(permissivePolicyFindings)),
			zap.Int("privilege_escalation_findings", len(privilegeEscalationRiskFindings)),
			zap.Int("mfa_findings", len(mfaStatusFindings)))
	}

	// Root account findings
	scanner.logger.Logger(ctx).Info("Checking root account usage")
	rootAccountFindingsResult, err := scanner.CheckRootAccountUsage(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("Error checking root account usage: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		span.RecordError(err)
		rootAccountFindings = append(rootAccountFindings, errorMsg)
	} else {
		rootAccountFindings = append(rootAccountFindings, rootAccountFindingsResult...)
		scanner.logger.Logger(ctx).Info("Completed root account check",
			zap.Int("finding_count", len(rootAccountFindingsResult)))
	}

	// Role trust policy findings
	scanner.logger.Logger(ctx).Info("Checking role trust policies")
	var roleTrustFindings []string
	roles, err := scanner.IAMClient.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		errorMsg := fmt.Sprintf("Error listing roles: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		span.RecordError(err)
		roleTrustFindings = append(roleTrustFindings, errorMsg)
	} else {
		for _, role := range roles.Roles {
			roleName := aws.StringValue(role.RoleName)
			scanner.logger.Logger(ctx).Debug("Checking role trust policy",
				zap.String("role_name", roleName))
			roleTrustPolicyFindings := scanner.CheckRoleTrustPolicies(ctx, role)
			roleTrustFindings = append(roleTrustFindings, roleTrustPolicyFindings...)
		}
		scanner.logger.Logger(ctx).Info("Completed role trust policy checks",
			zap.Int("role_count", len(roles.Roles)),
			zap.Int("finding_count", len(roleTrustFindings)))
	}

	// Process and summarize findings
	scanner.logger.Logger(ctx).Info("Processing and summarizing findings")
	scanItems, err := scanner.processAndSummarizeFindings(
		ctx,
		"iam",
		region,
		scanner.accountId,
		unusedAccessKeysFindings,
		overlyPermissivePoliciesFindings,
		privilegeEscalationFindings,
		mfaFindings,
		roleTrustFindings,
		rootAccountFindings,
	)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to process and summarize findings: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		span.RecordError(err)
		return nil, nil, fmt.Errorf(errorMsg)
	}

	// Append all categorized findings
	allFindings = append(allFindings, unusedAccessKeysFindings...)
	allFindings = append(allFindings, overlyPermissivePoliciesFindings...)
	allFindings = append(allFindings, privilegeEscalationFindings...)
	allFindings = append(allFindings, mfaFindings...)
	allFindings = append(allFindings, roleTrustFindings...)
	allFindings = append(allFindings, rootAccountFindings...)

	totalFindings := len(allFindings)
	scanner.logger.Logger(ctx).Info("Completed IAM security scan",
		zap.String("region", region),
		zap.Int("total_findings", totalFindings),
		zap.Int("unused_keys_findings", len(unusedAccessKeysFindings)),
		zap.Int("permissive_policy_findings", len(overlyPermissivePoliciesFindings)),
		zap.Int("privilege_escalation_findings", len(privilegeEscalationFindings)),
		zap.Int("mfa_findings", len(mfaFindings)),
		zap.Int("role_trust_findings", len(roleTrustFindings)),
		zap.Int("root_account_findings", len(rootAccountFindings)))
	span.SetAttributes(attribute.Int("total_findings", totalFindings))

	return allFindings, scanItems, nil
}

func (scanner *IAMScanner) processAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
	accountId string,
	unusedKeys []string,
	overlyPermissivePolicies []string,
	privilegeEscalation []string,
	mfaStatus []string,
	roleTrustPolicies []string,
	rootAccountFindings []string,
) ([]postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("iamscanner/processAndSummarizeFindings")
	ctx, span := tracer.Start(ctx, "processAndSummarizeFindings")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Starting findings summarization",
		zap.String("service", service),
		zap.String("region", region),
		zap.String("account_id", accountId))

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Process unused keys findings
	if len(unusedKeys) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing unused keys findings",
			zap.Int("finding_count", len(unusedKeys)))

		unusedKeysSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, unusedKeys)
		if err != nil {
			errorMsg := fmt.Sprintf("Error summarizing unused keys: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			span.RecordError(err)
			return nil, fmt.Errorf(errorMsg)
		}

		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: unusedKeys,
			Title:    "Unused Access Keys",
			Summary:  unusedKeysSummary.Summary,
			Remedy:   unusedKeysSummary.Remedies,
			Commands: unusedKeysSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed unused keys findings",
			zap.Int("finding_count", len(unusedKeys)))
	}

	// Process overly permissive policies findings
	if len(overlyPermissivePolicies) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing permissive policies findings",
			zap.Int("finding_count", len(overlyPermissivePolicies)))

		permissivePoliciesSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, overlyPermissivePolicies)
		if err != nil {
			errorMsg := fmt.Sprintf("Error summarizing overly permissive policies: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			span.RecordError(err)
			return nil, fmt.Errorf(errorMsg)
		}

		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: overlyPermissivePolicies,
			Title:    "Overly Permissive Policies",
			Summary:  permissivePoliciesSummary.Summary,
			Remedy:   permissivePoliciesSummary.Remedies,
			Commands: permissivePoliciesSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed permissive policies findings",
			zap.Int("finding_count", len(overlyPermissivePolicies)))
	}

	// Process privilege escalation findings
	if len(privilegeEscalation) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing privilege escalation findings",
			zap.Int("finding_count", len(privilegeEscalation)))

		privilegeEscalationSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, privilegeEscalation)
		if err != nil {
			errorMsg := fmt.Sprintf("Error summarizing privilege escalation risks: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			span.RecordError(err)
			return nil, fmt.Errorf(errorMsg)
		}

		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: privilegeEscalation,
			Title:    "Privilege Escalation Risks",
			Summary:  privilegeEscalationSummary.Summary,
			Remedy:   privilegeEscalationSummary.Remedies,
			Commands: privilegeEscalationSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed privilege escalation findings",
			zap.Int("finding_count", len(privilegeEscalation)))
	}

	// Process MFA status findings
	if len(mfaStatus) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing MFA status findings",
			zap.Int("finding_count", len(mfaStatus)))

		mfaSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, mfaStatus)
		if err != nil {
			errorMsg := fmt.Sprintf("Error summarizing MFA status: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			span.RecordError(err)
			return nil, fmt.Errorf(errorMsg)
		}

		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: mfaStatus,
			Title:    "MFA Status",
			Summary:  mfaSummary.Summary,
			Remedy:   mfaSummary.Remedies,
			Commands: mfaSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed MFA status findings",
			zap.Int("finding_count", len(mfaStatus)))
	}

	// Process role trust policy findings
	if len(roleTrustPolicies) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing role trust policy findings",
			zap.Int("finding_count", len(roleTrustPolicies)))

		roleTrustSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, roleTrustPolicies)
		if err != nil {
			errorMsg := fmt.Sprintf("Error summarizing role trust policies: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			span.RecordError(err)
			return nil, fmt.Errorf(errorMsg)
		}

		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: roleTrustPolicies,
			Title:    "Role Trust Policies",
			Summary:  roleTrustSummary.Summary,
			Remedy:   roleTrustSummary.Remedies,
			Commands: roleTrustSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed role trust policy findings",
			zap.Int("finding_count", len(roleTrustPolicies)))
	}

	// Process root account findings
	if len(rootAccountFindings) > 0 {
		scanner.logger.Logger(ctx).Debug("Summarizing root account findings",
			zap.Int("finding_count", len(rootAccountFindings)))

		rootAccountSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, accountId, rootAccountFindings)
		if err != nil {
			errorMsg := fmt.Sprintf("Error summarizing root account findings: %v", err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.Error(err))
			span.RecordError(err)
			return nil, fmt.Errorf(errorMsg)
		}

		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: rootAccountFindings,
			Title:    "Root Account Usage",
			Summary:  rootAccountSummary.Summary,
			Remedy:   rootAccountSummary.Remedies,
			Commands: rootAccountSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Processed root account findings",
			zap.Int("finding_count", len(rootAccountFindings)))
	}

	scanner.logger.Logger(ctx).Info("Completed findings summarization",
		zap.Int("total_scan_items", len(scanItems)))
	span.SetAttributes(attribute.Int("total_scan_items", len(scanItems)))

	return scanItems, nil
}
