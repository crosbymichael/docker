package daemon

import (
	"time"

	"github.com/docker/docker/pkg/archive"
)

// ContainerChanges returns a list of container fs changes
func (daemon *Daemon) ContainerChanges(name string) ([]archive.Change, error) {
	start := time.Now()
	container, err := daemon.GetContainer(name)
	if err != nil {
		return nil, err
	}

	container.Lock()
	defer container.Unlock()
	c, err := container.RWLayer.Changes()
	if err != nil {
		return nil, err
	}
	ContainerChangesTimer.UpdateSince(start)
	return c, nil
}
