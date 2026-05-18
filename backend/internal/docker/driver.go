// Package docker is the runtime layer driver. It only knows how to manipulate
// Docker; it knows nothing about features, environments, or branches.
package docker

import (
	"context"
	"fmt"
	"io"

	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// Driver is a thin facade over the Docker SDK.
type Driver struct {
	C *client.Client
}

func New(host string) (*Driver, error) {
	c, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &Driver{C: c}, nil
}

// EnsureNetwork makes sure the named bridge network exists and returns its id.
// Idempotent.
func (d *Driver) EnsureNetwork(ctx context.Context, name string) (string, error) {
	list, err := d.C.NetworkList(ctx, dtypes.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return "", err
	}
	for _, n := range list {
		if n.Name == name {
			return n.ID, nil
		}
	}
	res, err := d.C.NetworkCreate(ctx, name, dtypes.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{"open-orch/env": name},
	})
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

func (d *Driver) RemoveNetwork(ctx context.Context, name string) error {
	list, err := d.C.NetworkList(ctx, dtypes.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return err
	}
	for _, n := range list {
		if n.Name == name {
			if err := d.C.NetworkRemove(ctx, n.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// RunSpec is everything Docker needs to start one service in an env.
type RunSpec struct {
	Name        string            // container name
	Image       string            // image ref
	Network     string            // docker network
	EnvVars     map[string]string
	Labels      map[string]string // Traefik labels live here
	ExposePort  int               // 0 if no port
	HealthCmd   []string          // optional override
}

func (d *Driver) RunContainer(ctx context.Context, spec RunSpec) (string, error) {
	// Reconcile-friendly: remove any existing container with this name.
	if err := d.RemoveByName(ctx, spec.Name); err != nil {
		return "", err
	}

	envs := make([]string, 0, len(spec.EnvVars))
	for k, v := range spec.EnvVars {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	cfg := &container.Config{
		Image:  spec.Image,
		Env:    envs,
		Labels: spec.Labels,
	}
	if spec.ExposePort > 0 {
		cfg.ExposedPorts = natExposedSet(spec.ExposePort)
	}
	if len(spec.HealthCmd) > 0 {
		cfg.Healthcheck = &container.HealthConfig{Test: spec.HealthCmd}
	}

	hcfg := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	endpoints := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			spec.Network: {Aliases: []string{spec.Name}},
		},
	}

	created, err := d.C.ContainerCreate(ctx, cfg, hcfg, endpoints, nil, spec.Name)
	if err != nil {
		return "", fmt.Errorf("create container %s: %w", spec.Name, err)
	}
	if err := d.C.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return created.ID, fmt.Errorf("start container %s: %w", spec.Name, err)
	}
	return created.ID, nil
}

func (d *Driver) RemoveByName(ctx context.Context, name string) error {
	list, err := d.C.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return err
	}
	for _, c := range list {
		// names from API are prefixed with /
		for _, n := range c.Names {
			if n == "/"+name {
				_ = d.C.ContainerStop(ctx, c.ID, container.StopOptions{})
				if err := d.C.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// ContainerStatus reports running state and (if available) health.
type ContainerStatus struct {
	Exists  bool
	Running bool
	Health  string // healthy|unhealthy|starting|none
	ID      string
}

func (d *Driver) Inspect(ctx context.Context, name string) (ContainerStatus, error) {
	list, err := d.C.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil { return ContainerStatus{}, err }
	for _, c := range list {
		for _, n := range c.Names {
			if n == "/"+name {
				ins, err := d.C.ContainerInspect(ctx, c.ID)
				if err != nil { return ContainerStatus{}, err }
				health := "none"
				if ins.State != nil && ins.State.Health != nil {
					health = ins.State.Health.Status
				}
				return ContainerStatus{
					Exists:  true,
					Running: ins.State != nil && ins.State.Running,
					Health:  health,
					ID:      c.ID,
				}, nil
			}
		}
	}
	return ContainerStatus{}, nil
}

// PullImage fetches an image if it isn't already cached locally.
func (d *Driver) PullImage(ctx context.Context, ref string) error {
	rc, err := d.C.ImagePull(ctx, ref, dtypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

// ListContainersByLabel is used by reconciler to detect drift / stale ctrs.
func (d *Driver) ListByLabel(ctx context.Context, key, val string) ([]string, error) {
	list, err := d.C.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", key+"="+val)),
	})
	if err != nil { return nil, err }
	out := make([]string, 0, len(list))
	for _, c := range list {
		for _, n := range c.Names {
			out = append(out, n)
		}
	}
	return out, nil
}
