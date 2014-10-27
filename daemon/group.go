package daemon

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api"
	"github.com/docker/docker/nat"
	"github.com/docker/docker/runconfig"
	"github.com/docker/docker/utils"
)

func (daemon *Daemon) GroupsCreate(config *api.Group) error {
	if config.Name == "" {
		return fmt.Errorf("group name cannot be empty")
	}

	sorted, err := sortContainers(config.Containers)
	if err != nil {
		return err
	}
	config.Containers = sorted

	if err := daemon.createGroup(config); err != nil {
		if !os.IsExist(err) {
			return err
		}

		if err := daemon.updateGroup(config); err != nil {
			return err
		}
	}

	for _, c := range config.Containers {
		if err := daemon.updateGroupContainer(config.Name, c); err != nil {
			return err
		}
	}

	return nil
}

func (daemon *Daemon) GroupsStop(name string) error {
	containers, err := daemon.fetchGroupsContainers(name)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := c.Stop(10); err != nil {
			return err
		}
	}

	return nil
}

func (daemon *Daemon) GroupsKill(name string) error {
	containers, err := daemon.fetchGroupsContainers(name)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := c.Kill(); err != nil {
			return err
		}
	}

	return nil
}

func (daemon *Daemon) GroupsDelete(name string) error {
	containers, err := daemon.fetchGroupsContainers(name)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := daemon.eng.Job("rm", c.ID).Run(); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(filepath.Join(daemon.Config().Root, "groups", name)); err != nil {
		return err
	}

	if _, err := daemon.containerGraph.Purge(name); err != nil {
		return err
	}

	return nil
}

func (daemon *Daemon) createGroup(config *api.Group) error {
	var (
		groupsRoot = filepath.Join(daemon.Config().Root, "groups")
		groupDir   = filepath.Join(groupsRoot, config.Name)
	)

	if err := os.MkdirAll(groupsRoot, 0644); err != nil {
		return err
	}

	if err := os.Mkdir(groupDir, 0644); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(groupDir, "volumes"), 0644); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(groupDir, "config.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	config.Created = time.Now()

	if err := json.NewEncoder(f).Encode(config); err != nil {
		return err
	}

	if _, err := daemon.containerGraph.Set(config.Name, config.Name); err != nil {
		return fmt.Errorf("path: %s id: %s: %s", config.Name, config.Name, err)
	}

	return nil
}

func (daemon *Daemon) updateGroup(config *api.Group) error {
	oldConfig, err := daemon.fetchGroupConfig(config.Name)
	if err != nil {
		return err
	}

	config.Created = oldConfig.Created

	return daemon.updateGroupConfig(config)
}

func asRunConfig(groupDir string, c *api.Container) (*runconfig.Config, error) {
	r := &runconfig.Config{
		Image: c.Image,

		Cmd:        c.Cmd,
		Entrypoint: c.Entrypoint,
		Env:        c.Env,

		ExposedPorts: make(map[nat.Port]struct{}),
		Volumes:      make(map[string]struct{}),

		User:       c.User,
		WorkingDir: c.WorkingDir,
		Tty:        c.Tty,

		Memory:     c.Memory,
		MemorySwap: -1,
		CpuShares:  c.CpuShares,
		Cpuset:     c.Cpuset,
	}

	for _, p := range c.Ports {
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}

		r.ExposedPorts[nat.Port(fmt.Sprintf("%d/%s", p.Container, proto))] = struct{}{}
	}

	return r, nil
}

func sortContainers(containers []*api.Container) ([]*api.Container, error) {
	var (
		names      []string
		dependents = make(map[string][]string)
	)

	for _, c := range containers {
		names = append(names, c.Name)

		for _, link := range c.Links {
			target := strings.Split(link, ":")[0]
			dependents[target] = append(dependents[target], c.Name)
		}
	}

	sortedNames, err := utils.TopologicalSort(names, dependents)
	if err != nil {
		return []*api.Container{}, err
	}

	var sortedContainers []*api.Container

	for _, n := range sortedNames {
		for _, c := range containers {
			if c.Name == n {
				sortedContainers = append(sortedContainers, c)
			}
		}
	}

	return sortedContainers, nil
}

func hashPath(p string) string {
	h := md5.New()
	fmt.Fprint(h, p)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func (daemon *Daemon) updateGroupContainer(groupName string, c *api.Container) error {
	fullName := filepath.Join(groupName, c.Name)

	if container := daemon.Get(fullName); container != nil {
		if err := container.Kill(); err != nil {
			return err
		}

		if err := daemon.Destroy(container); err != nil {
			return err
		}
	}

	groupDir := filepath.Join(daemon.Config().Root, "groups", groupName)

	config, err := asRunConfig(groupDir, c)
	if err != nil {
		return err
	}

	container, _, err := daemon.CreateInGroup(config, c.Name, groupName)
	if err != nil {
		// TODO: atomic abort and cleanup??????
		return err
	}

	if err := container.WriteHostConfig(); err != nil {
		return err
	}

	if err := container.ToDisk(); err != nil {
		return err
	}

	log.Printf("created %s (%s)\n", fullName, container.ID)

	return nil
}

func (daemon *Daemon) fetchGroupConfig(name string) (*api.Group, error) {
	var config *api.Group

	f, err := os.Open(daemon.groupConfigPath(name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func (daemon *Daemon) updateGroupConfig(config *api.Group) error {
	f, err := os.OpenFile(daemon.groupConfigPath(config.Name), os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(config)
}

func (daemon *Daemon) groupConfigPath(name string) string {
	return filepath.Join(daemon.Config().Root, "groups", name, "config.json")
}

func (daemon *Daemon) fetchGroupsContainers(name string) (map[string]*Container, error) {
	config, err := daemon.fetchGroupConfig(name)
	if err != nil {
		return nil, err
	}

	containers := make(map[string]*Container)
	for _, cconfig := range config.Containers {
		c := daemon.Get(filepath.Join(config.Name, cconfig.Name))
		if c == nil {
			return nil, fmt.Errorf("container does not exist for group %s", cconfig.Name)
		}

		containers[cconfig.Name] = c
	}

	return containers, nil
}

func (daemon *Daemon) setGroupContainerVolumesConfig(groupName string, container *Container, ccfg *api.Container) error {
	groupDir := filepath.Join(daemon.Config().Root, "groups", groupName)
	volumesDir := filepath.Join(groupDir, "volumes", ccfg.Name)

	container.Volumes = make(map[string]string)
	container.VolumesRW = make(map[string]bool)

	makeHostPath := func(containerPath string) (string, error) {
		path := filepath.Join(volumesDir, hashPath(containerPath))

		if err := os.MkdirAll(path, 0755); err != nil {
			return "", err
		}

		return path, nil
	}

	// TODO: @crosbymichael this does not belong here
	for _, v := range ccfg.Volumes {
		if v.Host == "" {
			path, err := makeHostPath(v.Container)
			if err != nil {
				return err
			}
			v.Host = path
		}

		container.Volumes[v.Container] = v.Host

		if v.Mode != "RO" {
			container.VolumesRW[v.Container] = true
		}
	}

	for containerPath, _ := range container.Config.Volumes {
		if _, alreadyConfigured := container.Volumes[containerPath]; alreadyConfigured {
			continue
		}
		hostPath, err := makeHostPath(containerPath)
		if err != nil {
			return err
		}
		container.Volumes[containerPath] = hostPath
		container.VolumesRW[containerPath] = true
	}

	return nil
}

func (daemon *Daemon) setGroupContainerHostConfig(groupName string, container *Container, ccfg *api.Container) error {
	var hostConfig = &runconfig.HostConfig{}

	hostConfig.PortBindings = nat.PortMap{}

	for p := range container.Config.ExposedPorts {
	ports:
		for _, pp := range ccfg.Ports {
			if p.Int() == pp.Container {
				hostConfig.PortBindings[p] = append(hostConfig.PortBindings[p], nat.PortBinding{
					HostPort: strconv.Itoa(pp.Host),
				})

				break ports
			}
		}
	}

	for _, l := range ccfg.Links {
		parts := strings.Split(l, ":")

		if strings.Index(parts[0], "/") == -1 {
			parts[0] = fmt.Sprintf("%s/%s", groupName, parts[0])
		}

		if strings.Index(parts[0], "/") != 0 {
			parts[0] = "/" + parts[0]
		}

		if len(parts) == 1 {
			path := strings.Split(parts[0], "/")
			alias := path[len(path)-1]
			parts = append(parts, alias)
		}

		hostConfig.Links = append(hostConfig.Links, strings.Join(parts, ":"))
	}

	hostConfig.Privileged = ccfg.Privileged
	hostConfig.CapAdd = ccfg.CapAdd
	hostConfig.CapDrop = ccfg.CapDrop

	for _, d := range ccfg.Devices {
		hostConfig.Devices = append(hostConfig.Devices, runconfig.DeviceMapping{
			PathOnHost:        d.PathOnHost,
			PathInContainer:   d.PathInContainer,
			CgroupPermissions: d.CgroupPermissions,
		})
	}

	return daemon.setHostConfig(container, hostConfig)
}

func (daemon *Daemon) GroupsStart(name string) error {
	groupConfig, err := daemon.fetchGroupConfig(name)
	if err != nil {
		return err
	}

	containers, err := daemon.fetchGroupsContainers(name)
	if err != nil {
		return err
	}

	for _, ccfg := range groupConfig.Containers {
		container := containers[ccfg.Name]

		if err := daemon.setGroupContainerVolumesConfig(name, container, ccfg); err != nil {
			return err
		}

		if err := daemon.setGroupContainerHostConfig(name, container, ccfg); err != nil {
			return err
		}

		if err := container.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (daemon *Daemon) GroupsGet(name string) ([]*api.Group, error) {
	groupsRoot := filepath.Join(daemon.Config().Root, "groups")

	groups := []*api.Group{}
	files, err := ioutil.ReadDir(groupsRoot)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.Mode().IsDir() {
			if name == "" || name == file.Name() {
				groupRoot := filepath.Join(groupsRoot, file.Name())

				f, err := os.Open(filepath.Join(groupRoot, "config.json"))
				if err != nil {
					return nil, err
				}

				var group *api.Group
				if err := json.NewDecoder(f).Decode(&group); err != nil {
					f.Close()
					return nil, err
				}
				f.Close()

				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}
