package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// IncusClient implements the Incus REST API client
type IncusClient struct {
	endpoint string
	client   *http.Client
	apiKey   string
	baseURL  string
}

// NewIncusClient creates a new Incus API client
func NewIncusClient(endpoint, apiKey string) *IncusClient {
	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := endpoint

	// Handle Unix socket
	if strings.HasPrefix(endpoint, "unix://") {
		socketPath := strings.TrimPrefix(endpoint, "unix://")
		client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		}
		baseURL = "http://localhost" // Unix socket doesn't use host
	}

	return &IncusClient{
		endpoint: endpoint,
		client:   client,
		apiKey:   apiKey,
		baseURL:  baseURL,
	}
}

// GetInstance retrieves an instance by name
func (c *IncusClient) GetInstance(ctx context.Context, name string) (*Instance, error) {
	body, err := c.get(ctx, fmt.Sprintf("/1.0/instances/%s", name))
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata Instance `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.Metadata, nil
}

// ListInstances returns all instances
func (c *IncusClient) ListInstances(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/1.0/instances")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Metadata, nil
}

// CreateInstance creates a new instance
func (c *IncusClient) CreateInstance(ctx context.Context, config map[string]interface{}) error {
	return c.post(ctx, "/1.0/instances", config)
}

// DeleteInstance deletes an instance
func (c *IncusClient) DeleteInstance(ctx context.Context, name string) error {
	return c.delete(ctx, fmt.Sprintf("/1.0/instances/%s", name))
}

// StartInstance starts an instance
func (c *IncusClient) StartInstance(ctx context.Context, name string) error {
	return c.put(ctx, fmt.Sprintf("/1.0/instances/%s/state", name),
		map[string]string{"action": "start", "timeout": "30"})
}

// StopInstance stops an instance
func (c *IncusClient) StopInstance(ctx context.Context, name string) error {
	return c.put(ctx, fmt.Sprintf("/1.0/instances/%s/state", name),
		map[string]string{"action": "stop", "timeout": "30", "force": "false"})
}

// RestartInstance restarts an instance
func (c *IncusClient) RestartInstance(ctx context.Context, name string) error {
	return c.put(ctx, fmt.Sprintf("/1.0/instances/%s/state", name),
		map[string]string{"action": "restart", "timeout": "30"})
}

// CreateSnapshot creates a snapshot
func (c *IncusClient) CreateSnapshot(ctx context.Context, name, snapName string) error {
	return c.post(ctx, fmt.Sprintf("/1.0/instances/%s/snapshots", name),
		map[string]string{"name": snapName})
}

// ListSnapshots returns all snapshots for an instance
func (c *IncusClient) ListSnapshots(ctx context.Context, name string) ([]string, error) {
	body, err := c.get(ctx, fmt.Sprintf("/1.0/instances/%s/snapshots", name))
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Metadata, nil
}

// RestoreSnapshot restores an instance from a snapshot
func (c *IncusClient) RestoreSnapshot(ctx context.Context, name, snapName string) error {
	return c.put(ctx, fmt.Sprintf("/1.0/instances/%s", name),
		map[string]string{"restore": snapName})
}

// DeleteSnapshot deletes a snapshot
func (c *IncusClient) DeleteSnapshot(ctx context.Context, name, snapName string) error {
	return c.delete(ctx, fmt.Sprintf("/1.0/instances/%s/snapshots/%s", name, snapName))
}

// GetStoragePool returns a storage pool by name
func (c *IncusClient) GetStoragePool(ctx context.Context, name string) (*StoragePool, error) {
	body, err := c.get(ctx, fmt.Sprintf("/1.0/storage-pools/%s", name))
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata StoragePool `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.Metadata, nil
}

// ListStoragePools returns all storage pools
func (c *IncusClient) ListStoragePools(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/1.0/storage-pools")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Metadata, nil
}

// CreateStoragePool creates a storage pool
func (c *IncusClient) CreateStoragePool(ctx context.Context, config map[string]interface{}) error {
	return c.post(ctx, "/1.0/storage-pools", config)
}

// DeleteStoragePool deletes a storage pool
func (c *IncusClient) DeleteStoragePool(ctx context.Context, name string) error {
	return c.delete(ctx, fmt.Sprintf("/1.0/storage-pools/%s", name))
}

// GetNetwork returns a network by name
func (c *IncusClient) GetNetwork(ctx context.Context, name string) (*Network, error) {
	body, err := c.get(ctx, fmt.Sprintf("/1.0/networks/%s", name))
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata Network `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.Metadata, nil
}

// ListNetworks returns all networks
func (c *IncusClient) ListNetworks(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/1.0/networks")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Metadata, nil
}

// CreateNetwork creates a network
func (c *IncusClient) CreateNetwork(ctx context.Context, config map[string]interface{}) error {
	return c.post(ctx, "/1.0/networks", config)
}

// DeleteNetwork deletes a network
func (c *IncusClient) DeleteNetwork(ctx context.Context, name string) error {
	return c.delete(ctx, fmt.Sprintf("/1.0/networks/%s", name))
}

// GetProfile returns a profile by name
func (c *IncusClient) GetProfile(ctx context.Context, name string) (*Profile, error) {
	body, err := c.get(ctx, fmt.Sprintf("/1.0/profiles/%s", name))
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata Profile `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.Metadata, nil
}

// ListProfiles returns all profiles
func (c *IncusClient) ListProfiles(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/1.0/profiles")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Metadata, nil
}

// CreateProfile creates a profile
func (c *IncusClient) CreateProfile(ctx context.Context, config map[string]interface{}) error {
	return c.post(ctx, "/1.0/profiles", config)
}

// DeleteProfile deletes a profile
func (c *IncusClient) DeleteProfile(ctx context.Context, name string) error {
	return c.delete(ctx, fmt.Sprintf("/1.0/profiles/%s", name))
}

// GetClusterMembers returns cluster members
func (c *IncusClient) GetClusterMembers(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/1.0/cluster/members")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata []string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Metadata, nil
}

// GetServerInfo returns server information
func (c *IncusClient) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	body, err := c.get(ctx, "/1.0")
	if err != nil {
		return nil, err
	}

	var response struct {
		Metadata ServerInfo `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.Metadata, nil
}

// HTTP helpers

func (c *IncusClient) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (c *IncusClient) post(ctx context.Context, path string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path,
		bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *IncusClient) put(ctx context.Context, path string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+path,
		bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *IncusClient) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Data types

type Instance struct {
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	Type         string                 `json:"type"`
	Architecture string                 `json:"architecture"`
	CreatedAt    string                 `json:"created_at"`
	Config       map[string]string      `json:"config"`
	Devices      map[string]interface{} `json:"devices"`
}

type StoragePool struct {
	Name   string            `json:"name"`
	Driver string            `json:"driver"`
	Config map[string]string `json:"config"`
}

type Network struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

type Profile struct {
	Name    string                 `json:"name"`
	Config  map[string]string      `json:"config"`
	Devices map[string]interface{} `json:"devices"`
}

type ServerInfo struct {
	APIVersion  string                 `json:"api_version"`
	Auth        string                 `json:"auth"`
	Config      map[string]interface{} `json:"config"`
	Environment map[string]interface{} `json:"environment"`
}
