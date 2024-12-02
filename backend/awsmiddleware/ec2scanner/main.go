package ec2scanner

import (
	"context"
	"fmt"
	"strings"

	"guarddev/logger"
	"guarddev/modelapi/geminiapi"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type EC2Scanner struct {
	ec2Client *ec2.EC2
	logger    *logger.LogMiddleware
	Gemini    *geminiapi.Gemini
	accountId string
}

// Initialize an EC2 scanner with assumed credentials
func NewEC2Scanner(accessKey, secretKey, sessionToken, region string, accountId string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *EC2Scanner {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			sessionToken,
		),
	}))
	return &EC2Scanner{
		ec2Client: ec2.New(sess),
		logger:    logger,
		Gemini:    gemini,
		accountId: accountId,
	}
}

// List all EC2 instances
func (scanner *EC2Scanner) ListInstances(ctx context.Context) ([]*ec2.Instance, error) {
	tracer := otel.Tracer("ec2scanner/ListInstances")
	ctx, span := tracer.Start(ctx, "ListInstances")
	defer span.End()

	scanner.logger.Logger(ctx).Info("Listing EC2 instances")

	result, err := scanner.ec2Client.DescribeInstances(nil)
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing EC2 instances: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	var instances []*ec2.Instance
	for _, reservation := range result.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	scanner.logger.Logger(ctx).Info("EC2 instances retrieved",
		zap.Int("instance_count", len(instances)))

	span.SetAttributes(attribute.Int("instance_count", len(instances)))
	return instances, nil
}

// Check if an EC2 instance has a public IP
func (scanner *EC2Scanner) HasPublicIP(ctx context.Context, instance *ec2.Instance) (bool, string) {
	tracer := otel.Tracer("ec2scanner/HasPublicIP")
	ctx, span := tracer.Start(ctx, "HasPublicIP")
	defer span.End()

	instanceID := aws.StringValue(instance.InstanceId)
	if instance.PublicIpAddress != nil {
		publicIP := aws.StringValue(instance.PublicIpAddress)
		finding := fmt.Sprintf("Instance %s has a public IP: %s", instanceID, publicIP)
		scanner.logger.Logger(ctx).Warn("Instance has public IP",
			zap.String("instance_id", instanceID),
			zap.String("public_ip", publicIP))
		span.SetAttributes(
			attribute.String("instance_id", instanceID),
			attribute.String("public_ip", publicIP),
		)
		return true, finding
	}

	scanner.logger.Logger(ctx).Info("Instance has no public IP",
		zap.String("instance_id", instanceID))
	span.SetAttributes(attribute.String("instance_id", instanceID))
	return false, ""
}

// Check if an EC2 instance's volumes are encrypted
func (scanner *EC2Scanner) AreVolumesEncrypted(ctx context.Context, instance *ec2.Instance) (bool, []string) {
	tracer := otel.Tracer("ec2scanner/AreVolumesEncrypted")
	ctx, span := tracer.Start(ctx, "AreVolumesEncrypted")
	defer span.End()

	var findings []string
	allEncrypted := true
	instanceID := aws.StringValue(instance.InstanceId)

	scanner.logger.Logger(ctx).Info("Checking volume encryption",
		zap.String("instance_id", instanceID),
		zap.Int("volume_count", len(instance.BlockDeviceMappings)))

	for _, blockDevice := range instance.BlockDeviceMappings {
		volumeId := blockDevice.Ebs.VolumeId
		volumes, err := scanner.ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{
			VolumeIds: []*string{volumeId},
		})
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error retrieving volume %s for instance %s: %v", aws.StringValue(volumeId), instanceID, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("volume_id", aws.StringValue(volumeId)),
				zap.Error(err))
			findings = append(findings, errorMsg)
			allEncrypted = false
			continue
		}

		for _, volume := range volumes.Volumes {
			volumeID := aws.StringValue(volume.VolumeId)
			if !aws.BoolValue(volume.Encrypted) {
				finding := fmt.Sprintf("Volume %s attached to instance %s is not encrypted", volumeID, instanceID)
				scanner.logger.Logger(ctx).Warn("Unencrypted volume detected",
					zap.String("instance_id", instanceID),
					zap.String("volume_id", volumeID))
				findings = append(findings, finding)
				allEncrypted = false
			} else {
				scanner.logger.Logger(ctx).Info("Volume is encrypted",
					zap.String("instance_id", instanceID),
					zap.String("volume_id", volumeID))
			}
		}
	}

	span.SetAttributes(
		attribute.String("instance_id", instanceID),
		attribute.Bool("all_volumes_encrypted", allEncrypted),
		attribute.Int("finding_count", len(findings)),
	)

	return allEncrypted, findings
}

// Check security groups for open ports (e.g., SSH, RDP)
func (scanner *EC2Scanner) CheckSecurityGroups(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckSecurityGroups")
	ctx, span := tracer.Start(ctx, "CheckSecurityGroups")
	defer span.End()

	var findings []string
	instanceID := aws.StringValue(instance.InstanceId)

	scanner.logger.Logger(ctx).Info("Checking security groups",
		zap.String("instance_id", instanceID),
		zap.Int("security_group_count", len(instance.SecurityGroups)))

	for _, sg := range instance.SecurityGroups {
		sgID := aws.StringValue(sg.GroupId)
		sgDetails, err := scanner.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{sg.GroupId},
		})
		if err != nil {
			span.RecordError(err)
			errorMsg := fmt.Sprintf("Error describing security group %s: %v", sgID, err)
			scanner.logger.Logger(ctx).Error(errorMsg,
				zap.String("security_group_id", sgID),
				zap.Error(err))
			findings = append(findings, errorMsg)
			continue
		}

		for _, securityGroup := range sgDetails.SecurityGroups {
			for _, permission := range securityGroup.IpPermissions {
				for _, ipRange := range permission.IpRanges {
					if aws.StringValue(ipRange.CidrIp) == "0.0.0.0/0" {
						finding := fmt.Sprintf("Security group %s allows open access to port %d from any IP address (0.0.0.0/0)", sgID, aws.Int64Value(permission.FromPort))
						scanner.logger.Logger(ctx).Warn("Open security group rule detected",
							zap.String("instance_id", instanceID),
							zap.String("security_group_id", sgID),
							zap.Int64("port", aws.Int64Value(permission.FromPort)),
							zap.String("cidr", aws.StringValue(ipRange.CidrIp)))
						findings = append(findings, finding)
					}
				}
			}
		}
	}

	span.SetAttributes(
		attribute.String("instance_id", instanceID),
		attribute.Int("finding_count", len(findings)),
	)

	return findings
}

// Check IAM roles attached to EC2 instances
func (scanner *EC2Scanner) CheckInstanceIAMRole(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckInstanceIAMRole")
	ctx, span := tracer.Start(ctx, "CheckInstanceIAMRole")
	defer span.End()

	var findings []string
	instanceID := aws.StringValue(instance.InstanceId)

	scanner.logger.Logger(ctx).Info("Checking IAM role",
		zap.String("instance_id", instanceID))

	if instance.IamInstanceProfile != nil {
		profileArn := aws.StringValue(instance.IamInstanceProfile.Arn)
		finding := fmt.Sprintf("Instance %s has an IAM instance profile attached: %s", instanceID, profileArn)
		scanner.logger.Logger(ctx).Info("IAM role found",
			zap.String("instance_id", instanceID),
			zap.String("profile_arn", profileArn))
		findings = append(findings, finding)
	} else {
		finding := fmt.Sprintf("WARNING: Instance %s does not have an IAM role attached", instanceID)
		scanner.logger.Logger(ctx).Warn("No IAM role attached",
			zap.String("instance_id", instanceID))
		findings = append(findings, finding)
	}

	span.SetAttributes(
		attribute.String("instance_id", instanceID),
		attribute.Bool("has_iam_role", instance.IamInstanceProfile != nil),
	)

	return findings
}

// Check for CloudWatch monitoring
func (scanner *EC2Scanner) CheckCloudWatchMonitoring(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckCloudWatchMonitoring")
	ctx, span := tracer.Start(ctx, "CheckCloudWatchMonitoring")
	defer span.End()

	var findings []string
	instanceID := aws.StringValue(instance.InstanceId)

	scanner.logger.Logger(ctx).Info("Checking CloudWatch monitoring",
		zap.String("instance_id", instanceID))

	if instance.Monitoring != nil && aws.StringValue(instance.Monitoring.State) == "enabled" {
		finding := fmt.Sprintf("CloudWatch monitoring is enabled for instance %s", instanceID)
		scanner.logger.Logger(ctx).Info("CloudWatch monitoring enabled",
			zap.String("instance_id", instanceID))
		findings = append(findings, finding)
	} else {
		finding := fmt.Sprintf("WARNING: CloudWatch monitoring is not enabled for instance %s", instanceID)
		scanner.logger.Logger(ctx).Warn("CloudWatch monitoring not enabled",
			zap.String("instance_id", instanceID))
		findings = append(findings, finding)
	}

	span.SetAttributes(
		attribute.String("instance_id", instanceID),
		attribute.Bool("monitoring_enabled", instance.Monitoring != nil && aws.StringValue(instance.Monitoring.State) == "enabled"),
	)

	return findings
}

// Check instance metadata version
func (scanner *EC2Scanner) CheckMetadataVersion(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckMetadataVersion")
	ctx, span := tracer.Start(ctx, "CheckMetadataVersion")
	defer span.End()

	var findings []string
	instanceID := aws.StringValue(instance.InstanceId)

	scanner.logger.Logger(ctx).Info("Checking metadata version",
		zap.String("instance_id", instanceID))

	if instance.MetadataOptions != nil && strings.ToLower(aws.StringValue(instance.MetadataOptions.HttpTokens)) == "required" {
		finding := fmt.Sprintf("Instance %s is using IMDSv2 for metadata access", instanceID)
		scanner.logger.Logger(ctx).Info("IMDSv2 enabled",
			zap.String("instance_id", instanceID))
		findings = append(findings, finding)
	} else {
		finding := fmt.Sprintf("WARNING: Instance %s is not using IMDSv2 for metadata access", instanceID)
		scanner.logger.Logger(ctx).Warn("IMDSv2 not enabled",
			zap.String("instance_id", instanceID))
		findings = append(findings, finding)
	}

	span.SetAttributes(
		attribute.String("instance_id", instanceID),
		attribute.Bool("imdsv2_enabled", instance.MetadataOptions != nil && strings.ToLower(aws.StringValue(instance.MetadataOptions.HttpTokens)) == "required"),
	)

	return findings
}

// Check for unassociated Elastic IPs
func (scanner *EC2Scanner) CheckUnassociatedElasticIPs(ctx context.Context) ([]string, error) {
	tracer := otel.Tracer("ec2scanner/CheckUnassociatedElasticIPs")
	ctx, span := tracer.Start(ctx, "CheckUnassociatedElasticIPs")
	defer span.End()

	var findings []string

	scanner.logger.Logger(ctx).Info("Checking for unassociated Elastic IPs")

	addresses, err := scanner.ec2Client.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		span.RecordError(err)
		errorMsg := fmt.Sprintf("Error describing Elastic IPs: %v", err)
		scanner.logger.Logger(ctx).Error(errorMsg,
			zap.Error(err))
		return nil, fmt.Errorf(errorMsg)
	}

	for _, address := range addresses.Addresses {
		publicIP := aws.StringValue(address.PublicIp)
		if address.AssociationId == nil {
			finding := fmt.Sprintf("Elastic IP %s is not associated with any instance", publicIP)
			scanner.logger.Logger(ctx).Warn("Unassociated Elastic IP found",
				zap.String("public_ip", publicIP))
			findings = append(findings, finding)
		} else {
			scanner.logger.Logger(ctx).Info("Elastic IP is associated",
				zap.String("public_ip", publicIP),
				zap.String("association_id", aws.StringValue(address.AssociationId)))
		}
	}

	span.SetAttributes(
		attribute.Int("total_elastic_ips", len(addresses.Addresses)),
		attribute.Int("unassociated_count", len(findings)),
	)

	return findings, nil
}
