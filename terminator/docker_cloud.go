package terminator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type NodesResponse struct {
	Meta    *Meta     `json:"meta"`
	Objects []Node    `json:"objects"`
	Error   *APIError `json:"error"`
}

type APIError string

func (e *APIError) Error() string {
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

type dockerCloud struct {
	config *Config
}

func (t *dockerCloud) monitorUnreachableNodes(unreachableDockerCloudUUIDsCh chan<- string, errs chan<- error) {
	for range time.Tick(t.config.PollingInterval) {
		logger("INFO", args{"message": "Polling for unreachable Docker Cloud nodes"})
		nodes, err := t.fetchNodesByState("Unreachable")
		if err != nil {
			errs <- err
		} else {
			for _, node := range nodes {
				unreachableDockerCloudUUIDsCh <- *node.UUID
			}
		}
	}
}

func (t *dockerCloud) terminateNodes(dockerCloudUUIDsToTerminateCh <-chan string, errs chan<- error) {
	for uuid := range dockerCloudUUIDsToTerminateCh {
		logger("INFO", args{"message": "Terminating Docker Cloud node", "uuid": uuid})
		err := t.terminateNode(uuid)
		if err != nil {
			errs <- err
		}
	}
}

func (t *dockerCloud) fetchNodesByState(state string) ([]Node, error) {
	getNodeURL := fmt.Sprintf("https://cloud.docker.com/api/infra/v1/node/?state=%s", state)
	req, err := http.NewRequest("GET", getNodeURL, nil)
	if err != nil {
		return nil, Error{args{"message": "Could not construct Docker Cloud node request", "request": "GET " + getNodeURL, "error": err}}
	}
	req.Header.Set("Authorization", t.config.DockerCloudAuth)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, Error{args{"message": "Could not do Docker Cloud node request", "request": "GET " + getNodeURL, "error": err}}
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var nodes NodesResponse
	if err := dec.Decode(&nodes); err != nil {
		return nil, Error{args{"message": "Could not decode Docker Cloud node request", "request": "GET " + getNodeURL, "error": err}}
	}

	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		if nodes.Error != nil {
			err = nodes.Error
		}
		return nil, Error{args{"message": "Unexpected Docker Cloud node response", "request": "GET " + getNodeURL, "error": err}}
	}

	return nodes.Objects, nil
}

func (t *dockerCloud) terminateNode(uuid string) error {
	deleteNodeURL := fmt.Sprintf("https://cloud.docker.com/api/infra/v1/node/%s/", uuid)
	req, err := http.NewRequest("DELETE", deleteNodeURL, nil)
	if err != nil {
		return Error{args{"uuid": uuid, "message": "Could not construct Docker Cloud node request", "request": "DELETE " + deleteNodeURL, "error": err}}
	}
	req.Header.Set("Authorization", t.config.DockerCloudAuth)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Error{args{"uuid": uuid, "message": "Could not do Docker Cloud node request", "request": "DELETE " + deleteNodeURL, "error": err}}
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var nodes NodesResponse
	if err := dec.Decode(&nodes); err != nil {
		return Error{args{"message": "Could not decode Docker Cloud node request", "request": "DELETE " + deleteNodeURL, "error": err}}
	}

	// TODO: Do not return errors if we have already terminated this node before, which can
	// happen if an EC2 instances reports a terminated state for a while.
	if resp.StatusCode != http.StatusAccepted {
		err := errors.New(http.StatusText(resp.StatusCode))
		if nodes.Error != nil {
			err = nodes.Error
		}
		return Error{args{"message": "Unexpected Docker Cloud node response", "request": "DELETE " + deleteNodeURL, "error": err}}
	}

	return nil
}
