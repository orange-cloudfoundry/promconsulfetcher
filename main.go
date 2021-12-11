package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"

	"github.com/orange-cloudfoundry/promconsulfetcher/api"
	"github.com/orange-cloudfoundry/promconsulfetcher/clients"
	"github.com/orange-cloudfoundry/promconsulfetcher/config"
	"github.com/orange-cloudfoundry/promconsulfetcher/fetchers"
	"github.com/orange-cloudfoundry/promconsulfetcher/healthchecks"
	"github.com/orange-cloudfoundry/promconsulfetcher/scrapers"
	"github.com/orange-cloudfoundry/promconsulfetcher/userdocs"
)

var (
	configFile = kingpin.Flag("config", "Configuration File").Default("config.yml").Short('c').File()
)

func main() {
	kingpin.Version(version.Print("promconsulfetcher"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	c, err := config.DefaultConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err.Error())
	}

	if *configFile != nil {
		c, err = config.InitConfigFromFile(*configFile)
		if err != nil {
			log.Fatal("Error loading config: ", err.Error())
		}
	}

	backendFactory := clients.NewBackendFactory(*c)
	scraper := scrapers.NewScraper(backendFactory)

	healthCheck := healthchecks.NewHealthCheck()
	routeFetcher, err := fetchers.NewRoutesFetcher(c.ConsulConfig)
	if err != nil {
		log.Fatal("Error loading route fetcher: ", err.Error())
	}
	metricsFetcher := fetchers.NewMetricsFetcher(scraper, routeFetcher, c.ExternalExporters)

	rtr := mux.NewRouter()
	api.Register(
		rtr, metricsFetcher,
		userdocs.NewUserDoc(c.BaseURL),
	)

	srvSignal := make(chan os.Signal, 1)
	signal.Notify(srvSignal, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1)

	srvCtx, cancel := context.WithCancel(context.Background())

	go func() {
		sig := <-srvSignal
		if sig == syscall.SIGUSR1 {
			healthCheck.SetHealth(healthchecks.Degraded)
		}
		cancel()
	}()

	listener, err := makeListener(c)
	if err != nil {
		log.Fatal(err.Error())
	}
	srv := &http.Server{Handler: rtr}

	go func() {
		if err = srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	go func() {
		if err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", c.HealthCheckPort), healthCheck); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen healthcheck: %s\n", err)
		}
	}()
	defer srv.Close()

	<-srvCtx.Done()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer func() {
		cancel()
	}()

	err = srv.Shutdown(ctxShutDown)
	if err != nil {
		log.Fatalf("server shutdown gracefully Failed: %s\n", err.Error())
	}
	log.Info("server gracefully shutdown")
}

func makeListener(c *config.Config) (net.Listener, error) {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", c.Port)
	if !c.EnableSSL {
		log.Infof("Listen %s without tls ...", listenAddr)
		return net.Listen("tcp", listenAddr)
	}
	log.Infof("Listen %s with tls ...", listenAddr)
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = nil
	}
	if err == nil {
		if c.CACerts != "" {
			if ok := rootCAs.AppendCertsFromPEM([]byte(c.CACerts)); !ok {
				return nil, fmt.Errorf("error adding a CA cert to cert pool")
			}
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{c.SSLCertificate},
		ClientCAs:    rootCAs,
	}

	tlsConfig.BuildNameToCertificate()
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(listener, tlsConfig), nil
}

func checkDbConnection(db *gorm.DB) {
	if db == nil {
		return
	}
	for {
		err := db.DB().Ping()
		if err != nil {
			db.Close()
			log.Fatalf("Error when pinging database: %s", err.Error())
		}
		time.Sleep(5 * time.Minute)
	}
}
