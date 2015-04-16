package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/chrootarchive"
	"github.com/docker/docker/pkg/system"
)

func parseBindMount(spec string) (*BindMount, error) {
	var (
		bind = &BindMount{
			RW: true,
		}
		arr = strings.Split(spec, ":")
	)
	switch len(arr) {
	case 2:
		bind.Source = arr[0]
		bind.Destination = arr[1]
	case 3:
		bind.Source = arr[0]
		bind.Destination = arr[1]
		bind.RW = validMountMode(arr[2]) && arr[2] == "rw"
	default:
		return nil, fmt.Errorf("Invalid volume specification: %s", spec)
	}
	if !filepath.IsAbs(bind.Source) {
		return nil, fmt.Errorf("cannot bind mount volume: %s volume paths must be absolute.", bind.Source)
	}
	bind.Source = filepath.Clean(bind.Source)
	bind.Destination = filepath.Clean(bind.Destination)
	return bind, nil
}

func parseVolumesFrom(spec string) (string, string, error) {
	specParts := strings.SplitN(spec, ":", 2)
	if len(specParts) == 0 {
		return "", "", fmt.Errorf("malformed volumes-from specification: %s", spec)
	}
	var (
		id   = specParts[0]
		mode = "rw"
	)
	if len(specParts) == 2 {
		mode = specParts[1]
		if !validMountMode(mode) {
			return "", "", fmt.Errorf("invalid mode for volumes-from: %s", mode)
		}
	}
	return id, mode, nil
}

func validMountMode(mode string) bool {
	validModes := map[string]bool{
		"rw": true,
		"ro": true,
	}
	return validModes[mode]
}

func copyExistingContents(source, destination string) error {
	volList, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}
	if len(volList) > 0 {
		srcList, err := ioutil.ReadDir(destination)
		if err != nil {
			return err
		}
		if len(srcList) == 0 {
			// If the source volume is empty copy files from the root into the volume
			if err := chrootarchive.CopyWithTar(source, destination); err != nil {
				return err
			}
		}
	}
	return copyOwnership(source, destination)
}

// copyOwnership copies the permissions and uid:gid of the source file
// into the destination file
func copyOwnership(source, destination string) error {
	stat, err := system.Stat(source)
	if err != nil {
		return err
	}
	if err := os.Chown(destination, int(stat.Uid()), int(stat.Gid())); err != nil {
		return err
	}
	return os.Chmod(destination, os.FileMode(stat.Mode()))
}
