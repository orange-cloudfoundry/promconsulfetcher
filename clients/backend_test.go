package clients_test

import (
	"crypto/x509"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orange-cloudfoundry/promconsulfetcher/clients"
	"github.com/orange-cloudfoundry/promconsulfetcher/config"
	"github.com/orange-cloudfoundry/promconsulfetcher/models"
)

var _ = Describe("Backend", func() {
	Context("NewClient", func() {
		It("will give you a http client with given template", func() {
			factory := clients.NewBackendFactory(config.Config{
				CAPool:            &x509.CertPool{},
				SkipSSLValidation: false,
				Backends: config.BackendConfig{
					ClientAuthCertificate: nil,
				},
				DisableKeepAlives:   false,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
			})

			client := factory.NewClient(&models.Route{})

			Expect(client).ToNot(BeNil())
			Expect(client.Timeout).To(Equal(30 * time.Second))

			httpTrans, ok := client.Transport.(*http.Transport)
			Expect(ok).To(BeTrue())
			Expect(httpTrans.TLSClientConfig.ServerName).To(Equal(""))
			Expect(httpTrans.MaxIdleConns).To(Equal(100))
			Expect(httpTrans.MaxIdleConnsPerHost).To(Equal(100))
		})
	})
})
