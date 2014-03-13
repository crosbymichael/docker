package native

import (
	"github.com/dotcloud/docker/execdriver"
	"testing"
)

func TestGetDefaultNamespaces(t *testing.T) {
	container := getDefaultTemplate()
	if container == nil {
		t.Fatal("container returned by default template should not be nil")
	}

	ns := make(map[string]bool)
	for _, n := range container.Namespaces {
		ns[n.Key] = true
	}

	for _, n := range []string{
		"NEWNET",
		"NEWIPC",
		"NEWNS",
		"NEWPID",
		"NEWUTS",
	} {
		if !ns[n] {
			t.Fatalf("expected namespace %s but was not found", n)
		}
	}
}

func TestSetHostNetworking(t *testing.T) {
	command := &execdriver.Command{
		ID: "test-container",
		Network: &execdriver.Network{
			UseHostNetworkStack: true,
		},
	}
	command.Env = []string{"HOSTNAME=test"}

	container := createContainer(command)
	for _, ns := range container.Namespaces {
		if ns.Key == "NEWNET" {
			t.Fatal("host networking should not container the NEWNET namespace")
		}
	}
	if len(container.Networks) > 0 {
		t.Fatal("container should not container any networking configuration")
	}
}
