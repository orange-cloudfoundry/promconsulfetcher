
## Consul-template style query

This is the same query style described
here: https://github.com/hashicorp/consul-template/blob/master/docs/templating-language.md#service

It must have the form `<TAG>.<NAME>@<DATACENTER>~<NEAR>`

The `<TAG>` attribute is optional; if omitted, all nodes will be queried.

The `<DATACENTER>` attribute is optional; if omitted, the local datacenter is used.

The `<NEAR>` attribute is optional; if omitted, results are specified in lexical order. If provided a node name, results
are ordered by shortest round-trip time to the provided node. If provided `_agent`, results are ordered by shortest
round-trip time to the local agent.

## If metrics available on `/metrics` on your app

You have nothing to do, you can retrieve app instances metrics by simply call one of:

- [{{.BaseURL}}/v1/services/\[consul template style query\]/metrics]({{.BaseURL}}/v1/services/{consul template style query}/metrics)
- [{{.BaseURL}}/v1/services/metrics?consul_query="\[consul template style query\]"]({{.BaseURL}}/v1/services/metrics?consul_query="\[consul template style query\]")

## Set a different endpoint

## Consul tag

Add tag `promconsulfetcher.metric_path=/my-metrics/endpoint`

## In fetch URL

Add url param `metric_path=/my-metrics/endpoint`, e.g.:

- [{{.BaseURL}}/v1/services/\[consul template style query\]/metrics?metric_path=/my-metrics/endpoint]({{.BaseURL}}/v1/services/{consul template style query}/metrics?metric_path=/my-metrics/endpoint)

## Set a different scheme

## Consul tag

Add tag `promconsulfetcher.scheme=https`

## In fetch URL

Add url param `scheme=https`, e.g.:

- [{{.BaseURL}}/v1/services/\[consul template style query\]/metrics?scheme=https]({{.BaseURL}}/v1/services/{consul template style query}/metrics?scheme=https)

## Pass http headers to app, useful for authentication

If you do a request with headers, they are all passed to app.

This is useful for authentication purpose, example on basic auth

1. I have an app with metrics on `/metrics` but it is protected with basic auth `foo`/`bar`
2. You can perform curl: `curl https://foo:bar@{{.BaseURL}}/v1/services/my-app/metrics`
3. Basic auth header are passed to app and you can retrieve information (note that promconsulfetcher do not store anything)

## Retrieving only metrics from your app and not those from external

Use `/only-app-metrics` instead of `/metrics`, e.g.:

- [{{.BaseURL}}/v1/services/\[org_name\]/\[space_name\]/\[app_name\]/only-app-metrics]({{.BaseURL}}/v1/services/{org_name}/{space_name}/{app_name}/only-app-metrics)
- [{{.BaseURL}}/v1/services/only-app-metrics?app="\[app_id\]"]({{.BaseURL}}/v1/services/only-app-metrics?app="\[app_id\]")
