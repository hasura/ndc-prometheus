# Prometheus Data Connector

The Hasura Prometheus Connector allows for connecting to a Prometheus database, giving you an instant GraphQL API on top of your Prometheus data.

This connector is built using the [Go Data Connector SDK](https://github.com/hasura/ndc-sdk-go) and implements the [Data Connector Spec](https://github.com/hasura/ndc-spec).

## Features

### Metrics

#### How it works

The connector can introspect and automatically transform available metrics on the Prometheus server into collection queries. Each metric has a common structure:

```gql
{
  <label_1>
  <label_2>
  # ...
  timestamp
  value
  labels
  values {
    timestamp
    value
  }
}
```

> [!NOTE]
> Labels and metrics are introspected from the Prometheus server at the current time. You need to introspect again whenever there are new labels or metrics.

The configuration plugin introspects labels of each metric and defines them as collection columns that enable the ability of Hasura permissions and remote join. The connector supports basic comparison filters for labels.

```gql
{
  process_cpu_seconds_total(
    where: {
      timestamp: { _gt: "2024-09-24T10:00:00Z" }
      job: {
        _eq: "node"
        _neq: "prometheus"
        _in: ["node", "prometheus"]
        _nin: ["ndc-prometheus"]
        _regex: "prometheus.*"
        _nregex: "foo.*"
      }
    }
    args: { step: "5m", offset: "5m", timeout: "30s" }
  ) {
    job
    instance
    timestamp
    value
    values {
      timestamp
      value
    }
  }
}
```

The connector can detect if you want to request an instant query or range query via the `timestamp` column:

- `_eq`: instant query at the exact timestamp.
- `_gt` < `_lt`: range query.

The range query mode is the default if none of the timestamp operators is set.

The `timestamp` and `value` fields are the result of the instant query. If the request is a range query, `timestamp` and `value` are picked as the last item of the `values` series.

#### Common arguments

- `step`: the query resolution step width in duration format or float number of seconds. The step should be explicitly set for range queries. Even though the connector can estimate the approximate step width, the result may be empty due to too large an interval.
- `offset`: the offset modifier allows changing the time offset for individual instant and range vectors in a query.
- `timeout`: the evaluation timeout of the request.
- `fn`: the array of composable PromQL functions.
- `flat`: flatten grouped values out of the root array. Use the runtime setting if the value is null.

#### Aggregation

The `fn` argument is an array of [PromQL function](https://prometheus.io/docs/prometheus/latest/querying/functions/) parameters. You can set multiple functions that can be composed into the query. For example, with this PromQL query:

```
sum by (job) (rate(process_cpu_seconds_total[1m]))
```

The equivalent GraphQL query will be:

```gql
{
  process_cpu_seconds_total(
    where: { timestamp: { _gt: "2024-09-24T10:00:00Z" } }
    args: { step: "5m", fn: [{ rate: "5m" }, { sum: [job] }] }
  ) {
    job
    timestamp
    value
    values {
      timestamp
      value
    }
  }
}
```

### Native Query

#### How it works

When simple queries don't meet your need you can define native queries in [the configuration file](./tests/configuration/configuration.yaml) with prepared variables with the `${<name>}` template. Native queries are defined as collections.

```yaml
metadata:
  native_operations:
    queries:
      service_up:
        query: up{job="${job}", instance="${instance}"}
        labels:
          instance: {}
          job: {}
        arguments:
          instance:
            type: String
          job:
            type: String
```

The native query is exposed as a read-only function with 2 required fields `job` and `instance`.

```gql
{
  service_up(
    args: { step: "1m", job: "node", instance: "node-exporter:9100" }
    where: {
      timestamp: { _gt: "2024-10-11T00:00:00Z" }
      job: { _in: ["node"] }
    }
  ) {
    job
    instance
    values {
      timestamp
      value
    }
  }
}
```

> [!NOTE]
> Labels aren't automatically added. You need to define them manually.

> [!NOTE]
> Label and value boolean expressions in `where` are used to filter results after the query is executed.

### Prometheus APIs

#### Raw PromQL query

Execute a raw PromQL query directly. This API should only be used by the admin. The result contains labels and values only.

```gql
{
  promql_query(
    query: "process_cpu_seconds_total{job=\"node\"}"
    start: "2024-09-24T10:00:00Z"
    step: "5m"
  ) {
    labels
    values {
      timestamp
      value
    }
  }
}
```

#### Metadata APIs

| Operation Name              | REST Prometheus API                                                                                                   |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| prometheus_alertmanagers    | [/api/v1/alertmanagers](https://prometheus.io/docs/prometheus/latest/querying/api/#alertmanagers)                     |
| prometheus_alerts           | [/api/v1/alerts](https://prometheus.io/docs/prometheus/latest/querying/api/#alerts)                                   |
| prometheus_label_names      | [/api/v1/labels](https://prometheus.io/docs/prometheus/latest/querying/api/#getting-label-names)                      |
| prometheus_label_values     | [/api/v1/label/<label_name>/values](https://prometheus.io/docs/prometheus/latest/querying/api/#querying-label-values) |
| prometheus_rules            | [/api/v1/rules](https://prometheus.io/docs/prometheus/latest/querying/api/#rules)                                     |
| prometheus_series           | [/api/v1/series](https://prometheus.io/docs/prometheus/latest/querying/api/#finding-series-by-label-matchers)         |
| prometheus_targets          | [/api/v1/targets](https://prometheus.io/docs/prometheus/latest/querying/api/#targets)                                 |
| prometheus_targets_metadata | [/api/v1/targets/metadata](https://prometheus.io/docs/prometheus/latest/querying/api/#querying-target-metadata)       |

## Configuration

### Authentication

#### Basic Authentication

```yaml
connection_settings:
  authentication:
    basic:
      username:
        env: PROMETHEUS_USERNAME
      password:
        env: PROMETHEUS_PASSWORD
```

#### HTTP Authorization

```yaml
connection_settings:
  authentication:
    authorization:
      type:
        value: Bearer
      credentials:
        env: PROMETHEUS_AUTH_TOKEN
```

#### OAuth2

```yaml
connection_settings:
  authentication:
    oauth2:
      token_url:
        value: http://example.com/oauth2/token
      client_id:
        env: PROMETHEUS_OAUTH2_CLIENT_ID
      client_secret:
        env: PROMETHEUS_OAUTH2_CLIENT_SECRET
```

#### Google Cloud

The configuration accepts either the Google application credentials JSON string or a file path. If the object is empty, the client automatically loads the credential file from the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.

```yaml
connection_settings:
  authentication:
    google:
      # credentials:
      #   env: GOOGLE_APPLICATION_CREDENTIALS_JSON
      # credentials_file:
      #   env: GOOGLE_APPLICATION_CREDENTIALS
```

### Runtime Settings

```yaml
runtime:
  promptql: false
  disable_prometheus_api: false # disable native REST Prometheus APIs 
  default_quantile: 0.95
  flat: false
  unix_time_unit: s # enum: s, ms
  format:
    timestamp: rfc3339 # enum: rfc3339, unix
    value: float64 # enum: string, float64
```

#### Flatten values

By default, values are grouped by the label set. If you want to flatten out the values array, set `flat=true`.

#### Unix timestamp's unit

If you use integer values for duration and timestamp fields, the connector will transform them with this unix timestamp unit. Accept second (`s`) and millisecond (`ms`). The default unit is seconds.

#### Response Format

These settings specify the format of the response timestamp and value.

## PromptQL Mode (experiment)

### How it works

The PromptQL mode, if enabled, transforms the connector schema to be compatible with PromptQL:

- Support aggregate functions and aggregate grouping.
- Disable arguments: PromptQL has some push-down issues with aggregations if the model has arguments.
- Instead, new prepared models are added for range functions: `rate`, `irate`, `increase`, `quantile`.
- The quantile ratio is fixed, which is configured in the runtime config.

### Examples

```
Calculate the aggregate sum of process CPU seconds rate in the last 1 hour, group by timestamp, job, instance. Timestamp must be converted to UTC timezone.
```

```
Calculate the sum of go gc heap allocs quantile in the last 1 hour,  group by timestamp, job, instance. Timestamps must be converted to UTC timezone.
```

## Development

### Get started

#### Start Docker services

```sh
docker composes up -d
```

#### Introspect the configuration file

```sh
make generate-test-config
docker compose restart ndc-prometheus
```

#### Introspect and build DDN metadata

```sh
make build-supergraph-test
docker compose up -d --build engine
```

Browse the engine console at http://localhost:3000.
