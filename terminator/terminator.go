package terminator

import (
	"errors"
	"os"
	"strings"
	"sync"
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

type Terminator struct {
	config *Config

	mu              sync.Mutex
	terminatedNodes map[string]bool
	terminatedEC2s  map[string]bool
}

func New(config *Config) *Terminator {
	return &Terminator{
		config:          config,
		terminatedNodes: map[string]bool{},
		terminatedEC2s:  map[string]bool{},
	}
}

func (t *Terminator) Start() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	go t.monitorUnreachableDockerCloudNodes()
	go t.monitorTerminatedDockerCloudNodes()
	go t.monitorTerminatedEC2Instances()

	for range time.Tick(t.config.PollingInterval) {
		logger("INFO", args{"message": "Polling for termination candidates"})
	}

	return nil
}

func (t *Terminator) markDockerCloudNodeAsTerminated(uuid string) {
	t.mu.Lock()
	t.terminatedNodes[uuid] = true
	t.mu.Unlock()
}

func (t *Terminator) markEC2InstanceAsTerminated(uuid string) {
	t.mu.Lock()
	t.terminatedEC2s[uuid] = true
	t.mu.Unlock()
}
