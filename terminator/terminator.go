package terminator

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	PollingInterval    time.Duration
	DockerCloudAuth    string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
}

func ConfigFromEnv() *Config {
	pollingInterval, _ := time.ParseDuration(os.Getenv("POLLING_INTERVAL"))
	return &Config{
		PollingInterval:    pollingInterval,
		DockerCloudAuth:    os.Getenv("DOCKERCLOUD_AUTH"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:          os.Getenv("AWS_REGION"),
	}
}

func (config *Config) Validate() error {
	// TODO: These checks could be more nuanced

	if config == nil {
		return errors.New("nil config")
	}
	if config.PollingInterval == 0 {
		return errors.New("POLLING_INTERVAL invalid")
	}
	if strings.HasPrefix("Basic ", config.DockerCloudAuth) {
		return errors.New("DOCKERCLOUD_AUTH invalid")
	}
	if config.AWSAccessKeyID == "" {
		return errors.New("AWS_ACCESS_KEY_ID invalid")
	}
	if config.AWSSecretAccessKey == "" {
		return errors.New("AWS_SECRET_ACCESS_KEY invalid")
	}
	if config.AWSRegion == "" {
		return errors.New("AWS_REGION invalid")
	}

	return nil
}

func Start(config *Config) {
	if err := config.Validate(); err != nil {
		logger("FATAL", args{"error": err})
	}

	unreachableDockerCloudUUIDsCh := make(chan string)
	dockerCloudUUIDsToTerminateCh := make(chan string)
	errors := make(chan error)

	dc := &dockerCloud{config: config}
	go dc.monitorUnreachableNodes(unreachableDockerCloudUUIDsCh, errors)
	go dc.terminateNodes(dockerCloudUUIDsToTerminateCh, errors)

	ec2 := &awsEC2{config: config}
	go ec2.monitorTerminatedInstances(dockerCloudUUIDsToTerminateCh, errors)
	go ec2.terminateInstances(unreachableDockerCloudUUIDsCh, errors)

	for err := range errors {
		if e, ok := err.(Error); ok {
			logger("ERROR", e.args)
		} else {
			logger("ERROR", args{"error": err})
		}
	}
}
