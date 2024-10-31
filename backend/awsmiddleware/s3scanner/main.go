package s3scanner

import (
	"context"
	"fmt"
	"guarddev/logger"
	"guarddev/modelapi/geminiapi"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type S3Scanner struct {
	s3Client *s3.S3
	logger   *logger.LogMiddleware
	Gemini   *geminiapi.Gemini
}

// Initialize an S3 scanner with assumed credentials
func NewS3Scanner(accessKey, secretKey, sessionToken, region string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *S3Scanner {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			sessionToken,
		),
	}))
	return &S3Scanner{
		s3Client: s3.New(sess),
		logger:   logger,
		Gemini:   gemini,
	}
}

// List all S3 Buckets that are in the current region
func (scanner *S3Scanner) ListBuckets(ctx context.Context) ([]*s3.Bucket, error) {
	tracer := otel.Tracer("s3scanner/ListBuckets")
	ctx, span := tracer.Start(ctx, "ListBuckets")
	defer span.End()

	result, err := scanner.s3Client.ListBuckets(nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error listing S3 buckets: %v", err)
	}

	var bucketsInRegion []*s3.Bucket
	for _, bucket := range result.Buckets {
		bucketName := aws.StringValue(bucket.Name)
		bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
		if err != nil {
			// Log the error but continue to check other buckets
			scanner.logger.Logger(ctx).Warn(fmt.Sprintf("Error getting region for bucket %s: %v", bucketName, err))
			continue
		}
		if bucketRegion == *scanner.s3Client.Config.Region {
			bucketsInRegion = append(bucketsInRegion, bucket)
		} else {
			// Log that the bucket is skipped due to being in a different region
			scanner.logger.Logger(ctx).Info(fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region))
		}
	}

	span.SetAttributes(attribute.Int("bucket_count", len(bucketsInRegion)))
	return bucketsInRegion, nil
}

// Helper function to get the bucket region
func (scanner *S3Scanner) getBucketRegion(ctx context.Context, bucketName string) (string, error) {
	tracer := otel.Tracer("s3scanner/getBucketRegion")
	ctx, span := tracer.Start(ctx, "getBucketRegion")
	defer span.End()

	input := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	}
	result, err := scanner.s3Client.GetBucketLocation(input)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	region := aws.StringValue(result.LocationConstraint)
	if region == "" {
		region = "us-east-1"
	}
	return region, nil
}

// Check if an S3 bucket has public access (ACLs and Bucket Policies)
func (scanner *S3Scanner) IsBucketPublic(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/IsBucketPublic")
	ctx, span := tracer.Start(ctx, "IsBucketPublic")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	// Checking the bucket's policy status
	policyStatus, err := scanner.s3Client.GetBucketPolicyStatus(&s3.GetBucketPolicyStatusInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// If no bucket policy exists, move on to check ACL without logging an error
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchBucketPolicy" {
			// No policy means no public access via policy, so move on
		} else {
			span.RecordError(err)
			return false, fmt.Sprintf("Error fetching bucket policy for %s: %v", bucketName, err), nil
		}
	} else {
		if policyStatus.PolicyStatus != nil && aws.BoolValue(policyStatus.PolicyStatus.IsPublic) {
			return true, fmt.Sprintf("WARNING: Bucket %s is publicly accessible via policy.", bucketName), nil
		}
	}

	// Checking the bucket's ACL for public access
	acl, err := scanner.s3Client.GetBucketAcl(&s3.GetBucketAclInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error fetching ACL for bucket %s: %v", bucketName, err), err
	}

	for _, grant := range acl.Grants {
		if aws.StringValue(grant.Grantee.URI) == "http://acs.amazonaws.com/groups/global/AllUsers" {
			return true, fmt.Sprintf("WARNING: Bucket %s has public access via ACL.", bucketName), nil
		}
	}

	return false, fmt.Sprintf("Bucket %s is not publicly accessible.", bucketName), nil
}

// Check if bucket encryption is enabled
func (scanner *S3Scanner) IsEncryptionEnabled(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/IsEncryptionEnabled")
	ctx, span := tracer.Start(ctx, "IsEncryptionEnabled")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	result, err := scanner.s3Client.GetBucketEncryption(&s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == s3.ErrCodeNoSuchBucket {
				return false, fmt.Sprintf("Bucket %s not found.", bucketName), nil
			}
		}
		if awsErr, ok := err.(awserr.RequestFailure); ok {
			if awsErr.StatusCode() == 404 {
				return false, fmt.Sprintf("WARNING: Encryption not enabled on bucket %s.", bucketName), nil
			}
		}
		span.RecordError(err)
		return false, "", fmt.Errorf("error getting bucket encryption for %s: %v", bucketName, err)
	}

	if result.ServerSideEncryptionConfiguration != nil {
		return true, fmt.Sprintf("Bucket %s has encryption enabled.", bucketName), nil
	}

	return false, fmt.Sprintf("WARNING: Bucket %s does not have encryption enabled.", bucketName), nil
}

// Check if bucket versioning is enabled
func (scanner *S3Scanner) IsVersioningEnabled(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/IsVersioningEnabled")
	ctx, span := tracer.Start(ctx, "IsVersioningEnabled")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	versioning, err := scanner.s3Client.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		span.RecordError(err)
		return false, "", fmt.Errorf("error getting bucket versioning for %s: %v", bucketName, err)
	}

	if aws.StringValue(versioning.Status) == s3.BucketVersioningStatusEnabled {
		return true, fmt.Sprintf("Bucket %s has versioning enabled.", bucketName), nil
	}

	return false, fmt.Sprintf("WARNING: Bucket %s does not have versioning enabled.", bucketName), nil
}

// Check if bucket logging is enabled
func (scanner *S3Scanner) IsLoggingEnabled(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/IsLoggingEnabled")
	ctx, span := tracer.Start(ctx, "IsLoggingEnabled")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	logging, err := scanner.s3Client.GetBucketLogging(&s3.GetBucketLoggingInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		span.RecordError(err)
		return false, "", fmt.Errorf("error getting bucket logging for %s: %v", bucketName, err)
	}

	if logging.LoggingEnabled != nil {
		return true, fmt.Sprintf("Bucket %s has logging enabled.", bucketName), nil
	}

	return false, fmt.Sprintf("WARNING: Bucket %s does not have logging enabled.", bucketName), nil
}

// Check S3 Block Public Access settings
func (scanner *S3Scanner) CheckBlockPublicAccess(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/CheckBlockPublicAccess")
	ctx, span := tracer.Start(ctx, "CheckBlockPublicAccess")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	blockPublicAccess, err := scanner.s3Client.GetBucketPolicyStatus(&s3.GetBucketPolicyStatusInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// If no bucket policy exists, gracefully handle this case
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchBucketPolicy" {
			return true, fmt.Sprintf("Bucket %s does not have a policy, indicating it is not publicly accessible through policy.", bucketName), nil
		} else {
			span.RecordError(err)
			return false, "", fmt.Errorf("error checking block public access settings for bucket %s: %v", bucketName, err)
		}
	}

	if blockPublicAccess.PolicyStatus != nil && !aws.BoolValue(blockPublicAccess.PolicyStatus.IsPublic) {
		return true, fmt.Sprintf("Bucket %s has Block Public Access enabled.", bucketName), nil
	}

	return false, fmt.Sprintf("WARNING: Bucket %s does not have Block Public Access settings properly configured.", bucketName), nil
}

// Check if bucket lifecycle configuration is set
func (scanner *S3Scanner) IsLifecyclePolicyEnabled(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/IsLifecyclePolicyEnabled")
	ctx, span := tracer.Start(ctx, "IsLifecyclePolicyEnabled")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	lifecycle, err := scanner.s3Client.GetBucketLifecycleConfiguration(&s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.RequestFailure); ok && awsErr.StatusCode() == 404 {
			return false, fmt.Sprintf("WARNING: Bucket %s does not have a lifecycle policy configured.", bucketName), nil
		}
		span.RecordError(err)
		return false, "", fmt.Errorf("error getting lifecycle policy for bucket %s: %v", bucketName, err)
	}

	if lifecycle != nil {
		return true, fmt.Sprintf("Bucket %s has a lifecycle policy configured.", bucketName), nil
	}

	return false, fmt.Sprintf("WARNING: Bucket %s does not have a lifecycle policy configured.", bucketName), nil
}

// Check if MFA delete is enabled for bucket versioning
func (scanner *S3Scanner) IsMFADeleteEnabled(ctx context.Context, bucketName string) (bool, string, error) {
	tracer := otel.Tracer("s3scanner/IsMFADeleteEnabled")
	ctx, span := tracer.Start(ctx, "IsMFADeleteEnabled")
	defer span.End()

	// Get the bucket region and skip if it doesn't match the current client region
	bucketRegion, err := scanner.getBucketRegion(ctx, bucketName)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Sprintf("Error getting bucket region for %s: %v", bucketName, err), nil
	}
	if bucketRegion != *scanner.s3Client.Config.Region {
		return false, fmt.Sprintf("Skipping bucket %s as it is in region %s (current region: %s)", bucketName, bucketRegion, *scanner.s3Client.Config.Region), nil
	}

	versioning, err := scanner.s3Client.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		span.RecordError(err)
		return false, "", fmt.Errorf("error getting bucket versioning for %s: %v", bucketName, err)
	}

	if aws.StringValue(versioning.MFADelete) == "Enabled" {
		return true, fmt.Sprintf("Bucket %s has MFA delete enabled.", bucketName), nil
	}

	return false, fmt.Sprintf("WARNING: Bucket %s does not have MFA delete enabled.", bucketName), nil
}
