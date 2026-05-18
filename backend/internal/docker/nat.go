package docker

import (
	"fmt"

	"github.com/docker/go-connections/nat"
)

// natExposedSet returns the Docker exposed-port set for a single TCP port.
func natExposedSet(port int) nat.PortSet {
	p, _ := nat.NewPort("tcp", fmt.Sprintf("%d", port))
	return nat.PortSet{p: struct{}{}}
}
