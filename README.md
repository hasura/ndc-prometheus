# Prometheus Data Connector

The Hasura Prometheus Connector allows for connecting to a Prometheus database giving you an instant GraphQL API on top of your Prometheus data.

This connector is built using the [Go Data Connector SDK](https://github.com/hasura/ndc-sdk-go) and implements the [Data Connector Spec](https://github.com/hasura/ndc-spec).

## Features

### Instant Metrics

The connector can introspect and automatically transform available metrics on the Prometheus server to instant collection queries.

### Native Query

When simple queries don't meet your need you can define native queries in [the configuration file](./tests/configuration/configuration.yaml) with prepared variables with the `${<name>}` template.

```yaml
metadata:
  native_operations:
    queries:
      service_up:
        query: up{job="${job}", instance="${instance}"}
        labels: {}
        arguments:
          instance:
            type: String
          job:
            type: String
```

### Prometheus APIs

WIP
