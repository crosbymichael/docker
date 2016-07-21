package daemon

import (
	"github.com/docker/docker/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// have only been adding timers for successful function calls, if a function errors then it could
// add bad data for us.
var (
	ContainersRunningGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "containers_running_count",
		Help:      "The number of running containers.",
	})
	DeleteContainerTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_delete_ns",
		Help:      "The number of nanoseconds it takes to delete a container.",
	})
	ListContainerTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "containers_list_ns",
		Help:      "The number of nanoseconds it takes to list all containers.",
	})
	ContainerChangesTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_changes_ns",
		Help:      "The number of nanoseconds it takes to list fs changes in a container.",
	})
	ContainerStartTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_start_ns",
		Help:      "The number of nanoseconds it takes to start a container.",
	})
	ContainerStatsTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_stats_ns",
		Help:      "The number of nanoseconds it takes to collect stats for a container.",
	})
	ContainerCommitTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_commit_ns",
		Help:      "The number of nanoseconds it takes to commit a container.",
	})
	CreateContainerTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_create_ns",
		Help:      "The number of nanoseconds it takes to create a container.",
	})
	MountContainerTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "container_mount_ns",
		Help:      "The number of nanoseconds it takes to mount a container's rootfs.",
	})
	ImageDeleteTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "image_delete_ns",
		Help:      "The number of nanoseconds it takes to delete an image.",
	})
	ImageHistoryTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "image_history_ns",
		Help:      "The number of nanoseconds it takes to get the history of an image.",
	})
	ImageInspectTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "image_inspect_ns",
		Help:      "The number of nanoseconds it takes to inspect an image.",
	})
	ImagesListTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "images_list_ns",
		Help:      "The number of nanoseconds it takes to list all images.",
	})
	NetworkCreateTimer = metrics.NewTimer(metrics.TimerOpts{
		Namespace: "docker",
		Subsystem: "daemon",
		Name:      "network_create_ns",
		Help:      "The number of nanoseconds it takes to create a network.",
	})
)

func init() {
	prometheus.MustRegister(ContainersRunningGauge)
	prometheus.MustRegister(DeleteContainerTimer)
	prometheus.MustRegister(ListContainerTimer)
	prometheus.MustRegister(ContainerChangesTimer)
	prometheus.MustRegister(ContainerCommitTimer)
	prometheus.MustRegister(CreateContainerTimer)
	prometheus.MustRegister(MountContainerTimer)
	prometheus.MustRegister(ImageDeleteTimer)
	prometheus.MustRegister(ImageHistoryTimer)
	prometheus.MustRegister(ImageInspectTimer)
	prometheus.MustRegister(ImagesListTimer)
	prometheus.MustRegister(NetworkCreateTimer)
	prometheus.MustRegister(ContainerStartTimer)
	prometheus.MustRegister(ContainerStatsTimer)
}
