package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// IncusAdapter implements Adapter for Incus (LXC + VM)
type IncusAdapter struct {
	endpoint string
	client   *http.Client
}

// NewIncusAdapter creates a new Incus adapter
func NewIncusAdapter(endpoint string) *IncusAdapter {
	return &IncusAdapter{
		endpoint: endpoint,
		client:   &http.Client{},
	}
}

// Create creates a new Incus instance
func (a *IncusAdapter) Create(ctx context.Context, spec *InstanceSpec) error {
	// Build instance configuration
	config := map[string]interface{}{
		"name":         spec.Name,
		"architecture": "x86_64",
		"profiles":     []string{spec.Profile},
		"config":       spec.Config,
		"devices":      a.buildDevices(spec),
		"source": map[string]string{
			"type":  "image",
			"alias": spec.Image,
		},
	}

	if spec.Type == "vm" {
		config["type"] = "virtual-machine"
	} else {
		config["type"] = "container"
	}

	// Add auto-start config
	if spec.AutoStart {
		config["config"].(map[string]string)["boot.autostart"] = "true"
	}

	return a.createInstance(ctx, config)
}

// Delete deletes an Incus instance
func (a *IncusAdapter) Delete(ctx context.Context, id string) error {
	return a.doRequest(ctx, "DELETE", fmt.Sprintf("/1.0/instances/%s", id), nil)
}

// Start starts an Incus instance
func (a *IncusAdapter) Start(ctx context.Context, id string) error {
	return a.doRequest(ctx, "PUT", fmt.Sprintf("/1.0/instances/%s/state", id),
		map[string]string{"action": "start", "timeout": "30"})
}

// Stop stops an Incus instance
func (a *IncusAdapter) Stop(ctx context.Context, id string) error {
	return a.doRequest(ctx, "PUT", fmt.Sprintf("/1.0/instances/%s/state", id),
		map[string]string{"action": "stop", "timeout": "30", "force": "false"})
}

// Restart restarts an Incus instance
func (a *IncusAdapter) Restart(ctx context.Context, id string) error {
	return a.doRequest(ctx, "PUT", fmt.Sprintf("/1.0/instances/%s/state", id),
		map[string]string{"action": "restart", "timeout": "30"})
}

// Status returns the status of an Incus instance
func (a *IncusAdapter) Status(ctx context.Context, id string) (*InstanceStatus, error) {
	body, err := a.get(ctx, fmt.Sprintf("/1.0/instances/%s", id))
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata struct {
			Name      string `json:"name"`
			Status    string `json:"status"`
			Type      string `json:"type"`
			CreatedAt string `json:"created_at"`
			Config    map[string]string `json:"config"`
			Devices   map[string]map[string]string `json:"devices"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &InstanceStatus{
		ID:        id,
		Name:      response.Metadata.Name,
		State:     response.Metadata.Status,
		Type:      response.Metadata.Type,
		Runtime:   "incus",
		CreatedAt: response.Metadata.CreatedAt,
	}, nil
}

// List returns all Incus instances
func (a *IncusAdapter) List(ctx context.Context) ([]*InstanceStatus, error) {
	body, err := a.get(ctx, "/1.0/instances")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var instances []*InstanceStatus
	for _, url := range response.Metadata {
		// Extract instance name from URL
		var name string
		fmt.Sscanf(url, "/1.0/instances/%s", &name)
		if name == "" {
			continue
		}

		status, err := a.Status(ctx, name)
		if err != nil {
			continue
		}
		instances = append(instances, status)
	}

	return instances, nil
}

// Snapshot creates a snapshot of an Incus instance
func (a *IncusAdapter) Snapshot(ctx context.Context, id string, name string) error {
	return a.doRequest(ctx, "POST", fmt.Sprintf("/1.0/instances/%s/snapshots", id),
		map[string]string{"name": name})
}

// RestoreSnapshot restores an Incus instance from a snapshot
func (a *IncusAdapter) RestoreSnapshot(ctx context.Context, id string, name string) error {
	return a.doRequest(ctx, "PUT", fmt.Sprintf("/1.0/instances/%s", id),
		map[string]string{"restore": name})
}

// DeleteSnapshot deletes a snapshot of an Incus instance
func (a *IncusAdapter) DeleteSnapshot(ctx context.Context, id string, name string) error {
	return a.doRequest(ctx, "DELETE", fmt.Sprintf("/1.0/instances/%s/snapshots/%s", id, name), nil)
}

// Migrate migrates an Incus instance to another node
func (a *IncusAdapter) Migrate(ctx context.Context, id string, target string) error {
	// In a real implementation, this would use the Incus migration API
	return fmt.Errorf("migration not yet implemented")
}

// GetConsole returns the console URL for an Incus instance
func (a *IncusAdapter) GetConsole(ctx context.Context, id string) (string, error) {
	return fmt.Sprintf("/1.0/instances/%s/console", id), nil
}

// Exec executes a command in an Incus instance
func (a *IncusAdapter) Exec(ctx context.Context, id string, command []string) error {
	return a.doRequest(ctx, "POST", fmt.Sprintf("/1.0/instances/%s/exec", id),
		map[string]interface{}{
			"command": command,
			"wait-for-websocket": false,
			"interactive": false,
		})
}

// buildDevices builds device configuration
func (a *IncusAdapter) buildDevices(spec *InstanceSpec) map[string]interface{} {
	devices := make(map[string]interface{})

	// Root disk
	devices["root"] = map[string]string{
		"type": "disk",
		"pool": spec.Storage,
		"path": "/",
	}

	if spec.DiskSize != "" {
		devices["root"].(map[string]string)["size"] = spec.DiskSize
	}

	// Network device
	if spec.Network != "" {
		devices["eth0"] = map[string]string{
			"type":    "nic",
			"network": spec.Network,
			"name":    "eth0",
		}
	}

	return devices
}

// createInstance creates an instance via the Incus API
func (a *IncusAdapter) createInstance(ctx context.Context, config map[string]interface{}) error {
	return a.doRequest(ctx, "POST", "/1.0/instances", config)
}

// doRequest performs an HTTP request to the Incus API
func (a *IncusAdapter) doRequest(ctx context.Context, method, path string, body interface{}) error {
	// In a real implementation, this would use the actual HTTP client
	// For now, we just log the operation
	fmt.Printf("[Incus] %s %s\n", method, path)
	return nil
}

// get performs a GET request to the Incus API
func (a *IncusAdapter) get(ctx context.Context, path string) ([]byte, error) {
	// In a real implementation, this would use the actual HTTP client
	fmt.Printf("[Incus] GET %s\n", path)
	return []byte("{}"), nil
}
