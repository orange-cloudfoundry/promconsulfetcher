package models

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	SchemeTagsKey     = "promconsulfetcher.scheme"
	MetricPathTagsKey = "promconsulfetcher.metric_path"
)

const (
	dcRe          = `(@(?P<dc>[[:word:]\.\-\_]+))?`
	serviceNameRe = `(?P<name>[[:word:]\-\_]+)`
	nearRe        = `(~(?P<near>[[:word:]\.\-\_]+))?`
	tagRe         = `((?P<tag>[[:word:]=:\.\-\_]+)\.)?`
)

var (

	// CatalogServiceQueryRe is the regular expression to use.
	CatalogServiceQueryRe = regexp.MustCompile(`\A` + tagRe + serviceNameRe + dcRe + nearRe + `\z`)
)

func SearchToServiceSearch(search string) (ServiceSearch, error) {
	if !CatalogServiceQueryRe.MatchString(search) {
		return ServiceSearch{}, fmt.Errorf("catalog.service: invalid format: %q", search)
	}
	m := regexpMatch(CatalogServiceQueryRe, search)
	return ServiceSearch{
		Datacenter: m["dc"],
		Name:       m["name"],
		Near:       m["near"],
		Tag:        m["tag"],
	}, nil
}

type ServiceSearch struct {
	Datacenter string
	Name       string
	Near       string
	Tag        string
}

func (s ServiceSearch) String() string {
	name := s.Name
	if s.Tag != "" {
		name = s.Tag + "." + name
	}
	if s.Datacenter != "" {
		name = name + "@" + s.Datacenter
	}
	if s.Near != "" {
		name = name + "~" + s.Near
	}
	return fmt.Sprintf("catalog.service(%s)", name)
}

type Routes []*Route

type ServiceTags []string

type Route struct {
	ID              string
	Node            string
	Address         string
	Datacenter      string
	TaggedAddresses map[string]string
	NodeMeta        map[string]string
	ServiceID       string
	ServiceName     string
	ServiceAddress  string
	ServiceTags     ServiceTags
	ServiceMeta     map[string]string
	ServicePort     int
}

func (r *Route) FindScheme() string {
	for _, t := range r.ServiceTags {
		if strings.HasPrefix(t, SchemeTagsKey+"=") {
			return strings.TrimPrefix(t, SchemeTagsKey+"=")
		}
	}
	return ""
}

func (r *Route) FindMetricsPath() string {
	for _, t := range r.ServiceTags {
		if strings.HasPrefix(t, MetricPathTagsKey+"=") {
			return strings.TrimPrefix(t, MetricPathTagsKey+"=")
		}
	}
	return ""
}

// regexpMatch matches the given regexp and extracts the match groups into a
// named map.
func regexpMatch(re *regexp.Regexp, q string) map[string]string {
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(q, -1)

	if len(match) == 0 {
		return map[string]string{}
	}

	m := map[string]string{}
	for i, n := range match[0] {
		if names[i] != "" {
			m[names[i]] = n
		}
	}

	return m
}
