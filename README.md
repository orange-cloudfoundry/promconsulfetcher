# Promconsulfetcher

Promconsulfetcher was made for [consul](https://www.consul.io/) and the idea behind is to give ability to fetch metrics
from all services instance queried in a consul. This allow to aggregate metrics from a consul cluster where node and
their port is not accessible by a prometheus instance.

User can retrieve is metrics by simply call `/v1/services/[consul template style query]/metrics` which will merge all
metrics from service instances and add labels:

"node":            route.Node,
"id":              route.ID,
"address":         route.Address,
"datacenter":      route.Datacenter,
"service_name":    route.ServiceName,
"service_id":      route.ServiceID,
"service_address": route.ServiceAddress,
"service_port":    strconv.Itoa(route.ServicePort),

- `node`
- `id`
- `address`
- `datacenter`
- `service_id`
- `service_name`
- `service_address`
- `service_port`

## Example

Metrics from app instance 0:

```
go_memstats_mspan_sys_bytes{} 65536
```

Metrics from app instance 1:

```
go_memstats_mspan_sys_bytes{} 5600
```

become:

```
go_memstats_mspan_sys_bytes{node="node1",id="id1",address="host1",datacenter="dc1",service_id="id1",service_name="name",service_address="host1",service_port="4646",instance="172.76.112.90:61038"} 65536
go_memstats_mspan_sys_bytes{node="node2",id="id2",address="host2",datacenter="dc1",service_id="id2",service_name="name",service_address="host2",service_port="4646",instance="172.76.112.91:61010"} 65536
```

## How to use ?

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

- [my.promconsulfetcher.com/v1/services/\[consul template style query\]/metrics](my.promconsulfetcher.com/v1/services/{consul template style query}/metrics)
- [my.promconsulfetcher.com/v1/services/metrics?consul_query="\[consul template style query\]"](my.promconsulfetcher.com/v1/services/metrics?consul_query="\[consul template style query\]")

## Set a different endpoint

## Consul tag

Add tag `promconsulfetcher.metric_path=/my-metrics/endpoint`

## In fetch URL

Add url param `metric_path=/my-metrics/endpoint`, e.g.:

- [my.promconsulfetcher.com/v1/services/\[consul template style query\]/metrics?metric_path=/my-metrics/endpoint](my.promconsulfetcher.com/v1/services/{consul template style query}/metrics?metric_path=/my-metrics/endpoint)

## Set a different scheme

## Consul tag

Add tag `promconsulfetcher.scheme=https`

## In fetch URL

Add url param `scheme=https`, e.g.:

- [my.promconsulfetcher.com/v1/services/\[consul template style query\]/metrics?scheme=https](my.promconsulfetcher.com/v1/services/{consul template style query}/metrics?scheme=https)

## Pass http headers to app, useful for authentication

If you do a request with headers, they are all passed to app.

This is useful for authentication purpose, example on basic auth

1. I have an app with metrics on `/metrics` but it is protected with basic auth `foo`/`bar`
2. You can perform curl: `curl https://foo:bar@my.promconsulfetcher.com/v1/services/my-app/metrics`
3. Basic auth header are passed to app and you can retrieve information (note that promconsulfetcher do not store
   anything)

## Retrieving only metrics from your app and not those from external

Use `/only-app-metrics` instead of `/metrics`, e.g.:

- [my.promconsulfetcher.com/v1/services/\[org_name\]/\[space_name\]/\[app_name\]/only-app-metrics](my.promconsulfetcher.com/v1/services/{org_name}/{space_name}/{app_name}/only-app-metrics)
- [my.promconsulfetcher.com/v1/services/only-app-metrics?app="\[app_id\]"](my.promconsulfetcher.com/v1/services/only-app-metrics?app="\[app_id\]")

## How to deploy ?

Please download [latest release](/releases) for your platform and run it with `./promconsulfetcher`, you will now have
access to `http://localhost:8085` which is the user doc.

### Configure

Of course, default configuration will not work in most context, to configure you write a `config.yml` and configure as
described:

For understanding config definition format:

- `[]` means optional (by default parameter is required)
- `<>` means type to use

### Root configuration in config.yml

```yaml
# Port for listening
[ port: <int> | default = 8085]

# Port for listening health check
[ health_check_port: <int> | default = 8080]

# External url which give docs url to user
# This is an url pointing on this service of course
# using it let you separate logs part from user part
[ base_url: <string> | default = "http://localhost:8085" ]

# skip ssl validation when connecting to services found
[ skip_ssl_validation: <bool> ]

# set to true to enable ssl server
# you will need to set `tls_pem.cert_chain` and `tls_pem.private_key`
[ enable_ssl: <bool> ]

# CA(s) in pem format (multiple can bet set inside) 
# which will be use when connecting on service
[ ca_certs: <string> ]

# tls cert and private key in pem format if you want to enable ssl
tls_pem:
  cert_chain: <string>
  private_key: <string>

log:
  # log level to use for server
  # you can chose: `trace`, `debug`, `info`, `warn`, `error`, `fatal` or `panic`
  [ level: <string> | default = info ]
  # Set to true to force not have color when seeing logs
  [ no_color: <bool> ]
  # et to true to see logs as json format
  [ in_json: <bool> ]

# consul configuration for connecting
consul:
  # Defines the address of the Consul server
  [ address: <string> | default = "127.0.0.1:8500" ]
  
  # Defines the URI scheme for the Consul server
  [ scheme: <string> | default = "http" ]
  
  # Defines the datacenter to use.
  # If not provided, Consul uses the default agent datacenter
  [ datacenter: <string> ]
  
  # Token is used to provide a per-request ACL 
  # token which overwrites the agent's default token.
  [ token: <string> ]
  
  # Limits the duration for which a Watch can block. 
  # If not provided, the agent default values will be used.
  [ endpoint_wait_time: <string> ]
  
  # Defines the TLS configuration used for the secure connection to Consul Catalog
  tls:
    # the path to the certificate authority used for the secure 
    # connection to Consul Catalog, it defaults to the system bundle
    [ ca: <string> ]
    # the path to the public certificate used for the secure 
    # connection to Consul Catalog. When using this option, 
    # setting the key option is required
    [ cert: <string> ]
    # the path to the private key used for the secure 
    # connection to Consul Catalog. When using this option, 
    # setting the cert option is required
    [ key: <string> ]
    # if true, the TLS connection to Consul accepts any certificate 
    # presented by the server regardless of the hostnames it covers
    [ skip_ssl_validation: <bool> ]
  
  # Used to authenticate the HTTP client using HTTP Basic Authentication
  http_auth:
    # Username to use for HTTP Basic Authentication
    [ username: <string> ]
    # Password to use for HTTP Basic Authentication
    [ password: <string> ]

```

## Metrics

Promconsulfetcher expose metrics on `/metrics`:

- `promconsulfetcher_metric_fetch_failed_total`: Number of non fetched metrics without be an normal error.
- `promconsulfetcher_metric_fetch_success_total`: Number of fetched metrics succeeded for an app (app instance call are
  summed).
- `promconsulfetcher_latest_time_scrape_route`: Last time that route has been scraped in seconds.
- `promconsulfetcher_scrape_route_failed_total`: Number of non fetched metrics without be an normal error.

## Graceful shutdown

Promconsulfetcher when receiving a SIGINT or SIGTERM or SIGUSR1 signal will stop listening new connections and will wait
to finish opened requests before stopping. If opened requests are not finished after 15 seconds the server will be hard
closed.

## Health Check

Health check is available by default on port 8080. If promconsulfetcher is not healthy or not yet healthy it will
respond a 503 error, if not it will respond a 200.

User can send a `USR1` signal on promconsulfetcher to set unhealthy on health check in addition to stop gracefully.
