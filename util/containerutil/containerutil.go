package containerutil

import "os"

const (
	// ContainerRuntimeDocker docker runtime
	ContainerRuntimeDocker = "docker"
	// ContainerRuntimeContainerd containerd runtime
	ContainerRuntimeContainerd = "containerd"
)

func GetContainerRuntime() string {
	// The containerd runtime is used by default
	if os.Getenv("CONTAINER_RUNTIME") == ContainerRuntimeDocker {
		return ContainerRuntimeDocker
	}
	return ContainerRuntimeContainerd
}
