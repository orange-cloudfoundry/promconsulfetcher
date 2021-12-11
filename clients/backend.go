package clients

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/orange-cloudfoundry/promconsulfetcher/config"
	"github.com/orange-cloudfoundry/promconsulfetcher/models"
)

type BackendFactory struct {
	factory FactoryRoundTripper
}

func NewBackendFactory(c config.Config) *BackendFactory {
	backendTLSConfig := &tls.Config{
		InsecureSkipVerify: c.SkipSSLValidation,
		RootCAs:            c.CAPool,
	}
	if c.Backends.ClientAuthCertificate != nil {
		backendTLSConfig.Certificates = []tls.Certificate{*c.Backends.ClientAuthCertificate}
	}
	return &BackendFactory{
		factory: FactoryRoundTripper{
			Template: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				DisableKeepAlives:   c.DisableKeepAlives,
				MaxIdleConns:        c.MaxIdleConns,
				IdleConnTimeout:     90 * time.Second, // setting the value to golang default transport
				MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
				DisableCompression:  false,
				TLSClientConfig:     backendTLSConfig,
			},
		},
	}
}

func (f BackendFactory) NewClient(route *models.Route) *http.Client {
	return &http.Client{
		Transport: f.factory.New(""),
		Timeout:   30 * time.Second,
	}
}
