package client

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api"
	"github.com/docker/docker/nat"
	"github.com/docker/docker/pkg/units"
)

type GroupContainer struct {
	Image string
	Build string

	Command     interface{}
	Entrypoint  interface{}
	Environment []string

	Ports   []string
	Volumes []string
	Links   []string

	User       string
	WorkingDir string `yaml:"working_dir"`
	Tty        bool

	Memory    string
	CpuShares int64  `yaml:"cpu_shares"`
	Cpuset    string `yaml:"cpu_set"`

	Privileged bool
	CapAdd     []string `yaml:"cap_add"`
	CapDrop    []string `yaml:"cap_drop"`
	Devices    []string
}

type GroupConfig struct {
	Name       string
	Containers map[string]*GroupContainer
}

type GroupContext struct {
	CurrentDirectory string
}

func (ctx *GroupContext) Expand(path string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		return filepath.Join(ctx.CurrentDirectory, path)
	}
}

// Transforms a GroupConfig (read from YAML) into an api.Group (for posting as JSON)
// Does not handle auto-build or auto-pull of images - see cli.transformGroupConfig
func preprocessGroupConfig(raw *GroupConfig, ctx *GroupContext) (*api.Group, error) {
	group := &api.Group{
		Name: raw.Name,
	}

	for containerName, c := range raw.Containers {
		container := &api.Container{
			Name:  containerName,
			Image: c.Image,

			CpuShares: c.CpuShares,
			Cpuset:    c.Cpuset,

			Privileged: c.Privileged,
			CapAdd:     c.CapAdd,
			CapDrop:    c.CapDrop,
		}

		cmd, ok := parseCommand(c.Command)
		if !ok {
			return nil, fmt.Errorf("%s: command must be either a string or list of strings", containerName)
		}
		container.Cmd = cmd

		entrypoint, ok := parseCommand(c.Entrypoint)
		if !ok {
			return nil, fmt.Errorf("%s: entrypoint must be either a string or list of strings", containerName)
		}
		container.Entrypoint = entrypoint

		container.Env = c.Environment

		if c.Memory != "" {
			ram, err := units.RAMInBytes(c.Memory)
			if err != nil {
				return nil, err
			}
			container.Memory = ram
		}

		_, portBindings, err := nat.ParsePortSpecs(c.Ports)
		if err != nil {
			return nil, err
		}

		for p, b := range portBindings {
			pp := &api.Port{
				Container: p.Int(),
				Proto:     p.Proto(),
			}

			if len(b) > 0 {
				// FIXME: support more than one
				bb := b[0]

				hp, err := strconv.Atoi(bb.HostPort)
				if err != nil {
					return nil, err
				}

				pp.Host = hp
			}

			container.Ports = append(container.Ports, pp)
		}

		for _, rawVolume := range c.Volumes {
			parts := strings.Split(rawVolume, ":")
			switch len(parts) {
			case 0:
				return nil, fmt.Errorf("invalid volume format %s", rawVolume)
			case 1:
				container.Volumes = append(container.Volumes, &api.Volume{
					Container: parts[0],
					Mode:      "rw",
				})
			case 2:
				container.Volumes = append(container.Volumes, &api.Volume{
					Container: parts[1],
					Host:      ctx.Expand(parts[0]),
					Mode:      "rw",
				})
			case 3:
				container.Volumes = append(container.Volumes, &api.Volume{
					Container: parts[1],
					Host:      ctx.Expand(parts[0]),
					Mode:      strings.ToUpper(parts[2]),
				})
			}
		}

		container.Links = c.Links

		container.User = c.User
		container.WorkingDir = c.WorkingDir
		container.Tty = c.Tty

		for _, rawDevice := range c.Devices {
			parts := strings.Split(rawDevice, ":")

			switch len(parts) {
			case 1:
				container.Devices = append(container.Devices, &api.Device{
					PathOnHost:        parts[0],
					PathInContainer:   parts[0],
					CgroupPermissions: "rwm",
				})
			case 2:
				container.Devices = append(container.Devices, &api.Device{
					PathOnHost:        parts[0],
					PathInContainer:   parts[1],
					CgroupPermissions: "rwm",
				})
			case 3:
				container.Devices = append(container.Devices, &api.Device{
					PathOnHost:        parts[0],
					PathInContainer:   parts[1],
					CgroupPermissions: parts[2],
				})
			default:
				return nil, fmt.Errorf("invalid device format %s", rawDevice)
			}
		}

		group.Containers = append(group.Containers, container)
	}

	return group, nil
}

func parseCommand(cmd interface{}) ([]string, bool) {
	switch v := cmd.(type) {
	case string:
		return []string{"/bin/sh", "-c", v}, true
	case []interface{}:
		return parseListOfStrings(v)
	case nil:
		return []string{}, true
	default:
		return []string{}, false
	}
}

func parseListOfStrings(v []interface{}) ([]string, bool) {
	list := []string{}

	for _, i := range v {
		str, ok := i.(string)
		if !ok {
			return []string{}, false
		}
		list = append(list, str)
	}

	return list, true
}

// Transforms a GroupConfig (read from YAML) into an api.Group (for posting as JSON)
func (cli *DockerCli) transformGroupConfig(raw *GroupConfig) (*api.Group, error) {
	pwd, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	group, err := preprocessGroupConfig(raw, &GroupContext{CurrentDirectory: pwd})
	if err != nil {
		return nil, err
	}

	for _, processedContainer := range group.Containers {
		c := raw.Containers[processedContainer.Name]

		tag, err := cli.resolveContainerConfigImageTag(group.Name, processedContainer.Name, c)
		if err != nil {
			return nil, err
		}

		processedContainer.Image = tag
	}

	return group, nil
}

func (cli *DockerCli) resolveContainerConfigImageTag(groupName string, containerName string, c *GroupContainer) (string, error) {
	if c.Build != "" {
		if c.Image != "" {
			return "", fmt.Errorf("%s specifies both 'build' and 'image'", containerName)
		}

		tag := fmt.Sprintf("%s-%s", groupName, containerName)

		imageExists, err := cli.checkImageExists(tag)
		if err != nil {
			return "", err
		}

		if !imageExists {
			if err := cli.build(c.Build, buildOptions{tag: tag}); err != nil {
				return "", err
			}
		}

		return tag, nil
	} else {
		imageExists, err := cli.checkImageExists(c.Image)
		if err != nil {
			return "", err
		}

		if !imageExists {
			if err := cli.pullImage(c.Image); err != nil {
				return "", err
			}
		}

		return c.Image, nil
	}
}

func (cli *DockerCli) checkImageExists(image string) (bool, error) {
	_, status, err := readBody(cli.call("GET", "/images/"+image+"/json", nil, false))
	if err == nil {
		return true, nil
	} else {
		if status == 404 {
			return false, nil
		} else {
			return false, err
		}
	}
}
