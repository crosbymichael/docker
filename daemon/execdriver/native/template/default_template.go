package template

import (
	"github.com/docker/libcontainer/apparmor"
	"github.com/docker/libcontainer/configs"
)

// New returns the docker default configuration for libcontainer
func New() *configs.Config {
	container := &configs.Config{
		Capabilities: []string{
			"CHOWN",
			"DAC_OVERRIDE",
			"FSETID",
			"FOWNER",
			"MKNOD",
			"NET_RAW",
			"SETGID",
			"SETUID",
			"SETFCAP",
			"SETPCAP",
			"NET_BIND_SERVICE",
			"SYS_CHROOT",
			"KILL",
			"AUDIT_WRITE",
		},
		Namespaces: configs.Namespaces([]configs.Namespace{
			{Type: "NEWNS"},
			{Type: "NEWUTS"},
			{Type: "NEWIPC"},
			{Type: "NEWPID"},
			{Type: "NEWNET"},
		}),
		Cgroups: &configs.Cgroup{
			Parent:          "docker",
			AllowAllDevices: false,
		},
	}

	if apparmor.IsEnabled() {
		container.AppArmorProfile = "docker-default"
	}

	return container
}
