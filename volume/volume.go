package volume

type Driver interface {
	Name() string
	Create(string) (Volume, error)
	Remove(Volume) error
}

type Volume interface {
	// Name returns the name of the volume
	Name() string
	// Mount mounts the volume and returns the absolute path to
	// where it can be found.
	Mount() (string, error)
	// Unmount unmounts the volume.
	Unmount() error
}
