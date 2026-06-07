package runtime

import (
	"context"
	"fmt"
)

// Adapter is the interface for all runtime adapters
type Adapter interface {
	Create(ctx context.Context, spec *InstanceSpec) error
	Delete(ctx context.Context, id string) error
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Status(ctx context.Context, id string) (*InstanceStatus, error)
	List(ctx context.Context) ([]*InstanceStatus, error)
	Snapshot(ctx context.Context, id string, name string) error
	RestoreSnapshot(ctx context.Context, id string, name string) error
	DeleteSnapshot(ctx context.Context, id string, name string) error
	Migrate(ctx context.Context, id string, target string) error
	GetConsole(ctx context.Context, id string) (string, error)
	Exec(ctx context.Context, id string, command []string) error
}

// InstanceSpec defines instance specification
type InstanceSpec struct {
	Name      string
	Runtime   string
	Image     string
	Type      string // container, vm, microvm
	CPU       string
	Memory    string
	Storage   string
	DiskSize  string
	Network   string
	Profile   string
	AutoStart bool
	Ephemeral bool
	Config    map[string]string
	Devices   map[string]Device
}

// Device defines a device configuration
type Device struct {
	Type       string
	Properties map[string]string
}

// InstanceStatus defines instance status
type InstanceStatus struct {
	ID        string
	Name      string
	State     string
	Type      string
	Runtime   string
	Node      string
	IP        string
	InternalIP string
	PublicIP   string
	CreatedAt string
	StartedAt string
	Uptime    string
	Snapshots int
	Processes int
	CPUUsage  float64
	MemoryUsage float64
	DiskUsage float64
}

// Factory creates runtime adapters
type Factory struct {
	adapters map[string]Adapter
}

// NewFactory creates a new adapter factory
func NewFactory() *Factory {
	return &Factory{
		adapters: make(map[string]Adapter),
	}
}

// Register registers an adapter for a runtime type
func (f *Factory) Register(runtimeType string, adapter Adapter) {
	f.adapters[runtimeType] = adapter
}

// Get returns the adapter for a runtime type
func (f *Factory) Get(runtimeType string) (Adapter, error) {
	adapter, ok := f.adapters[runtimeType]
	if !ok {
		return nil, fmt.Errorf("unsupported runtime type: %s", runtimeType)
	}
	return adapter, nil
}

// SupportedRuntimes returns all supported runtime types
func (f *Factory) SupportedRuntimes() []string {
	var runtimes []string
	for rt := range f.adapters {
		runtimes = append(runtimes, rt)
	}
	return runtimes
}

// DefaultFactory creates a factory with all default adapters
func DefaultFactory() *Factory {
	f := NewFactory()
	f.Register("incus-lxc", NewIncusAdapter("unix:///var/lib/incus/unix.socket"))
	f.Register("incus-vm", NewIncusAdapter("unix:///var/lib/incus/unix.socket"))
	f.Register("kata", NewKataAdapter("/run/kata-containers/kata.sock"))
	f.Register("kata-clh", NewKataAdapter("/run/kata-containers/kata-clh.sock"))
	f.Register("kata-tee", NewKataAdapter("/run/kata-containers/kata-tee.sock"))
	f.Register("cube", NewCubeAdapter("http://cubesandbox-api:8443"))
	f.Register("kubevirt", NewKubeVirtAdapter(""))
	return f
}
