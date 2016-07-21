package metrics

import (
	"net/http"

	"github.com/docker/docker/api/server/router"
	"github.com/docker/docker/metrics"
	"golang.org/x/net/context"
)

type metricsRouter struct {
	routes []router.Route
}

// NewRouter initializes a new metrics router
func NewRouter() router.Router {
	r := &metricsRouter{}
	r.routes = []router.Route{
		router.NewGetRoute("/metrics", r.getMetrics),
	}
	return r
}

// Routes returns the available routers to the metrics controller
func (m *metricsRouter) Routes() []router.Route {
	return m.routes
}

func (m *metricsRouter) getMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	metricsType := vars["type"]
	if metricsType == "" || metricsType == "prometheus" {
		metrics.Handler().ServeHTTP(w, r)
		return nil
	}
	return nil
}
