package iamscanner

import (
	"context"
	"fmt"
	"guarddev/database/postgres"

	"github.com/aws/aws-sdk-go/service/iam"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (scanner *IAMScanner) ScanIAM(ctx context.Context, region string) ([]string, []postgres.CreateNewScanItemEntryParams, error) {
	tracer := otel.Tracer("iamscanner/ScanIAM")
	ctx, span := tracer.Start(ctx, "ScanIAM")
	defer span.End()

	var allFindings []string

	users, err := scanner.ListUsers(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	span.SetAttributes(attribute.Int("user_count", len(users)))

	// Categorize findings by type
	var unusedAccessKeysFindings, overlyPermissivePoliciesFindings, privilegeEscalationFindings, mfaFindings, rootAccountFindings []string

	for _, user := range users {
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
	}

	// Root account findings
	rootAccountFindingsResult, err := scanner.CheckRootAccountUsage(ctx)
	if err != nil {
		span.RecordError(err)
		rootAccountFindings = append(rootAccountFindings, fmt.Sprintf("Error checking root account usage: %v", err))
	} else {
		rootAccountFindings = append(rootAccountFindings, rootAccountFindingsResult...)
	}

	// Role trust policy findings
	var roleTrustFindings []string
	roles, err := scanner.IAMClient.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		span.RecordError(err)
		roleTrustFindings = append(roleTrustFindings, fmt.Sprintf("Error listing roles: %v", err))
	} else {
		for _, role := range roles.Roles {
			roleTrustPolicyFindings := scanner.CheckRoleTrustPolicies(ctx, role)
			roleTrustFindings = append(roleTrustFindings, roleTrustPolicyFindings...)
		}
	}

	// Process and summarize categorized findings using LLM
	scanItems, err := scanner.processAndSummarizeFindings(
		ctx,
		"iam",
		region,
		unusedAccessKeysFindings,
		overlyPermissivePoliciesFindings,
		privilegeEscalationFindings,
		mfaFindings,
		roleTrustFindings,
		rootAccountFindings,
	)
	if err != nil {
		span.RecordError(err)
		return nil, nil, err
	}

	// Append all categorized findings to allFindings
	allFindings = append(allFindings, unusedAccessKeysFindings...)
	allFindings = append(allFindings, overlyPermissivePoliciesFindings...)
	allFindings = append(allFindings, privilegeEscalationFindings...)
	allFindings = append(allFindings, mfaFindings...)
	allFindings = append(allFindings, roleTrustFindings...)

	span.SetAttributes(attribute.Int("total_findings", len(allFindings)))

	return allFindings, scanItems, nil
}

func (scanner *IAMScanner) processAndSummarizeFindings(
	ctx context.Context,
	service string,
	region string,
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

	scanItems := []postgres.CreateNewScanItemEntryParams{}

	// Define prompts for each type of finding and call the LLM
	if len(unusedKeys) > 0 {
		unusedKeysSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, unusedKeys)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing unused keys: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: unusedKeys,
			Title:    "Unused Access Keys",
			Summary:  unusedKeysSummary.Summary,
			Remedy:   unusedKeysSummary.Remedies,
			Commands: unusedKeysSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Unused Access Keys", zap.Any("Unused Access Keys", unusedKeysSummary))
	}

	if len(overlyPermissivePolicies) > 0 {
		permissivePoliciesSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, overlyPermissivePolicies)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing overly permissive policies: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: overlyPermissivePolicies,
			Title:    "Overly Permissive Policies",
			Summary:  permissivePoliciesSummary.Summary,
			Remedy:   permissivePoliciesSummary.Remedies,
			Commands: permissivePoliciesSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Overly Permissive Policies", zap.Any("Overly Permissive Policies", permissivePoliciesSummary))
	}

	if len(privilegeEscalation) > 0 {
		privilegeEscalationSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, privilegeEscalation)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing privilege escalation risks: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: privilegeEscalation,
			Title:    "Privilege Escalation Risks",
			Summary:  privilegeEscalationSummary.Summary,
			Remedy:   privilegeEscalationSummary.Remedies,
			Commands: privilegeEscalationSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Privilege Escalation Risks", zap.Any("Privilege Escalation Risks", privilegeEscalationSummary))
	}

	if len(mfaStatus) > 0 {
		mfaSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, mfaStatus)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing MFA status: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: mfaStatus,
			Title:    "MFA Status",
			Summary:  mfaSummary.Summary,
			Remedy:   mfaSummary.Remedies,
			Commands: mfaSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("MFA Status", zap.Any("MFA Status", mfaSummary))
	}

	if len(roleTrustPolicies) > 0 {
		roleTrustSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, roleTrustPolicies)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing role trust policies: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: roleTrustPolicies,
			Title:    "Role Trust Policies",
			Summary:  roleTrustSummary.Summary,
			Remedy:   roleTrustSummary.Remedies,
			Commands: roleTrustSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Role Trust Policies", zap.Any("Role Trust Policies", roleTrustSummary))
	}

	// Root account findings
	if len(rootAccountFindings) > 0 {
		rootAccountSummary, err := scanner.Gemini.SummarizeFindings(ctx, service, region, rootAccountFindings)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error summarizing root account findings: %v", err)
		}
		scanItems = append(scanItems, postgres.CreateNewScanItemEntryParams{
			Findings: rootAccountFindings,
			Title:    "Root Account Security",
			Summary:  rootAccountSummary.Summary,
			Remedy:   rootAccountSummary.Remedies,
			Commands: rootAccountSummary.Commands,
		})
		scanner.logger.Logger(ctx).Info("Root Account Security", zap.Any("Root Account Security", rootAccountSummary))
	}

	return scanItems, nil
}
