package fetchers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"

	"github.com/orange-cloudfoundry/promconsulfetcher/config"
	"github.com/orange-cloudfoundry/promconsulfetcher/errors"
	"github.com/orange-cloudfoundry/promconsulfetcher/metrics"
	"github.com/orange-cloudfoundry/promconsulfetcher/models"
	"github.com/orange-cloudfoundry/promconsulfetcher/scrapers"
)

func ptrString(v string) *string {
	return &v
}

type MetricsFetcher struct {
	scraper           *scrapers.Scraper
	routesFetcher     RoutesFetch
	externalExporters config.ExternalExporters
}

func NewMetricsFetcher(scraper *scrapers.Scraper, routesFetcher RoutesFetch, externalExporters config.ExternalExporters) *MetricsFetcher {
	return &MetricsFetcher{
		scraper:           scraper,
		routesFetcher:     routesFetcher,
		externalExporters: externalExporters,
	}
}

func (f MetricsFetcher) Metrics(consulQuery string, metricPathDefault, schemeDefault string, onlyAppMetrics bool, headers http.Header) (map[string]*dto.MetricFamily, error) {
	serviceSearch, err := models.SearchToServiceSearch(consulQuery)
	if err != nil {
		return nil, err
	}
	routes, err := f.routesFetcher.Routes(serviceSearch)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return make(map[string]*dto.MetricFamily), errors.ErrNoAppFound(consulQuery)
	}

	jobs := make(chan *models.Route, len(routes))
	errFetch := &errors.ErrFetch{}
	wg := &sync.WaitGroup{}

	muWrite := sync.Mutex{}
	metricsUnmerged := make([]map[string]*dto.MetricFamily, 0)

	if !onlyAppMetrics && f.externalExporters != nil && len(f.externalExporters) > 0 {
		for _, rte := range routes {
			for _, ee := range f.externalExporters {
				routeExternalExporter, err := ee.ToRoute(rte)
				if err != nil {
					err = fmt.Errorf("error when setting external exporters routes: %s", err.Error())
					newMetrics := f.scrapeExternalExporterError(routeExternalExporter, ee, err)
					metricsUnmerged = append(metricsUnmerged, newMetrics)
					log.WithField("external_exporter", ee.Name).
						WithField("action", "route convert").
						WithField("service", ee.Name).
						Warningf(err.Error())
					continue
				}
				routes = append(routes, routeExternalExporter)
			}
		}
	}

	wg.Add(len(routes))
	for w := 1; w <= 5; w++ {
		go func(jobs <-chan *models.Route, errFetch *errors.ErrFetch, headers http.Header) {
			for j := range jobs {
				if j.Node == "external_exporter" {
					headers = nil
				}
				newMetrics, err := f.Metric(j, metricPathDefault, schemeDefault, headers)
				if err != nil {
					if errF, ok := err.(*errors.ErrFetch); ok && (f.externalExporters == nil || len(f.externalExporters) == 0) {
						muWrite.Lock()
						*errFetch = *errF
						muWrite.Unlock()
						wg.Done()
						continue
					}
					log.Warnf("Cannot get metric for instance %s for service name %s", j.ServiceAddress, j.ServiceName)
					newMetrics = f.scrapeError(j, err)
					metrics.MetricFetchFailedTotal.With(metrics.RouteToLabel(j)).Inc()
				} else {
					metrics.MetricFetchSuccessTotal.With(metrics.RouteToLabelNoInstance(j)).Inc()
				}
				muWrite.Lock()
				metricsUnmerged = append(metricsUnmerged, newMetrics)
				muWrite.Unlock()
				wg.Done()
			}
		}(jobs, errFetch, headers)
	}
	for _, route := range routes {
		jobs <- route
	}
	wg.Wait()
	close(jobs)
	if errFetch.Code != 0 {
		return make(map[string]*dto.MetricFamily), errFetch
	}

	if len(metricsUnmerged) == 0 {
		return make(map[string]*dto.MetricFamily), nil
	}

	base := metricsUnmerged[0]
	for _, metricKV := range metricsUnmerged[1:] {
		for k, metricFamily := range metricKV {
			baseMetricFamily, ok := base[k]
			if !ok {
				base[k] = metricFamily
				continue
			}
			baseMetricFamily.Metric = append(baseMetricFamily.Metric, metricFamily.Metric...)
		}
	}
	return base, nil
}

func (f MetricsFetcher) Metric(route *models.Route, metricPathDefault, schemeDefault string, headers http.Header) (map[string]*dto.MetricFamily, error) {
	reader, err := f.scraper.Scrape(route, metricPathDefault, schemeDefault, headers)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	parser := &expfmt.TextParser{}
	metricsGroup, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}

	for _, metricGroup := range metricsGroup {
		for _, metric := range metricGroup.Metric {
			metric.Label = f.cleanMetricLabels(
				metric.Label,
				"node_name", "node_id", "node_address",
				"datacenter", "service_name", "service_id",
				"service_address", "service_port",
			)
			metric.Label = append(metric.Label,
				&dto.LabelPair{
					Name:  ptrString("node_name"),
					Value: ptrString(route.Node),
				},
				&dto.LabelPair{
					Name:  ptrString("node_id"),
					Value: ptrString(route.ID),
				},
				&dto.LabelPair{
					Name:  ptrString("node_address"),
					Value: ptrString(route.Address),
				},
				&dto.LabelPair{
					Name:  ptrString("datacenter"),
					Value: ptrString(route.Datacenter),
				},
				&dto.LabelPair{
					Name:  ptrString("service_name"),
					Value: ptrString(route.ServiceName),
				},
				&dto.LabelPair{
					Name:  ptrString("service_id"),
					Value: ptrString(route.ServiceID),
				},
				&dto.LabelPair{
					Name:  ptrString("service_address"),
					Value: ptrString(route.ServiceAddress),
				},
				&dto.LabelPair{
					Name:  ptrString("service_port"),
					Value: ptrString(strconv.Itoa(route.ServicePort)),
				},
			)

		}
	}
	return metricsGroup, nil
}

func (f MetricsFetcher) cleanMetricLabels(labels []*dto.LabelPair, names ...string) []*dto.LabelPair {
	finalLabels := make([]*dto.LabelPair, 0)
	for _, label := range labels {
		toAdd := true
		for _, name := range names {
			if label.Name != nil && *label.Name == name {
				toAdd = false
				break
			}
		}
		if toAdd {
			finalLabels = append(finalLabels, label)
		}
	}
	return finalLabels
}

func (f MetricsFetcher) scrapeError(route *models.Route, err error) map[string]*dto.MetricFamily {
	name := "promconsulfetcher_scrape_error"
	help := "Promconsulfetcher scrap error on your instance"
	metric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
		ConstLabels: prometheus.Labels{
			"node_name":       route.Node,
			"node_id":         route.ID,
			"node_address":    route.Address,
			"datacenter":      route.Datacenter,
			"service_name":    route.ServiceName,
			"service_id":      route.ServiceID,
			"service_address": route.ServiceAddress,
			"service_port":    strconv.Itoa(route.ServicePort),
			"error":           err.Error(),
		},
	})
	metric.Inc()
	var dtoMetric dto.Metric
	metric.Write(&dtoMetric)
	metricType := dto.MetricType_COUNTER
	return map[string]*dto.MetricFamily{
		"promconsulfetcher_scrape_error": {
			Name:   ptrString(name),
			Help:   ptrString(help),
			Type:   &metricType,
			Metric: []*dto.Metric{&dtoMetric},
		},
	}
}

func (f MetricsFetcher) scrapeExternalExporterError(route *models.Route, externalExporter *config.ExternalExporter, err error) map[string]*dto.MetricFamily {
	name := "promconsulfetcher_scrape_external_exporter_error"
	help := "Promconsulfetcher scrap external exporter error on your instance"
	metric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
		ConstLabels: prometheus.Labels{
			"node_name":       route.Node,
			"node_id":         route.ID,
			"node_address":    route.Address,
			"datacenter":      route.Datacenter,
			"service_name":    route.ServiceName,
			"service_id":      route.ServiceID,
			"service_address": route.ServiceAddress,
			"service_port":    strconv.Itoa(route.ServicePort),
			"error":           err.Error(),
		},
	})
	metric.Inc()
	var dtoMetric dto.Metric
	metric.Write(&dtoMetric)
	metricType := dto.MetricType_COUNTER
	return map[string]*dto.MetricFamily{
		"promconsulfetcher_scrape_external_exporter_error": {
			Name:   ptrString(name),
			Help:   ptrString(help),
			Type:   &metricType,
			Metric: []*dto.Metric{&dtoMetric},
		},
	}
}
