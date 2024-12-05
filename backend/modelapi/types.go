package modelapi

import (
	"context"
	"guarddev/types"
)

// ModelAPI defines the interface that all model implementations must satisfy
type ModelAPI interface {
	// SummarizeFindings analyzes AWS resource findings and provides summary, remedies, and commands
	SummarizeFindings(ctx context.Context, service string, region string, accountId string, findings []string) (*types.SummarizeFindings, error)
}
