package terminator

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Implements the aws.Provider interface
func (config *Config) Retrieve() (credentials.Value, error) {
	return credentials.Value{
		AccessKeyID:     config.AWSAccessKeyID,
		SecretAccessKey: config.AWSSecretAccessKey,
	}, nil
}

// Implements the aws.Provider interface
func (config *Config) IsExpired() bool {
	return false
}

func (t *Terminator) newEC2Service() (*ec2.EC2, error) {
	creds := credentials.NewCredentials(t.config)
	config := aws.NewConfig().WithRegion(t.config.AWSRegion).WithCredentials(creds).WithMaxRetries(3)
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	return ec2.New(sess), nil
}

func (t *Terminator) monitorTerminatedEC2Instances() {
	svc, err := t.newEC2Service()
	if err != nil {
		logger("ERROR", args{"error": err})
		return
	}
	for range time.Tick(t.config.PollingInterval) {
		params := &ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("tag:UUID"),
					Values: []*string{aws.String("*")},
				},
				{
					Name:   aws.String("instance-state-name"),
					Values: []*string{aws.String("terminated"), aws.String("shutting-down")},
				},
			},
		}
		if t.config.DockerCloudNamespace != "" {
			params.Filters = append(params.Filters, &ec2.Filter{
				Name: aws.String("tag:Docker ID username"),
				Values: []*string{
					aws.String(t.config.DockerCloudNamespace),
				},
			})
		}
		resp, err := svc.DescribeInstances(params)
		if err != nil {
			logger("ERROR", args{"error": err})
			continue
		}
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				for _, tag := range instance.Tags {
					if *tag.Key == "UUID" {
						uuid := *tag.Value
						t.terminateDockerCloudNode(uuid)
					}
				}
			}
		}
	}
}

func (t *Terminator) terminateEC2Instance(uuid string) {
	// We may get delayed instructions to terminate previously terminated instances.
	if t.terminatedEC2s[uuid] {
		return
	}

	svc, err := t.newEC2Service()
	if err != nil {
		logger("ERROR", args{"uuid": uuid, "error": err})
		return
	}

	logger("INFO", args{"uuid": uuid, "message": "Terminating EC2 instance"})

	var instanceIDs []*string
	{ // Brackets just for scoping vars
		params := &ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("tag:UUID"),
					Values: []*string{
						aws.String(uuid),
					},
				},
			},
		}
		if t.config.DockerCloudNamespace != "" {
			params.Filters = append(params.Filters, &ec2.Filter{
				Name: aws.String("tag:Docker ID username"),
				Values: []*string{
					aws.String(t.config.DockerCloudNamespace),
				},
			})
		}
		resp, err := svc.DescribeInstances(params)
		if err != nil {
			logger("ERROR", args{"uuid": uuid, "error": err})
			return
		}
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				instanceIDs = append(instanceIDs, instance.InstanceId)
			}
		}
	}

	if len(instanceIDs) == 0 {
		t.markEC2InstanceAsTerminated(uuid)

		logger("INFO", args{"uuid": uuid, "message": "No EC2 instance found"})
		return
	}

	{ // Brackets just for scoping vars
		params := &ec2.TerminateInstancesInput{
			InstanceIds: instanceIDs,
		}
		// Shuts down one or more EC2 instances. This operation is idempotent; if you terminate
		// an instance more than once, each call succeeds.
		_, err := svc.TerminateInstances(params)
		if err != nil {
			logger("ERROR", args{"uuid": uuid, "error": err})
			return
		}
	}

	// Only need to attempt these requests once per UUID.
	t.markEC2InstanceAsTerminated(uuid)
}
