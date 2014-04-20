package fs

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dotcloud/docker/pkg/cgroups"
)

type cpuacctGroup struct {
}

func (s *cpuacctGroup) Set(d *data) error {
	// we just want to join this group even though we don't set anything
	if _, err := d.join("cpuacct"); err != nil && err != cgroups.ErrNotFound {
		return err
	}
	return nil
}

func (s *cpuacctGroup) Remove(d *data) error {
	return removePath(d.path("cpuacct"))
}

func (s *cpuacctGroup) Stats(d *data) (map[string]float64, error) {
	paramData := make(map[string]float64)
	path, err := d.path("cpuacct")
	if err != nil {
		return paramData, fmt.Errorf("Unable to read %s cgroup param: %s", path, err)
	}
	cpuPath := filepath.Join(path, "cpuacct.stat")
	f, err := os.Open(cpuPath)
	if err != nil {
		return paramData, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	cpuTotal := 0.0
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		t := fields[0]
		v, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			fmt.Printf("Error parsing cpu stats: %s", err)
			continue
		}
		// set the raw data in map
		paramData[t] = float64(v)
		cpuTotal += v
	}
	// calculate percentage from jiffies
	// get sys uptime
	uf, err := os.Open("/proc/uptime")
	if err != nil {
		return paramData, fmt.Errorf("Unable to open /proc/uptime")
	}
	defer uf.Close()
	uptimeData, _ := ioutil.ReadAll(uf)
	uptimeFields := strings.Fields(string(uptimeData))
	uptimeFloat, err := strconv.ParseFloat(uptimeFields[0], 64)
	uptime := int(uptimeFloat)
	if err != nil {
		return paramData, fmt.Errorf("Error parsing cpu stats: %s", err)
	}
	// find starttime of process
	pidProcsPath := filepath.Join(path, "cgroup.procs")
	pf, err := os.Open(pidProcsPath)
	if err != nil {
		return paramData, fmt.Errorf("Error parsing cpu stats: %s", err)
	}
	defer pf.Close()
	pr := bufio.NewReader(pf)
	l, _, _ := pr.ReadLine()
	starttime, _ := strconv.Atoi(string(l))
	// get total elapsed seconds since proc start
	seconds := uptime - (starttime / 100)
	// finally calc percentage
	cpuPercentage := 100.0 * ((cpuTotal / 100.0) / float64(seconds))
	paramData["percentage"] = cpuPercentage
	return paramData, nil
}
