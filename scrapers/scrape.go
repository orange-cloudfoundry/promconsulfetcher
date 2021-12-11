package scrapers

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/orange-cloudfoundry/promconsulfetcher/clients"
	"github.com/orange-cloudfoundry/promconsulfetcher/errors"
	"github.com/orange-cloudfoundry/promconsulfetcher/models"
)

const acceptHeader = `application/openmetrics-text; version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1`

type Scraper struct {
	backendFactory *clients.BackendFactory
	outboundIp     string
}

func NewScraper(backendFactory *clients.BackendFactory) *Scraper {
	return &Scraper{backendFactory: backendFactory}

}

func (s *Scraper) GetOutboundIP() string {
	if s.outboundIp != "" {
		return s.outboundIp
	}

	// address doesn't need to exists
	conn, err := net.Dial("udp", "10.0.0.1:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	s.outboundIp = localAddr.IP.String()
	return s.outboundIp
}

func (s Scraper) Scrape(route *models.Route, metricPathDefault, metricSchemeDefault string, headers http.Header) (io.ReadCloser, error) {
	scheme := metricSchemeDefault
	routeScheme := route.FindScheme()
	if routeScheme != "" {
		scheme = routeScheme
	}
	endpoint := metricPathDefault
	routeMetricPath := route.FindMetricsPath()
	if routeMetricPath != "" {
		endpoint = routeMetricPath
	}
	portStr := ""
	if route.ServicePort > 0 {
		portStr = fmt.Sprintf(":%d", route.ServicePort)
	}
	address := route.ServiceAddress + portStr
	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s%s", scheme, address, endpoint), nil)
	if err != nil {
		return nil, err
	}
	if len(headers) > 0 {
		for k, v := range headers {
			req.Header[k] = v
		}
	}
	req.Header.Add("Accept", acceptHeader)
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", fmt.Sprintf("%f", (30*time.Second).Seconds()))
	req.Header.Set("X-Forwarded-Proto", scheme)
	req.Header.Set("X-Promconsulfetcher-Scrapping", "true")
	req.Header.Set("X-Forwarded-For", s.GetOutboundIP())
	client := s.backendFactory.NewClient(route)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
			return nil, errors.ErrNoEndpointFound(
				fmt.Sprintf(
					"%s/%s (status code %d)",
					route.ServiceName,
					route.ServiceID,
					resp.StatusCode,
				), endpoint,
			)
		}
		return nil, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	if resp.Header.Get("Content-Encoding") != "gzip" {
		return resp.Body, nil
	}
	gzReader, err := NewReaderGzip(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	return gzReader, nil
}

type ReaderGzip struct {
	main io.ReadCloser
	gzip *gzip.Reader
}

func NewReaderGzip(main io.ReadCloser) (*ReaderGzip, error) {
	gzReader, err := gzip.NewReader(main)
	if err != nil {
		return nil, err
	}
	return &ReaderGzip{
		main: main,
		gzip: gzReader,
	}, nil
}

func (r ReaderGzip) Read(p []byte) (n int, err error) {
	return r.gzip.Read(p)
}

func (r ReaderGzip) Close() error {
	r.gzip.Close()
	r.main.Close()
	return nil
}
