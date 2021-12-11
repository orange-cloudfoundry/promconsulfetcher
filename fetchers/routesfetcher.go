package fetchers

import (
	"sort"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/orange-cloudfoundry/promconsulfetcher/config"
	"github.com/orange-cloudfoundry/promconsulfetcher/models"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . RoutesFetch

type RoutesFetch interface {
	Routes(search models.ServiceSearch) (models.Routes, error)
}

type RoutesFetcher struct {
	consulClient *api.Client
}

func NewRoutesFetcher(consulConfig config.ConsulConfig) (*RoutesFetcher, error) {
	client, err := createClient(consulConfig)
	if err != nil {
		return nil, err
	}
	return &RoutesFetcher{
		consulClient: client,
	}, nil
}

func (f *RoutesFetcher) Routes(search models.ServiceSearch) (models.Routes, error) {

	entries, _, err := f.consulClient.Catalog().Service(search.Name, search.Tag, &api.QueryOptions{
		Datacenter: search.Datacenter,
		Near:       search.Near,
	})
	if err != nil {
		return nil, errors.Wrap(err, search.String())
	}

	var list models.Routes
	for _, s := range entries {
		list = append(list, &models.Route{
			ID:              s.ID,
			Node:            s.Node,
			Address:         s.Address,
			Datacenter:      s.Datacenter,
			TaggedAddresses: s.TaggedAddresses,
			NodeMeta:        s.NodeMeta,
			ServiceID:       s.ServiceID,
			ServiceName:     s.ServiceName,
			ServiceAddress:  s.ServiceAddress,
			ServiceTags:     deepCopyAndSortTags(s.ServiceTags),
			ServiceMeta:     s.ServiceMeta,
			ServicePort:     s.ServicePort,
		})
	}
	return list, nil
}

func createClient(cfg config.ConsulConfig) (*api.Client, error) {
	config := api.Config{
		Address:    cfg.Address,
		Scheme:     cfg.Scheme,
		Datacenter: cfg.DataCenter,
		WaitTime:   time.Duration(cfg.EndpointWaitTime),
		Token:      cfg.Token,
	}

	if cfg.HTTPAuth != nil {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: cfg.HTTPAuth.Username,
			Password: cfg.HTTPAuth.Password,
		}
	}

	if cfg.TLS != nil {
		config.TLSConfig = api.TLSConfig{
			Address:            cfg.Address,
			CAFile:             cfg.TLS.CA,
			CertFile:           cfg.TLS.Cert,
			KeyFile:            cfg.TLS.Key,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}
	}

	return api.NewClient(&config)
}

// deepCopyAndSortTags deep copies the tags in the given string slice and then
// sorts and returns the copied result.
func deepCopyAndSortTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		newTags = append(newTags, tag)
	}
	sort.Strings(newTags)
	return newTags
}
