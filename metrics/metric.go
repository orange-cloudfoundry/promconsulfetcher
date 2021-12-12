package metrics

import (
	"github.com/orange-cloudfoundry/promconsulfetcher/models"

	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricFetchFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "promconsulfetcher_metric_fetch_failed_total",
			Help: "Number of non fetched metrics without be an normal error.",
		},
		[]string{"node_name", "node_id", "node_address", "datacenter", "service_name", "service_id", "service_address", "service_port"},
	)
	MetricFetchSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "promconsulfetchermetric_fetch_success_total",
			Help: "Number of fetched metrics succeeded for an app (app instance call are added).",
		},
		[]string{"node_name", "node_id", "node_address", "datacenter", "service_name", "service_id", "service_address", "service_port"},
	)
	LatestScrapeRoute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "promconsulfetcher_latest_time_scrape_route",
			Help: "Last time that route has been scraped in seconds.",
		},
		[]string{},
	)
	ScrapeRouteFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "promconsulfetcher_scrape_route_failed_total",
			Help: "Number of non fetched metrics without be an normal error.",
		},
		[]string{},
	)
)

func RouteToLabel(route *models.Route) prometheus.Labels {
	return map[string]string{
		"node_name":       route.Node,
		"node_id":         route.ID,
		"node_address":    route.Address,
		"datacenter":      route.Datacenter,
		"service_name":    route.ServiceName,
		"service_id":      route.ServiceID,
		"service_address": route.ServiceAddress,
		"service_port":    strconv.Itoa(route.ServicePort),
	}
}

func RouteToLabelNoInstance(route *models.Route) prometheus.Labels {
	return map[string]string{
		"node_name":       route.Node,
		"node_id":         route.ID,
		"node_address":    route.Address,
		"datacenter":      route.Datacenter,
		"service_name":    route.ServiceName,
		"service_id":      route.ServiceID,
		"service_address": route.ServiceAddress,
		"service_port":    strconv.Itoa(route.ServicePort),
	}
}

func init() {
	prometheus.MustRegister(MetricFetchFailedTotal)
	prometheus.MustRegister(LatestScrapeRoute)
	prometheus.MustRegister(ScrapeRouteFailedTotal)
	prometheus.MustRegister(MetricFetchSuccessTotal)
}
