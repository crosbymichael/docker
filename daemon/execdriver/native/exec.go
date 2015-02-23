// +build linux

package native

import (
	"fmt"
	"os"
	"syscall"

	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/utils"
)

// TODO(vishh): Add support for running in priviledged mode and running as a different user.
func (d *driver) Exec(c *execdriver.Command, processConfig *execdriver.ProcessConfig, pipes *execdriver.Pipes, startCallback execdriver.StartCallback) (int, error) {
	active := d.activeContainers[c.ID]
	if active == nil {
		return -1, fmt.Errorf("No active container exists with ID %s", c.ID)
	}

	var term execdriver.Terminal
	var err error

	if processConfig.Tty {
		config := active.Config()
		rootuid, err := config.HostUID()
		if err != nil {
			return -1, err
		}
		term, err = NewTtyConsole(processConfig, pipes, rootuid)
	} else {
		term, err = execdriver.NewStdConsole(processConfig, pipes)
	}
	if err != nil {
		return -1, err
	}

	processConfig.Terminal = term

	p := &libcontainer.Process{
		Args:   append([]string{c.ProcessConfig.Entrypoint}, c.ProcessConfig.Arguments...),
		Env:    c.ProcessConfig.Env,
		Stdin:  c.ProcessConfig.Stdin,
		Stdout: c.ProcessConfig.Stdout,
		Stderr: c.ProcessConfig.Stderr,
		Cwd:    c.ProcessConfig.Cmd.Dir,
		User:   c.ProcessConfig.User,
	}
	if err := active.Start(p); err != nil {
		return -1, err
	}

	if startCallback != nil {
		pid, err := p.Pid()
		if err != nil {
			p.Signal(os.Kill)
			p.Wait()
			return -1, err
		}
		startCallback(&c.ProcessConfig, pid)
	}

	ps, err := p.Wait()
	if err != nil {
		p.Signal(os.Kill)
		p.Wait()
		return -1, err
	}
	return utils.ExitStatus(ps.Sys().(syscall.WaitStatus)), nil
}
