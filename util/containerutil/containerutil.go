package containerutil

import "os"

const (
	// ContainerRuntimeDocker docker runtime
	ContainerRuntimeDocker = "docker"
	// ContainerRuntimeContainerd containerd runtime
	ContainerRuntimeContainerd = "containerd"
)

// GetContainerRuntime get container runtime
func GetContainerRuntime() string {
	if os.Getenv("CONTAINER_RUNTIME") != "" {
		if os.Getenv("CONTAINER_RUNTIME") == ContainerRuntimeDocker {
			return ContainerRuntimeDocker
		}
		return ContainerRuntimeContainerd
	}
	// The containerd runtime is used by default
	_, err := os.Stat("/run/containerd/containerd.sock")
	if err != nil {
		// no such file or dir, Indicates that the user does not have containerd installed and uses docker as the runtime
		return ContainerRuntimeDocker
	}
	return ContainerRuntimeContainerd
}
