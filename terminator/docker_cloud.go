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

func (t *Terminator) monitorUnreachableDockerCloudNodes() {
	for range time.Tick(t.config.PollingInterval) {
		nodes, err := t.fetchNodesByState("Unreachable")
		if err != nil {
			logger("ERROR", args{"error": err})
		} else {
			for _, node := range nodes {
				t.terminateDockerCloudNode(*node.UUID)
			}
		}
	}
}

func (t *Terminator) monitorTerminatedDockerCloudNodes() {
	for range time.Tick(t.config.PollingInterval) {
		nodes, err := t.fetchNodesByState("Terminated")
		if err != nil {
			logger("ERROR", args{"error": err})
		} else {
			for _, node := range nodes {
				// These UUIDs can be safely ignored for termination by us
				t.markDockerCloudNodeAsTerminated(*node.UUID)

				t.terminateEC2Instance(*node.UUID)
			}
		}
	}
}

func (t *Terminator) terminateDockerCloudNode(uuid string) {
	// We may get delayed instructions to terminate previously terminated nodes.
	if t.terminatedNodes[uuid] {
		return
	}

	logger("INFO", args{"uuid": uuid, "message": "Terminating Docker Cloud node"})

	namespacePath := ""
	if t.config.DockerCloudNamespace != "" {
		namespacePath = t.config.DockerCloudNamespace + "/"
	}
	deleteNodeURL := fmt.Sprintf("https://cloud.docker.com/api/infra/v1/%snode/%s/", namespacePath, uuid)
	req, err := http.NewRequest("DELETE", deleteNodeURL, nil)
	if err != nil {
		logger("ERROR", args{"uuid": uuid, "error": err})
		return
	}
	req.Header.Set("Authorization", t.config.DockerCloudAuth)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger("ERROR", args{"uuid": uuid, "error": err})
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var nodes NodesResponse
	if err := dec.Decode(&nodes); err != nil {
		logger("ERROR", args{"uuid": uuid, "error": err})
		return
	}

	// Only attempt these requests once per UUID.
	t.markDockerCloudNodeAsTerminated(uuid)

	if resp.StatusCode != http.StatusAccepted {
		err := errors.New(http.StatusText(resp.StatusCode))
		if nodes.Error != nil {
			err = nodes.Error
		}
		logger("ERROR", args{"uuid": uuid, "error": err})
		return
	}

}

func (t *Terminator) fetchNodesByState(state string) ([]Node, error) {
	namespacePath := ""
	if t.config.DockerCloudNamespace != "" {
		namespacePath = t.config.DockerCloudNamespace + "/"
	}
	getNodeURL := fmt.Sprintf("https://cloud.docker.com/api/infra/v1/%snode/?state=%s", namespacePath, state)
	req, err := http.NewRequest("GET", getNodeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", t.config.DockerCloudAuth)
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
