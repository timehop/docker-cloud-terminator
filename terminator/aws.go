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

type awsEC2 struct {
	config *Config
}

func (t *awsEC2) newEC2Service() (*ec2.EC2, error) {
	creds := credentials.NewCredentials(t.config)
	aws.NewConfig().WithRegion(t.config.AWSRegion).WithCredentials(creds)
	sess, err := session.NewSession(&aws.Config{
		MaxRetries: aws.Int(3),
	})
	if err != nil {
		return nil, err
	}
	return ec2.New(sess), nil
}

func (t *awsEC2) monitorTerminatedInstances(dockerCloudUUIDsToTerminateCh chan<- string, errs chan<- error) {
	svc, err := t.newEC2Service()
	if err != nil {
		errs <- err
		return
	}
	for range time.Tick(t.config.PollingInterval) {
		logger("INFO", args{"message": "Polling for terminated EC2 instances"})

		params := &ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("tag:Docker-Cloud-UUID"),
					Values: []*string{aws.String("*")},
				},
				{
					Name:   aws.String("instance-state-name"),
					Values: []*string{aws.String("terminated"), aws.String("shutting-down")},
				},
			},
		}
		resp, err := svc.DescribeInstances(params)
		if err != nil {
			errs <- err
			continue
		}
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				for _, tag := range instance.Tags {
					if *tag.Key == "Docker-Cloud-UUID" {
						dockerCloudUUIDsToTerminateCh <- *tag.Value
					}
				}
			}
		}
	}
}

func (t *awsEC2) terminateInstances(unreachableDockerCloudUUIDsCh <-chan string, errs chan<- error) {
	svc, err := t.newEC2Service()
	if err != nil {
		errs <- err
		return
	}
	for uuid := range unreachableDockerCloudUUIDsCh {
		logger("INFO", args{"message": "Terminating EC2 instance", "tag:Docker-Cloud-UUID": uuid})

		var instanceIDs []*string
		{
			params := &ec2.DescribeTagsInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("tag:Docker-Cloud-UUID"),
						Values: []*string{
							aws.String(uuid),
						},
					},
				},
			}
			resp, err := svc.DescribeTags(params)
			if err != nil {
				errs <- err
				continue
			}
			for _, tag := range resp.Tags {
				instanceIDs = append(instanceIDs, tag.ResourceId)
			}
		}
		{
			params := &ec2.TerminateInstancesInput{
				InstanceIds: instanceIDs,
			}
			// Shuts down one or more EC2 instances. This operation is idempotent; if you terminate
			// an instance more than once, each call succeeds.
			_, err := svc.TerminateInstances(params)
			if err != nil {
				errs <- err
				continue
			}
		}
	}
}
