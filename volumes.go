package docker

import (
	"fmt"
	"github.com/dotcloud/docker/archive"
	"github.com/dotcloud/docker/execdriver"
	"github.com/dotcloud/docker/pkg/mount"
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type BindMap struct {
	SrcPath string
	DstPath string
	Mode    string
}

func prepareVolumesForContainer(container *Container) error {
	if container.Volumes == nil || len(container.Volumes) == 0 {
		container.Volumes = make(map[string]string)
		container.VolumesRW = make(map[string]bool)
	}

	if err := applyVolumesFrom(container); err != nil {
		return err
	}
	if err := createVolumes(container); err != nil {
		return err
	}
	return nil
}

func mountVolumesForContainer(container *Container, envPath string) error {
	// Setup the root fs as a bind mount of the base fs
	var (
		root    = container.RootfsPath()
		runtime = container.runtime
	)
	if err := os.MkdirAll(root, 0755); err != nil && !os.IsExist(err) {
		return nil
	}

	// Create a bind mount of the base fs as a place where we can add mounts
	// without affecting the ability to access the base fs
	if err := mount.Mount(container.basefs, root, "none", "bind,rw"); err != nil {
		return err
	}

	// Make sure the root fs is private so the mounts here don't propagate to basefs
	if err := mount.ForceMount(root, root, "none", "private"); err != nil {
		return err
	}

	mounts := []execdriver.Mount{
		{runtime.sysInitPath, "/.dockerinit", false},
		{envPath, "/.dockerenv", false},
		{container.ResolvConfPath, "/etc/resolv.conf", false},
	}

	if container.HostnamePath != "" && container.HostsPath != "" {
		mounts = append(mounts, execdriver.Mount{container.HostnamePath, "/etc/hostname", false})
		mounts = append(mounts, execdriver.Mount{container.HostsPath, "/etc/hosts", false})
	}

	// Mount user specified volumes
	for r, v := range container.Volumes {
		mounts = append(mounts, execdriver.Mount{v, r, container.VolumesRW[r]})
	}

	container.command.Mounts = mounts

	return nil
}

func unmountVolumesForContainer(container *Container) {
	if err := mount.Unmount(container.RootfsPath()); err != nil {
		log.Printf("Failed to umount container: %v", err)
	}
}

func applyVolumesFrom(container *Container) error {
	if container.Config.VolumesFrom != "" {
		for _, containerSpec := range strings.Split(container.Config.VolumesFrom, ",") {
			var (
				mountRW   = true
				specParts = strings.SplitN(containerSpec, ":", 2)
			)

			switch len(specParts) {
			case 0:
				return fmt.Errorf("Malformed volumes-from specification: %s", container.Config.VolumesFrom)
			case 2:
				switch specParts[1] {
				case "ro":
					mountRW = false
				case "rw": // mountRW is already true
				default:
					return fmt.Errorf("Malformed volumes-from specification: %s", containerSpec)
				}
			}

			c := container.runtime.Get(specParts[0])
			if c == nil {
				return fmt.Errorf("Container %s not found. Impossible to mount its volumes", container.ID)
			}

			for volPath, id := range c.Volumes {
				if _, exists := container.Volumes[volPath]; exists {
					continue
				}
				if err := os.MkdirAll(filepath.Join(container.basefs, volPath), 0755); err != nil {
					return err
				}
				container.Volumes[volPath] = id
				if isRW, exists := c.VolumesRW[volPath]; exists {
					container.VolumesRW[volPath] = isRW && mountRW
				}
			}

		}
	}
	return nil
}

func getBindMap(container *Container) (map[string]BindMap, error) {
	var (
		// Create the requested bind mounts
		binds = make(map[string]BindMap)
		// Define illegal container destinations
		illegalDsts = []string{"/", "."}
	)

	for _, bind := range container.hostConfig.Binds {
		// FIXME: factorize bind parsing in parseBind
		var (
			src, dst, mode string
			arr            = strings.Split(bind, ":")
		)

		if len(arr) == 2 {
			src = arr[0]
			dst = arr[1]
			mode = "rw"
		} else if len(arr) == 3 {
			src = arr[0]
			dst = arr[1]
			mode = arr[2]
		} else {
			return nil, fmt.Errorf("Invalid bind specification: %s", bind)
		}

		// Bail if trying to mount to an illegal destination
		for _, illegal := range illegalDsts {
			if dst == illegal {
				return nil, fmt.Errorf("Illegal bind destination: %s", dst)
			}
		}

		bindMap := BindMap{
			SrcPath: src,
			DstPath: dst,
			Mode:    mode,
		}
		binds[filepath.Clean(dst)] = bindMap
	}
	return binds, nil
}

func createVolumes(container *Container) error {
	binds, err := getBindMap(container)
	if err != nil {
		return err
	}

	volumesDriver := container.runtime.volumes.driver
	// Create the requested volumes if they don't exist
	for volPath := range container.Config.Volumes {
		volPath = filepath.Clean(volPath)
		volIsDir := true
		// Skip existing volumes
		if _, exists := container.Volumes[volPath]; exists {
			continue
		}
		var srcPath string
		var isBindMount bool
		srcRW := false
		// If an external bind is defined for this volume, use that as a source
		if bindMap, exists := binds[volPath]; exists {
			isBindMount = true
			srcPath = bindMap.SrcPath
			if strings.ToLower(bindMap.Mode) == "rw" {
				srcRW = true
			}
			if stat, err := os.Stat(bindMap.SrcPath); err != nil {
				return err
			} else {
				volIsDir = stat.IsDir()
			}
			// Otherwise create an directory in $ROOT/volumes/ and use that
		} else {

			// Do not pass a container as the parameter for the volume creation.
			// The graph driver using the container's information ( Image ) to
			// create the parent.
			c, err := container.runtime.volumes.Create(nil, nil, "", "", nil)
			if err != nil {
				return err
			}
			srcPath, err = volumesDriver.Get(c.ID)
			if err != nil {
				return fmt.Errorf("Driver %s failed to get volume rootfs %s: %s", volumesDriver, c.ID, err)
			}
			srcRW = true // RW by default
		}

		if p, err := filepath.EvalSymlinks(srcPath); err != nil {
			return err
		} else {
			srcPath = p
		}

		container.Volumes[volPath] = srcPath
		container.VolumesRW[volPath] = srcRW

		// Create the mountpoint
		volPath = filepath.Join(container.basefs, volPath)
		rootVolPath, err := utils.FollowSymlinkInScope(volPath, container.basefs)
		if err != nil {
			return err
		}

		if _, err := os.Stat(rootVolPath); err != nil {
			if os.IsNotExist(err) {
				if volIsDir {
					if err := os.MkdirAll(rootVolPath, 0755); err != nil {
						return err
					}
				} else {
					if err := os.MkdirAll(filepath.Dir(rootVolPath), 0755); err != nil {
						return err
					}
					if f, err := os.OpenFile(rootVolPath, os.O_CREATE, 0755); err != nil {
						return err
					} else {
						f.Close()
					}
				}
			}
		}

		// Do not copy or change permissions if we are mounting from the host
		if srcRW && !isBindMount {
			volList, err := ioutil.ReadDir(rootVolPath)
			if err != nil {
				return err
			}
			if len(volList) > 0 {
				srcList, err := ioutil.ReadDir(srcPath)
				if err != nil {
					return err
				}
				if len(srcList) == 0 {
					// If the source volume is empty copy files from the root into the volume
					if err := archive.CopyWithTar(rootVolPath, srcPath); err != nil {
						return err
					}

					var stat syscall.Stat_t
					if err := syscall.Stat(rootVolPath, &stat); err != nil {
						return err
					}
					var srcStat syscall.Stat_t
					if err := syscall.Stat(srcPath, &srcStat); err != nil {
						return err
					}
					// Change the source volume's ownership if it differs from the root
					// files that were just copied
					if stat.Uid != srcStat.Uid || stat.Gid != srcStat.Gid {
						if err := os.Chown(srcPath, int(stat.Uid), int(stat.Gid)); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
