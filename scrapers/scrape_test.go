package scrapers_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/orange-cloudfoundry/promconsulfetcher/clients"
	"github.com/orange-cloudfoundry/promconsulfetcher/config"
	"github.com/orange-cloudfoundry/promconsulfetcher/models"
	"github.com/orange-cloudfoundry/promconsulfetcher/scrapers"
)

var _ = Describe("Scraper", func() {
	var err error
	var scraper *scrapers.Scraper
	var server *ghttp.Server

	BeforeEach(func() {

		c, err := config.DefaultConfig()
		Expect(err).ShouldNot(HaveOccurred())

		backendFactory := clients.NewBackendFactory(*c)
		scraper = scrapers.NewScraper(backendFactory)

		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Context("Scrape", func() {
		var serverURL *url.URL
		var content = "test_scrape_error 0"
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/metrics")),
					ghttp.RespondWith(http.StatusOK, content),
				),
			)
			serverURL, err = url.Parse(server.URL())
			Expect(err).ToNot(HaveOccurred())
		})

		It("scrapes metrics from an app", func() {
			host, portStr, err := net.SplitHostPort(serverURL.Host)
			Expect(err).ShouldNot(HaveOccurred())
			port, err := strconv.Atoi(portStr)
			Expect(err).ShouldNot(HaveOccurred())
			route := &models.Route{
				ID:             "a758f25d-2d01-419e-b63b-de3aabcd9e15",
				Address:        host,
				ServiceAddress: host,
				ServicePort:    port,
			}

			resp, err := scraper.Scrape(route, "/metrics", "http", http.Header{})
			Expect(err).ShouldNot(HaveOccurred())
			defer resp.Close()

			body, err := ioutil.ReadAll(resp)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(body)).To(Equal(content))
		})
	})

	Context("GetOutboundIP", func() {
		It("gets local ip", func() {
			ip := scraper.GetOutboundIP()
			Expect(ip).Should(MatchRegexp(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`))

		})
	})
})
