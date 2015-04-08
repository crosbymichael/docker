// +build daemon

package v2

import "github.com/docker/docker/daemon"

func InternalDaemon(d *daemon.Daemon) Docker {
	return &translation{
		d: d,
	}
}

// translation makes the in process docker daemon conform the the Docker interface
// for use with the server's dependencies and Client.
type translation struct {
	d *daemon.Daemon
}

func (t *translation) FetchContainers() ([]*Container, error) {
	containers := t.d.List()
	var out []*Container
	for _, c := range containers {
		out = append(out, t.containerToAPIObject(c))
	}
	return out, nil
}

func (t *translation) FetchContainer(id string) (*Container, error) {
	c, err := t.d.Get(id)
	if err != nil {
		return nil, err
	}
	return t.containerToAPIObject(c), nil
}

// containerToAPIObject converts docker's internal Container type to the correct API response
// so that internal objects have no impact on the API response allowing internal types to be
// refactored and API responses to be stable and versioned.
func (t *translation) containerToAPIObject(container *daemon.Container) *Container {
	c := &Container{
		ID:      container.ID,
		Image:   container.ImageID,
		Args:    append([]string{container.Path}, container.Args...),
		Created: container.Created,
		Name:    container.Name,
	}
	c.Kind = "container"
	c.Version = "2"
	return c
}
