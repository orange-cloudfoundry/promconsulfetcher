package config

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	txttpl "text/template"

	"github.com/orange-cloudfoundry/promconsulfetcher/models"
)

type ExternalExporters []*ExternalExporter

type ExternalExporter struct {
	Name        string                     `yaml:"name"`
	Host        string                     `yaml:"host"`
	MetricsPath string                     `yaml:"metrics_path"`
	Scheme      string                     `yaml:"scheme"`
	Params      map[string][]ValueTemplate `yaml:"params"`
	IsTls       bool                       `yaml:"-"`
}

func (ee *ExternalExporter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ExternalExporter
	err := unmarshal((*plain)(ee))
	if err != nil {
		return err
	}
	if ee.Host == "" {
		return fmt.Errorf("Host must be provided on external exporter")
	}
	if ee.MetricsPath == "" {
		ee.MetricsPath = "/metrics"
	}
	if ee.Name == "" {
		ee.Name = ee.Host + ee.MetricsPath
	}
	if ee.Scheme == "" {
		ee.Scheme = "http"
	}
	if ee.Scheme == "https" {
		ee.IsTls = true
	}

	return nil
}

func (ee *ExternalExporter) ToRoute(route *models.Route) (*models.Route, error) {
	urlValues, err := ee.ParamsToURLValues(route)
	if err != nil {
		return nil, fmt.Errorf("error on external exporter `%s`: %s", ee.Name, err.Error())
	}

	hostSplit := strings.SplitN(ee.Host, ":", 2)
	host := hostSplit[0]
	port := -1
	if len(hostSplit) > 1 {
		port, _ = strconv.Atoi(hostSplit[1]) //nolint
	}
	return &models.Route{
		ID:              ee.Name,
		Node:            "external_exporter",
		Address:         host,
		Datacenter:      route.Datacenter,
		TaggedAddresses: map[string]string{},
		NodeMeta:        map[string]string{},
		ServiceID:       ee.Name,
		ServiceName:     ee.Name,
		ServiceAddress:  host,
		ServiceTags: models.ServiceTags{
			fmt.Sprintf("%s=%s", models.SchemeTagsKey, ee.Scheme),
			fmt.Sprintf("%s=%s?%s", models.MetricPathTagsKey, ee.MetricsPath, urlValues.Encode()),
		},
		ServiceMeta: map[string]string{},
		ServicePort: port,
	}, nil
}

func (ee *ExternalExporter) ParamsToURLValues(route *models.Route) (url.Values, error) {
	urlValue := make(url.Values)
	var err error
	for key, values := range ee.Params {
		finalValues := make([]string, len(values))
		for i, valueTpl := range values {
			finalValues[i], err = valueTpl.ResolveTags(route)
			if err != nil {
				return nil, err
			}
		}
		urlValue[key] = finalValues
	}
	return urlValue, nil
}

type ValueTemplate struct {
	Raw string
	tpl *txttpl.Template
}

func (vt *ValueTemplate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawString := ""
	err := unmarshal(&rawString)
	if err != nil {
		return err
	}
	vt.Raw = rawString
	if !strings.Contains(vt.Raw, "{{") {
		return nil
	}

	vt.tpl, err = txttpl.New("").Parse(vt.Raw)
	if err != nil {
		return err
	}
	return nil
}

func (vt *ValueTemplate) ResolveTags(route *models.Route) (string, error) {
	if vt.tpl == nil {
		return vt.Raw, nil
	}
	buf := &bytes.Buffer{}
	err := vt.tpl.Execute(buf, route)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
