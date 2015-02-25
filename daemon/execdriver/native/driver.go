// +build linux,cgo

package native

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/docker/pkg/reexec"
	sysinfo "github.com/docker/docker/pkg/system"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/apparmor"
	"github.com/docker/libcontainer/cgroups/systemd"
	"github.com/docker/libcontainer/configs"
	"github.com/docker/libcontainer/system"
	"github.com/docker/libcontainer/utils"
)

const (
	DriverName = "native"
	Version    = "0.2"
)

type driver struct {
	root             string
	initPath         string
	activeContainers map[string]libcontainer.Container
	machineMemory    int64
	factory          libcontainer.Factory
	sync.Mutex
}

func NewDriver(root, initPath string) (*driver, error) {
	meminfo, err := sysinfo.ReadMemInfo()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	// native driver root is at docker_root/execdriver/native. Put apparmor at docker_root
	if err := apparmor.InstallDefaultProfile(); err != nil {
		return nil, err
	}
	cgm := libcontainer.Cgroupfs
	if systemd.UseSystemd() {
		cgm = libcontainer.SystemdCgroups
	}
	log.Println(reexec.Self())
	f, err := libcontainer.New(root, cgm, libcontainer.InitPath(reexec.Self(), DriverName))
	if err != nil {
		return nil, err
	}
	return &driver{
		root:             root,
		initPath:         initPath,
		activeContainers: make(map[string]libcontainer.Container),
		machineMemory:    meminfo.MemTotal,
		factory:          f,
	}, nil
}

type execOutput struct {
	exitCode int
	err      error
}

func (d *driver) Run(c *execdriver.Command, pipes *execdriver.Pipes, startCallback execdriver.StartCallback) (execdriver.ExitStatus, error) {
	// take the Command and populate the libcontainer.Config from it
	container, err := d.createContainer(c)
	if err != nil {
		return execdriver.ExitStatus{ExitCode: -1}, err
	}

	//if err := d.createContainerRoot(c.ID); err != nil {
	//return execdriver.ExitStatus{ExitCode: -1}, err
	//}

	//if err := d.writeContainerFile(container, c.ID); err != nil {
	//return execdriver.ExitStatus{ExitCode: -1}, err
	//}

	var term execdriver.Terminal

	if c.ProcessConfig.Tty {
		rootuid, err := container.HostUID()
		if err != nil {
			return execdriver.ExitStatus{ExitCode: -1}, err
		}
		term, err = NewTtyConsole(&c.ProcessConfig, pipes, rootuid)
	} else {
		term, err = execdriver.NewStdConsole(&c.ProcessConfig, pipes)
	}
	if err != nil {
		return execdriver.ExitStatus{ExitCode: -1}, err
	}
	c.ProcessConfig.Terminal = term

	cont, err := d.factory.Create(c.ID, container)
	if err != nil {
		return execdriver.ExitStatus{ExitCode: -1}, err
	}
	d.Lock()
	d.activeContainers[c.ID] = cont
	d.Unlock()
	defer func() {
		cont.Destroy()
		d.cleanContainer(c.ID)
	}()

	args := append([]string{c.ProcessConfig.Entrypoint}, c.ProcessConfig.Arguments...)
	p := &libcontainer.Process{
		Args:   append([]string{DriverName}, args...),
		Env:    c.ProcessConfig.Env,
		Stdin:  c.ProcessConfig.Stdin,
		Stdout: c.ProcessConfig.Stdout,
		Stderr: c.ProcessConfig.Stderr,
		Cwd:    c.ProcessConfig.Cmd.Dir,
		User:   c.ProcessConfig.User,
	}

	if err := cont.Start(p); err != nil {
		return execdriver.ExitStatus{ExitCode: -1}, err
	}

	if startCallback != nil {
		pid, err := p.Pid()
		if err != nil {
			p.Signal(os.Kill)
			p.Wait()
			return execdriver.ExitStatus{ExitCode: -1}, err
		}
		startCallback(&c.ProcessConfig, pid)
	}

	ps, err := p.Wait()
	if err != nil {
		p.Signal(os.Kill)
		p.Wait()
		return execdriver.ExitStatus{ExitCode: -1}, err
	}

	oomKill := false
	oomKillNotification, err := cont.NotifyOOM()
	if err == nil {
		_, oomKill = <-oomKillNotification
	} else {
		log.Warnf("WARNING: Your kernel does not support OOM notifications: %s", err)
	}

	return execdriver.ExitStatus{ExitCode: utils.ExitStatus(ps.Sys().(syscall.WaitStatus)), OOMKilled: oomKill}, nil
}

func (d *driver) Kill(p *execdriver.Command, sig int) error {
	if p.ProcessConfig.Process == nil {
		return errors.New("exec: not started")
	}
	return syscall.Kill(p.ProcessConfig.Process.Pid, syscall.Signal(sig))
}

func (d *driver) Pause(c *execdriver.Command) error {
	active := d.activeContainers[c.ID]
	if active == nil {
		return fmt.Errorf("active container for %s does not exist", c.ID)
	}
	return active.Pause()
}

func (d *driver) Unpause(c *execdriver.Command) error {
	active := d.activeContainers[c.ID]
	if active == nil {
		return fmt.Errorf("active container for %s does not exist", c.ID)
	}
	return active.Resume()
}

func (d *driver) Terminate(c *execdriver.Command) error {
	// lets check the start time for the process
	active := d.activeContainers[c.ID]
	if active == nil {
		return fmt.Errorf("active container for %s does not exist", c.ID)
	}
	state, err := active.State()
	if err != nil {
		return err
	}
	pid := state.InitProcessPid

	currentStartTime, err := system.GetProcessStartTime(pid)
	if err != nil {
		return err
	}

	if state.InitProcessStartTime == currentStartTime {
		err = syscall.Kill(pid, 9)
		syscall.Wait4(pid, nil, 0, nil)
	}
	d.cleanContainer(c.ID)

	return err

}

func (d *driver) Info(id string) execdriver.Info {
	return &info{
		ID:     id,
		driver: d,
	}
}

func (d *driver) Name() string {
	return fmt.Sprintf("%s-%s", DriverName, Version)
}

func (d *driver) GetPidsForContainer(id string) ([]int, error) {
	d.Lock()
	active := d.activeContainers[id]
	d.Unlock()

	if active == nil {
		return nil, fmt.Errorf("active container for %s does not exist", id)
	}
	return active.Processes()
}

func (d *driver) writeContainerFile(container *configs.Config, id string) error {
	data, err := json.Marshal(container)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(d.root, id, "container.json"), data, 0655)
}

func (d *driver) cleanContainer(id string) error {
	d.Lock()
	delete(d.activeContainers, id)
	d.Unlock()
	return os.RemoveAll(filepath.Join(d.root, id, "container.json"))
}

func (d *driver) createContainerRoot(id string) error {
	return os.MkdirAll(filepath.Join(d.root, id), 0655)
}

func (d *driver) Clean(id string) error {
	return os.RemoveAll(filepath.Join(d.root, id))
}

func (d *driver) Stats(id string) (*execdriver.ResourceStats, error) {
	c := d.activeContainers[id]
	now := time.Now()
	stats, err := c.Stats()
	if err != nil {
		return nil, err
	}
	memoryLimit := c.Config().Cgroups.Memory
	// if the container does not have any memory limit specified set the
	// limit to the machines memory
	if memoryLimit == 0 {
		memoryLimit = d.machineMemory
	}
	return &execdriver.ResourceStats{
		Stats:       stats,
		Read:        now,
		MemoryLimit: memoryLimit,
	}, nil
}

func getEnv(key string, env []string) string {
	for _, pair := range env {
		parts := strings.Split(pair, "=")
		if parts[0] == key {
			return parts[1]
		}
	}
	return ""
}

type TtyConsole struct {
	MasterPty libcontainer.Console
}

func NewTtyConsole(pc *execdriver.ProcessConfig, pipes *execdriver.Pipes, rootuid int) (*TtyConsole, error) {
	console, err := libcontainer.NewConsole(rootuid, rootuid)
	if err != nil {
		return nil, err
	}

	tty := &TtyConsole{
		MasterPty: console,
	}

	if err := tty.AttachPipes(pipes); err != nil {
		tty.Close()
		return nil, err
	}

	pc.Console = console.Path()

	return tty, nil
}

func (t *TtyConsole) Resize(h, w int) error {
	return term.SetWinsize(t.MasterPty.Fd(), &term.Winsize{Height: uint16(h), Width: uint16(w)})
}

func (t *TtyConsole) AttachPipes(pipes *execdriver.Pipes) error {
	go func() {
		if wb, ok := pipes.Stdout.(interface {
			CloseWriters() error
		}); ok {
			defer wb.CloseWriters()
		}

		io.Copy(pipes.Stdout, t.MasterPty)
	}()

	if pipes.Stdin != nil {
		go func() {
			io.Copy(t.MasterPty, pipes.Stdin)

			pipes.Stdin.Close()
		}()
	}

	return nil
}

func (t *TtyConsole) Close() error {
	return t.MasterPty.Close()
}
