package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Handler returns an http.Handler for serving the api endpoints
//
// this uses the default registry which could be an issue if the some of the
// vendored packages that we use register their metrics here as well.  It could
// polute what we are trying to collect
func Handler() http.Handler {
	return prometheus.Handler()
}

type TimerOpts prometheus.Opts

func NewTimer(o TimerOpts) Timer {
	return &timer{
		m: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: o.Namespace,
			Subsystem: o.Subsystem,
			Name:      o.Name,
			Help:      o.Help,
		}),
	}
}

// Timer is a Metric that represents a the time of an operation in nanoseconds
type Timer interface {
	prometheus.Collector
	prometheus.Metric

	// UpdateSince will set the the Timer to the number of
	// nanoseconds since the time provided
	UpdateSince(time.Time)
}

type timer struct {
	m prometheus.Summary
}

func (t *timer) UpdateSince(since time.Time) {
	ns := time.Now().Sub(since).Nanoseconds()
	t.m.Observe(float64(ns))
}

func (t *timer) Write(d *dto.Metric) error {
	return t.m.Write(d)
}

func (t *timer) Desc() *prometheus.Desc {
	return t.m.Desc()
}

func (t *timer) Describe(c chan<- *prometheus.Desc) {
	t.m.Describe(c)
}

func (t *timer) Collect(c chan<- prometheus.Metric) {
	t.m.Collect(c)
}
