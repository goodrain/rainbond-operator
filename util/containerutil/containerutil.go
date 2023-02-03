package containerutil

import "os"

const (
	// ContainerRuntimeDocker docker runtime
	ContainerRuntimeDocker = "docker"
	// ContainerRuntimeContainerd containerd runtime
	ContainerRuntimeContainerd = "containerd"
)

func GetContainerRuntime() string {
	if os.Getenv("CONTAINER_RUNTIME") != "" {
		if os.Getenv("CONTAINER_RUNTIME") == ContainerRuntimeContainerd {
			return ContainerRuntimeContainerd
		}
		return ContainerRuntimeDocker
	}
	// The containerd runtime is used by default
	_, err := os.Stat("/var/run/docker.sock")
	if err != nil {
		// no such file or dir, Indicates that the user does not have containerd installed and uses docker as the runtime
		return ContainerRuntimeContainerd
	}
	return ContainerRuntimeDocker
}
