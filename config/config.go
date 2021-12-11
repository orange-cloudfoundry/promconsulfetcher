package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type ConsulConfig struct {
	Address          string                  `yaml:"address"`
	Scheme           string                  `yaml:"scheme"`
	DataCenter       string                  `yaml:"datacenter"`
	Token            string                  `yaml:"token"`
	TLS              *ClientTLS              `yaml:"tls"`
	HTTPAuth         *EndpointHTTPAuthConfig `yaml:"http_auth"`
	EndpointWaitTime yamlTimeDur             `yaml:"endpoint_wait_time"`
}

type EndpointHTTPAuthConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type ClientTLS struct {
	CA                 string `yaml:"ca,omitempty"`
	Cert               string `yaml:"cert,omitempty"`
	Key                string `yaml:"key,omitempty"`
	InsecureSkipVerify bool   `yaml:"skip_ssl_validation"`
}

type BackendConfig struct {
	ClientAuthCertificate *tls.Certificate `yaml:"-"`
	MaxConns              int64            `yaml:"max_conns"`

	TLSPem `yaml:",inline"` // embed to get cert_chain and private_key for client authentication
}

type yamlTimeDur time.Duration

func (t *yamlTimeDur) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tm string
	if err := unmarshal(&tm); err != nil {
		return err
	}

	td, err := time.ParseDuration(tm)
	if err != nil {
		return fmt.Errorf("failed to parse '%s' to time.Duration: %v", tm, err)
	}

	*t = yamlTimeDur(td)
	return nil
}

func (t *yamlTimeDur) Duration() time.Duration {
	return time.Duration(*t)
}

type Log struct {
	Level   string `yaml:"level"`
	NoColor bool   `yaml:"no_color"`
	InJson  bool   `yaml:"in_json"`
}

func (c *Log) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Log
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: c.NoColor,
	})
	if c.Level != "" {
		lvl, err := log.ParseLevel(c.Level)
		if err != nil {
			return err
		}
		log.SetLevel(lvl)
	}
	if c.InJson {
		log.SetFormatter(&log.JSONFormatter{})
	}

	return nil
}

type TLSPem struct {
	CertChain  string `yaml:"cert_chain"`
	PrivateKey string `yaml:"private_key"`
}

type Config struct {
	ConsulConfig      ConsulConfig    `yaml:"consul,omitempty"`
	Logging           Log             `yaml:"logging,omitempty"`
	Port              uint16          `yaml:"port,omitempty"`
	HealthCheckPort   uint16          `yaml:"health_check_port,omitempty"`
	EnableSSL         bool            `yaml:"enable_ssl,omitempty"`
	SSLCertificate    tls.Certificate `yaml:"-"`
	TLSPEM            TLSPem          `yaml:"tls_pem,omitempty"`
	CACerts           string          `yaml:"ca_certs,omitempty"`
	CAPool            *x509.CertPool  `yaml:"-"`
	SkipSSLValidation bool            `yaml:"skip_ssl_validation,omitempty"`

	Backends BackendConfig `yaml:"backends,omitempty"`

	DisableKeepAlives   bool `yaml:"disable_keep_alives"`
	MaxIdleConns        int  `yaml:"max_idle_conns,omitempty"`
	MaxIdleConnsPerHost int  `yaml:"max_idle_conns_per_host,omitempty"`

	BaseURL string `yaml:"base_url"`

	ExternalExporters ExternalExporters `yaml:"external_exporters"`
}

var defaultConfig = Config{
	ConsulConfig: ConsulConfig{
		Address:          "127.0.0.1:8500",
		Scheme:           "http",
		DataCenter:       "",
		Token:            "",
		TLS:              nil,
		HTTPAuth:         nil,
		EndpointWaitTime: 0,
	},
	Logging:             Log{},
	Port:                8085,
	HealthCheckPort:     8080,
	DisableKeepAlives:   true,
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 2,
	BaseURL:             "http://localhost:8085",
}

func DefaultConfig() (*Config, error) {
	c := defaultConfig
	return &c, nil
}

func (c *Config) Process() error {
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	if c.Backends.CertChain != "" && c.Backends.PrivateKey != "" {
		certificate, err := tls.X509KeyPair([]byte(c.Backends.CertChain), []byte(c.Backends.PrivateKey))
		if err != nil {
			errMsg := fmt.Sprintf("Error loading key pair: %s", err.Error())
			return fmt.Errorf(errMsg)
		}
		c.Backends.ClientAuthCertificate = &certificate
	}

	if c.EnableSSL {
		if c.TLSPEM.PrivateKey == "" || c.TLSPEM.CertChain == "" {
			return fmt.Errorf("Error parsing PEM blocks of router.tls_pem, missing cert or key.")
		}

		certificate, err := tls.X509KeyPair([]byte(c.TLSPEM.CertChain), []byte(c.TLSPEM.PrivateKey))
		if err != nil {
			errMsg := fmt.Sprintf("Error loading key pair: %s", err.Error())
			return fmt.Errorf(errMsg)
		}
		c.SSLCertificate = certificate
	}

	if err := c.buildCertPool(); err != nil {
		return err
	}
	return nil
}

func (c *Config) buildCertPool() error {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return err
	}

	if c.CACerts != "" {
		if ok := certPool.AppendCertsFromPEM([]byte(c.CACerts)); !ok {
			return fmt.Errorf("Error while adding CACerts to gorouter's cert pool: \n%s\n", c.CACerts)
		}
	}
	c.CAPool = certPool
	return nil
}

func (c *Config) Initialize(configYAML []byte) error {
	return yaml.Unmarshal(configYAML, &c)
}

func InitConfigFromFile(file *os.File) (*Config, error) {
	c, err := DefaultConfig()
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = c.Initialize(b)
	if err != nil {
		return nil, err
	}

	err = c.Process()
	if err != nil {
		return nil, err
	}

	return c, nil
}
