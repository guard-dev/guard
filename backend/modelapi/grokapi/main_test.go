package grokapi

import (
	"context"
	"guarddev/logger"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSummarizeFindings(t *testing.T) {
	// Initialize logger
	logMiddleware := logger.Connect(logger.LoggerConnectProps{Production: false})
	ctx := context.Background()
	log := logMiddleware.Logger(ctx)

	// Create Grok client
	grokClient := Connect(context.Background(), GrokConnectProps{
		Logger: logMiddleware,
	})

	// Test data
	service := "s3"
	region := "us-east-1"
	accountId := "123456789012"
	findings := []string{
		"S3 bucket 'example-bucket' has public access enabled through its bucket policy",
		"S3 bucket 'example-bucket' does not have encryption enabled",
		"S3 bucket 'example-bucket' does not have versioning enabled",
	}

	// Make the API call
	log.Info("Making API call to Grok...")
	result, err := grokClient.SummarizeFindings(context.Background(), service, region, accountId, findings)

	// Assert no error occurred
	assert.NoError(t, err)

	// Assert the result is not nil
	assert.NotNil(t, result)

	log.Info("=== Test Results ===", zap.String("Summary", result.Summary), zap.String("Remediation", result.Remedies), zap.Strings("Commmands", result.Commands))
}
