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
)

type EC2Scanner struct {
	ec2Client *ec2.EC2
	logger    *logger.LogMiddleware
	Gemini    *geminiapi.Gemini
}

// Initialize an EC2 scanner with assumed credentials
func NewEC2Scanner(accessKey, secretKey, sessionToken, region string, logger *logger.LogMiddleware, gemini *geminiapi.Gemini) *EC2Scanner {
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
	}
}

// List all EC2 instances
func (scanner *EC2Scanner) ListInstances(ctx context.Context) ([]*ec2.Instance, error) {
	tracer := otel.Tracer("ec2scanner/ListInstances")
	ctx, span := tracer.Start(ctx, "ListInstances")
	defer span.End()

	result, err := scanner.ec2Client.DescribeInstances(nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing EC2 instances: %v", err)
	}

	var instances []*ec2.Instance
	for _, reservation := range result.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	span.SetAttributes(attribute.Int("instance_count", len(instances)))
	return instances, nil
}

// Check if an EC2 instance has a public IP
func (scanner *EC2Scanner) HasPublicIP(ctx context.Context, instance *ec2.Instance) (bool, string) {
	tracer := otel.Tracer("ec2scanner/HasPublicIP")
	ctx, span := tracer.Start(ctx, "HasPublicIP")
	defer span.End()

	if instance.PublicIpAddress != nil {
		return true, fmt.Sprintf("Instance %s has a public IP: %s", aws.StringValue(instance.InstanceId), aws.StringValue(instance.PublicIpAddress))
	}

	return false, ""
}

// Check if an EC2 instance's volumes are encrypted
func (scanner *EC2Scanner) AreVolumesEncrypted(ctx context.Context, instance *ec2.Instance) (bool, []string) {
	tracer := otel.Tracer("ec2scanner/AreVolumesEncrypted")
	ctx, span := tracer.Start(ctx, "AreVolumesEncrypted")
	defer span.End()

	var findings []string
	allEncrypted := true

	for _, blockDevice := range instance.BlockDeviceMappings {
		volumeId := blockDevice.Ebs.VolumeId
		volumes, err := scanner.ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{
			VolumeIds: []*string{volumeId},
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error retrieving volume %s for instance %s: %v", aws.StringValue(volumeId), aws.StringValue(instance.InstanceId), err))
			allEncrypted = false
			continue
		}

		for _, volume := range volumes.Volumes {
			if !aws.BoolValue(volume.Encrypted) {
				findings = append(findings, fmt.Sprintf("Volume %s attached to instance %s is not encrypted", aws.StringValue(volumeId), aws.StringValue(instance.InstanceId)))
				allEncrypted = false
			}
		}
	}

	return allEncrypted, findings
}

// Check security groups for open ports (e.g., SSH, RDP)
func (scanner *EC2Scanner) CheckSecurityGroups(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckSecurityGroups")
	ctx, span := tracer.Start(ctx, "CheckSecurityGroups")
	defer span.End()

	var findings []string

	for _, sg := range instance.SecurityGroups {
		sgDetails, err := scanner.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{sg.GroupId},
		})
		if err != nil {
			span.RecordError(err)
			findings = append(findings, fmt.Sprintf("Error describing security group %s: %v", aws.StringValue(sg.GroupId), err))
			continue
		}

		for _, securityGroup := range sgDetails.SecurityGroups {
			for _, permission := range securityGroup.IpPermissions {
				for _, ipRange := range permission.IpRanges {
					if aws.StringValue(ipRange.CidrIp) == "0.0.0.0/0" {
						findings = append(findings, fmt.Sprintf("Security group %s allows open access to port %d from any IP address (0.0.0.0/0)", aws.StringValue(sg.GroupId), aws.Int64Value(permission.FromPort)))
					}
				}
			}
		}
	}

	return findings
}

// Check IAM roles attached to EC2 instances
func (scanner *EC2Scanner) CheckInstanceIAMRole(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckInstanceIAMRole")
	ctx, span := tracer.Start(ctx, "CheckInstanceIAMRole")
	defer span.End()

	var findings []string

	if instance.IamInstanceProfile != nil {
		profileArn := aws.StringValue(instance.IamInstanceProfile.Arn)
		findings = append(findings, fmt.Sprintf("Instance %s has an IAM instance profile attached: %s", aws.StringValue(instance.InstanceId), profileArn))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: Instance %s does not have an IAM role attached", aws.StringValue(instance.InstanceId)))
	}

	return findings
}

// Check for CloudWatch monitoring
func (scanner *EC2Scanner) CheckCloudWatchMonitoring(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckCloudWatchMonitoring")
	ctx, span := tracer.Start(ctx, "CheckCloudWatchMonitoring")
	defer span.End()

	var findings []string

	if instance.Monitoring != nil && aws.StringValue(instance.Monitoring.State) == "enabled" {
		findings = append(findings, fmt.Sprintf("CloudWatch monitoring is enabled for instance %s", aws.StringValue(instance.InstanceId)))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: CloudWatch monitoring is not enabled for instance %s", aws.StringValue(instance.InstanceId)))
	}

	return findings
}

// Check instance metadata version
func (scanner *EC2Scanner) CheckMetadataVersion(ctx context.Context, instance *ec2.Instance) []string {
	tracer := otel.Tracer("ec2scanner/CheckMetadataVersion")
	ctx, span := tracer.Start(ctx, "CheckMetadataVersion")
	defer span.End()

	var findings []string

	if instance.MetadataOptions != nil && strings.ToLower(aws.StringValue(instance.MetadataOptions.HttpTokens)) == "required" {
		findings = append(findings, fmt.Sprintf("Instance %s is using IMDSv2 for metadata access", aws.StringValue(instance.InstanceId)))
	} else {
		findings = append(findings, fmt.Sprintf("WARNING: Instance %s is not using IMDSv2 for metadata access", aws.StringValue(instance.InstanceId)))
	}

	return findings
}

// Check for unassociated Elastic IPs
func (scanner *EC2Scanner) CheckUnassociatedElasticIPs(ctx context.Context) ([]string, error) {
	tracer := otel.Tracer("ec2scanner/CheckUnassociatedElasticIPs")
	ctx, span := tracer.Start(ctx, "CheckUnassociatedElasticIPs")
	defer span.End()

	var findings []string

	addresses, err := scanner.ec2Client.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("error describing Elastic IPs: %v", err)
	}

	for _, address := range addresses.Addresses {
		if address.AssociationId == nil {
			findings = append(findings, fmt.Sprintf("Elastic IP %s is not associated with any instance", aws.StringValue(address.PublicIp)))
		}
	}

	return findings, nil
}
