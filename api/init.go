package api

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/orange-cloudfoundry/promconsulfetcher/fetchers"
	"github.com/orange-cloudfoundry/promconsulfetcher/userdocs"
)

type Api struct {
	metFetcher *fetchers.MetricsFetcher
}

func Register(rtr *mux.Router, metFetcher *fetchers.MetricsFetcher, us *userdocs.UserDoc) {
	api := &Api{
		metFetcher: metFetcher,
	}

	handlerMetrics := handlers.CompressHandler(http.HandlerFunc(api.metrics))
	rtr.Handle("/v1/services/{consul_query:.*}/metrics", handlerMetrics).
		Methods(http.MethodGet)

	rtr.Handle("/v1/services/metrics", handlerMetrics).
		Methods(http.MethodGet)

	handlerOnlyAppMetrics := handlers.CompressHandler(forceOnlyForApp(http.HandlerFunc(api.metrics)))

	rtr.Handle("/v1/services/{consul_query:.*}/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	rtr.Handle("/v1/services/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	rtr.PathPrefix("/assets/").Handler(http.FileServer(http.FS(userdocs.Assets)))
	rtr.Handle("/doc", us)
	rtr.Handle("/", http.RedirectHandler("/doc", http.StatusPermanentRedirect))
	rtr.Handle("/metrics", promhttp.Handler())
}
