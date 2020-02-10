package v1

// ComponentStatus component status
type ComponentStatus string

const (
	//ComponentStatusRunning running
	ComponentStatusRunning = "Running"
	// ComponentStatusIniting initing
	ComponentStatusIniting = "Initing"
	//ComponentStatusCreating creating
	ComponentStatusCreating = "Creating"
	// ComponentStatusTerminating terminal
	ComponentStatusTerminating = "Terminating" // TODO fanyangyang暂未实现
	// ComponentStatusFailed failed
	ComponentStatusFailed = "Failed"
)

// RbdComponentStatus rainbond component status
type RbdComponentStatus struct {
	Name string `json:"name"`

	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas"`

	// Total number of ready pods targeted by this deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas"`

	Status          ComponentStatus `json:"status"` //根据PodStatuses总结汇总该组件的状态
	Message         string          `json:"message"`
	Reason          string          `json:"reason"`
	ISInitComponent bool            `json:"isInitComponent"`

	PodStatuses []PodStatus `json:"podStatus"`
}

// PodStatus represents information about the status of a pod, which belongs to RbdComponent.
type PodStatus struct {
	Name              string               `json:"name"`
	Phase             string               `json:"phase"`
	HostIP            string               `json:"hostIP"`
	Reason            string               `json:"reason"`
	Message           string               `json:"message"`
	ContainerStatuses []PodContainerStatus `json:"container_statuses"`
}

type PodContainerStatus struct {
	ContainerID string `json:"containerID"`
	Image       string `json:"image"`
	// Specifies whether the container has passed its readiness probe.
	Ready   bool   `json:"ready"`
	State   string `json:"state"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}
