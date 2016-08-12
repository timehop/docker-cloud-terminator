package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	// Be kind to devs and include line numbers with each log logsput.
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logger("INFO", args{"polling": os.Getenv("POLLING_INTERVAL"), "providers": os.Getenv("CLOUD_PROVIDERS")})

	pollingInterval, err := time.ParseDuration(os.Getenv("POLLING_INTERVAL"))
	if err != nil {
		logger("FATAL", args{"error": err})
	}
	cloudProviders := strings.Split(os.Getenv("CLOUD_PROVIDERS"), ",")

	for range time.Tick(pollingInterval) {
		nodes, err := fetchNodesByState("Unreachable")
		if err != nil {
			logger("ERROR", args{"error": err})
			continue
		}
		if len(nodes) == 0 {
			logger("INFO", args{"message": "No nodes in state Unreachable"})
			continue
		}
		for _, node := range nodes {
			logger("INFO", args{"node": *node.UUID, "message": "Terminating docker cloud node"})
			if err := terminateDockerCloudNode(node); err != nil {
				logger("ERROR", args{"node": *node.UUID, "error": err, "message": "Failed to terminate node"})
				continue
			}

			for _, provider := range cloudProviders {
				switch provider {
				case "AWS", "aws":
					logger("INFO", args{"node": *node.UUID, "message": "Terminating ec2 instance"})
					if err := terminateEC2Instance(node); err != nil {
						logger("ERROR", args{"node": *node.UUID, "error": err, "message": "Failed to terminate ec2 instance"})
						continue
					}
				}
			}
		}
	}
}

func fetchNodesByState(state string) ([]Node, error) {
	getNodeURL := fmt.Sprintf("https://cloud.docker.com/api/infra/v1/node/?state=%s", state)
	req, err := http.NewRequest("GET", getNodeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", os.Getenv("DOCKERCLOUD_AUTH"))
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var nodes NodesResponse
	if err := dec.Decode(&nodes); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		if nodes.Error != nil {
			err = nodes.Error
		}
		return nil, err
	}
	return nodes.Objects, nil
}

func terminateDockerCloudNode(node Node) error {
	if node.UUID == nil {
		return nil
	}

	deleteNodeURL := fmt.Sprintf("https://cloud.docker.com%s", *node.ResourceURI)
	req, err := http.NewRequest("DELETE", deleteNodeURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", os.Getenv("DOCKERCLOUD_AUTH"))
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var nodes NodesResponse
	if err := dec.Decode(&nodes); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		if nodes.Error != nil {
			err = nodes.Error
		}
		return err
	}
	logger("INFO", args{"node": *node.UUID, "message": "Docker Cloud node terminated"})
	return nil
}

// Requires the following to be found in the env:
// AWS_ACCESS_KEY_ID
// AWS_SECRET_ACCESS_KEY
// AWS_REGION
func terminateEC2Instance(node Node) error {
	if node.UUID == nil {
		return nil
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	svc := ec2.New(sess)

	var instanceIDs []string
	{
		params := &ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("tag:Docker-Cloud-UUID"),
					Values: []*string{
						aws.String(*node.UUID), // Required
					},
				},
			},
		}
		resp, err := svc.DescribeTags(params)
		if err != nil {
			return err
		}
		for _, tag := range resp.Tags {
			instanceIDs = append(instanceIDs, *tag.ResourceId)
		}
	}

	{
		for _, instanceID := range instanceIDs {
			params := &ec2.TerminateInstancesInput{
				InstanceIds: []*string{ // Required
					aws.String(instanceID), // Required
				},
			}
			// Shuts down one or more EC2 instances. This operation is idempotent; if you terminate
			// an instance more than once, each call succeeds.
			_, err := svc.TerminateInstances(params)
			if err != nil {
				return err
			}
			logger("INFO", args{"node": *node.UUID, "message": "EC2 instance terminated", "instance_id": instanceID})
		}
	}
	return nil
}

// Shortcut
type args map[string]interface{}

// E.g.: logger("INFO", args{"uuid":*node.UUID, "error":err})
// Outputs: 'level'='INFO' 'uuid'='caa1dd48-bd8a-4bc0-907a-76fa0207ce33' 'error'='Not found'
func logger(level string, params args) {
	var logs []string
	for k, v := range params {
		logs = append(logs, fmt.Sprintf("%q=%q", k, v))
	}
	// Make the output consistent
	sort.Strings(logs)
	// Make sure that the line number is set from the calling stack frame
	log.Output(2, strings.ToUpper(level)+" "+strings.Join(logs, " "))
	if level == "FATAL" || level == "fatal" {
		os.Exit(1)
	}
}

type NodesResponse struct {
	Meta    *Meta  `json:"meta"`
	Objects []Node `json:"objects"`
	Error   *Error `json:"error"`
}

type Error string

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return string(*e)
}

type Meta struct {
	Limit      *int    `json:"limit"`
	Next       *string `json:"next"`
	Offset     *int    `json:"offset"`
	Previous   *string `json:"previous"`
	TotalCount *int    `json:"total_count"`
}

type PrivateIP struct {
	CIDR string `json:"cidr"`
	Name string `json:"name"`
}

type Tag struct {
	Name string `json:"name"`
}

type Node struct {
	AvailabilityZone     *string     `json:"availability_zone"`
	UUID                 *string     `json:"uuid"`
	ResourceURI          *string     `json:"resource_uri"`
	ExternalFDNQ         *string     `json:"external_fqdn"`
	State                *string     `json:"state"`
	NodeCluster          *string     `json:"node_cluster"`
	NodeType             *string     `json:"node_type"`
	Region               *string     `json:"region"`
	DockerExecdriver     *string     `json:"docker_execdriver"`
	DockerGraphdriver    *string     `json:"docker_graphdriver"`
	DockerVersion        *string     `json:"docker_version"`
	CPU                  *int        `json:"cpu"`
	Disk                 *int        `json:"disk"`
	Memory               *int        `json:"memory"`
	CurrentNumContainers *int        `json:"current_num_containers"`
	LastSeen             *string     `json:"last_seen"`
	PublicIP             *string     `json:"public_ip"`
	Tunnel               *string     `json:"tunnel"`
	DeployedDatetime     *string     `json:"deployed_datetime"`
	DestroyedDatetime    *string     `json:"destroyed_datetime"`
	Tags                 []Tag       `json:"tags"`
	Nickname             *string     `json:"nickname"`
	PrivateIPS           []PrivateIP `json:"private_ips"`
}
