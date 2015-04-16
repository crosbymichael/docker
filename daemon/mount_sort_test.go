package daemon

import (
	"fmt"
	"testing"

	"github.com/docker/docker/daemon/execdriver"
)

func TestSortMountsByDepth(t *testing.T) {
	m := []execdriver.Mount{
		{
			Destination: "/etc",
		},
		{
			Destination: "/etc/docker",
		},
		{
			Destination: "/foo/bar",
		},
		{
			Destination: "/etc/docker/test",
		},
	}

	sortMounts(m)
	for _, ms := range m {
		fmt.Println(ms.Destination)
	}
}
