package runconfig

import (
	"testing"
)

func TestParseLxcConfOpt(t *testing.T) {
	opts := []string{"lxc.utsname=docker", "lxc.utsname = docker "}

	for _, o := range opts {
		k, v, err := parseLxcOpt(o)
		if err != nil {
			t.FailNow()
		}
		if k != "lxc.utsname" {
			t.Fail()
		}
		if v != "docker" {
			t.Fail()
		}
	}
}

func TestParseHostNetworkOpts(t *testing.T) {
	var (
		config     = &Config{}
		hostConfig = &HostConfig{}
	)

	if err := parseNetworkMode("host", config, hostConfig); err != nil {
		t.Fatal(err)
	}
	if !config.NetworkDisabled {
		t.Fatal("network should be disabled with host networking")
	}
	if !hostConfig.UseHostNetworkStack {
		t.Fatal("use host network stack should be true")
	}
}
