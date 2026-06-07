package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Instance is the Schema for the instances API
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSpec   `json:"spec,omitempty"`
	Status InstanceStatus `json:"status,omitempty"`
}

// InstanceSpec defines the desired state of Instance
type InstanceSpec struct {
	// Runtime specifies the container runtime to use
	// +kubebuilder:validation:Enum=incus-lxc;incus-vm;kata;kata-clh;kata-tee;cube;kubevirt
	Runtime string `json:"runtime"`

	// Image specifies the container image to use
	Image string `json:"image"`

	// Resources specifies resource limits and requests
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Network specifies network configuration
	Network NetworkConfiguration `json:"network,omitempty"`

	// Security specifies security context
	Security SecurityContext `json:"security,omitempty"`

	// Lifecycle specifies lifecycle hooks and policies
	Lifecycle LifecycleConfiguration `json:"lifecycle,omitempty"`

	// Storage specifies storage configuration
	Storage StorageConfiguration `json:"storage,omitempty"`

	// NodeSelector specifies node selection constraints
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations specifies tolerations for pod scheduling
	Tolerations []Toleration `json:"tolerations,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	// CPU specifies CPU requirements (e.g., "1", "2", "4")
	CPU string `json:"cpu,omitempty"`

	// Memory specifies memory requirements (e.g., "1GB", "4Gi")
	Memory string `json:"memory,omitempty"`

	// Storage specifies storage requirements
	Storage StorageResource `json:"storage,omitempty"`

	// GPU specifies GPU requirements
	GPU GPUResource `json:"gpu,omitempty"`
}

// StorageResource defines storage resource requirements
type StorageResource struct {
	// Size specifies the storage size (e.g., "10GB", "100Gi")
	Size string `json:"size,omitempty"`

	// Pool specifies the storage pool name
	Pool string `json:"pool,omitempty"`

	// Type specifies the storage type (e.g., "disk", "ceph", "nfs")
	Type string `json:"type,omitempty"`
}

// GPUResource defines GPU resource requirements
type GPUResource struct {
	// Count specifies the number of GPUs
	Count int `json:"count,omitempty"`

	// Type specifies the GPU type (e.g., "nvidia", "amd")
	Type string `json:"type,omitempty"`

	// MIG specifies MIG configuration
	MIG string `json:"mig,omitempty"`
}

// NetworkConfiguration defines network configuration
type NetworkConfiguration struct {
	// VPC specifies the VPC name
	VPC string `json:"vpc,omitempty"`

	// Subnet specifies the subnet CIDR
	Subnet string `json:"subnet,omitempty"`

	// SecurityGroups specifies security group IDs
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// PublicIP specifies whether to allocate a public IP
	PublicIP bool `json:"publicIP,omitempty"`

	// LoadBalancer specifies whether to use a load balancer
	LoadBalancer bool `json:"loadBalancer,omitempty"`

	// Ingress specifies ingress rules
	Ingress []IngressRule `json:"ingress,omitempty"`

	// Egress specifies egress rules
	Egress []EgressRule `json:"egress,omitempty"`
}

// IngressRule defines an ingress rule
type IngressRule struct {
	Protocol   string `json:"protocol"`
	Port       int    `json:"port"`
	SourceCIDR string `json:"sourceCIDR,omitempty"`
	SourceSG   string `json:"sourceSG,omitempty"`
}

// EgressRule defines an egress rule
type EgressRule struct {
	Protocol     string `json:"protocol"`
	Port         int    `json:"port"`
	DestinationCIDR string `json:"destinationCIDR,omitempty"`
}

// SecurityContext defines security context
type SecurityContext struct {
	// TEE specifies whether to use Trusted Execution Environment
	TEE bool `json:"tee,omitempty"`

	// Seccomp specifies whether to use seccomp profiles
	Seccomp bool `json:"seccomp,omitempty"`

	// AppArmor specifies whether to use AppArmor profiles
	AppArmor bool `json:"apparmor,omitempty"`

	// Isolation specifies the isolation level
	// +kubebuilder:validation:Enum=container;vm;microvm;tee
	Isolation string `json:"isolation,omitempty"`

	// Rootless specifies whether to run in rootless mode
	Rootless bool `json:"rootless,omitempty"`

	// ReadOnlyRootFilesystem specifies whether the root filesystem is read-only
	ReadOnlyRootFilesystem bool `json:"readOnlyRootFilesystem,omitempty"`
}

// LifecycleConfiguration defines lifecycle configuration
type LifecycleConfiguration struct {
	// AutoStart specifies whether to auto-start the instance
	AutoStart bool `json:"autoStart,omitempty"`

	// AutoRestart specifies whether to auto-restart the instance
	AutoRestart bool `json:"autoRestart,omitempty"`

	// Ephemeral specifies whether the instance is ephemeral
	Ephemeral bool `json:"ephemeral,omitempty"`

	// PreStop specifies pre-stop hooks
	PreStop []LifecycleHook `json:"preStop,omitempty"`

	// PostStart specifies post-start hooks
	PostStart []LifecycleHook `json:"postStart,omitempty"`
}

// LifecycleHook defines a lifecycle hook
type LifecycleHook struct {
	// Type specifies the hook type
	Type string `json:"type"`

	// Command specifies the command to execute
	Command []string `json:"command,omitempty"`

	// HTTP specifies the HTTP request to make
	HTTP *HTTPHook `json:"http,omitempty"`
}

// HTTPHook defines an HTTP lifecycle hook
type HTTPHook struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// StorageConfiguration defines storage configuration
type StorageConfiguration struct {
	// Volumes specifies additional volumes
	Volumes []VolumeSpec `json:"volumes,omitempty"`
}

// VolumeSpec defines a volume specification
type VolumeSpec struct {
	// Name specifies the volume name
	Name string `json:"name"`

	// Size specifies the volume size
	Size string `json:"size,omitempty"`

	// Pool specifies the storage pool
	Pool string `json:"pool,omitempty"`

	// Type specifies the volume type (e.g., "disk", "nfs", "ceph")
	Type string `json:"type,omitempty"`

	// MountPath specifies the mount path in the container
	MountPath string `json:"mountPath,omitempty"`

	// ReadOnly specifies whether the volume is read-only
	ReadOnly bool `json:"readOnly,omitempty"`
}

// Toleration defines a toleration for pod scheduling
type Toleration struct {
	Key      string `json:"key,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect,omitempty"`
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	// Phase specifies the current phase of the instance
	// +kubebuilder:validation:Enum=Pending;Creating;Running;Stopping;Stopped;Terminating;Terminated;Failed;Unknown
	Phase string `json:"phase,omitempty"`

	// Node specifies the node where the instance is running
	Node string `json:"node,omitempty"`

	// Runtime specifies the runtime handler being used
	Runtime string `json:"runtimeHandler,omitempty"`

	// IP specifies the primary IP address
	IP string `json:"ip,omitempty"`

	// InternalIP specifies the internal IP address
	InternalIP string `json:"internalIP,omitempty"`

	// PublicIP specifies the public IP address
	PublicIP string `json:"publicIP,omitempty"`

	// StartTime specifies when the instance started
	StartTime *time.Time `json:"startTime,omitempty"`

	// Uptime specifies the instance uptime
	Uptime string `json:"uptime,omitempty"`

	// Restarts specifies the number of restarts
	Restarts int32 `json:"restarts,omitempty"`

	// Conditions specifies the current conditions
	Conditions []InstanceCondition `json:"conditions,omitempty"`
}

// InstanceCondition defines an instance condition
type InstanceCondition struct {
	Type               string      `json:"type"`
	Status             string      `json:"status"`
	LastTransitionTime *time.Time  `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// InstanceList contains a list of Instance
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=inst;vm;ct
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtime`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.status.node`
// +kubebuilder:printcolumn:name="IP",type=string,JSONPath=`.status.ip`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Uptime",type=string,JSONPath=`.status.uptime`,priority=1
// +kubebuilder:printcolumn:name="Restarts",type=integer,JSONPath=`.status.restarts`,priority=1

// +kubebuilder:object:root=true
// InstanceList contains a list of Instance

func init() {
	// Register types with scheme
	// SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
