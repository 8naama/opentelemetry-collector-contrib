# Apache Doris Exporter
<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [alpha]: traces, metrics, logs   |
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aexporter%2Fdoris%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aexporter%2Fdoris) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aexporter%2Fdoris%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aexporter%2Fdoris) |
| Code coverage | [![codecov](https://codecov.io/github/open-telemetry/opentelemetry-collector-contrib/graph/main/badge.svg?component=exporter_doris)](https://app.codecov.io/gh/open-telemetry/opentelemetry-collector-contrib/tree/main/?components%5B0%5D=exporter_doris&displayType=list) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@atoulme](https://www.github.com/atoulme), [@joker-star-l](https://www.github.com/joker-star-l) |

[alpha]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
<!-- end autogenerated section -->

This exporter supports sending traces, metrics, and logs data to [Apache Doris](https://doris.apache.org/) (version >= 2.1.1). 

## Configuration

The following configuration options are supported:

* `endpoint` The http stream load address.
* `database` (default = otel) The database name.
* `username` The authentication username.
* `password` The authentication password.
* `table`
  * `logs` (default = otel_logs) The table name for logs.
  * `traces` (default = otel_traces) The table name for traces.
  * `metrics` (default = otel_metrics) The table name for metrics.
* `create_schema` (default = true) Whether databases and tables are created automatically in doris.
* `mysql_endpoint` The mysql protocol address of doris. Only use to create the schema; ignored if `create_schema` is false.
* `history_days` (default = 0) Data older than these days will be deleted; ignored if `create_schema` is false. If set to 0, historical data will not be deleted.
* `create_history_days` (default = 0) The number of days in the history partition that was created when the table was created; ignored if `create_schema` is false. If `history_days` is not 0, `create_history_days` needs to be less than or equal to `history_days`.
* `replication_num` (default = 1) The number of replicas of the table; ignored if `create_schema` is false.
* `timezone` (default is the time zone of the opentelemetry collector if IANA Time Zone Database is found, else is UTC) The time zone of doris, e.g. Asia/Shanghai.
* `log_response` (default = false) Whether to log the response of doris stream load.
* `label_prefix` (default = open_telemetry) the prefix of the label in doris stream load. The final generated label is {label_prefix}{db}{table}{yyyyMMddHHmmss}{uuid}.
* `headers` (default is empty map) The headers of doris stream load. Details: [header parameters](https://doris.apache.org/docs/data-operate/import/import-way/stream-load-manual#load-configuration-parameters) and [group commit](https://doris.apache.org/docs/data-operate/import/group-commit-manual#stream-load).
* `log_progress_interval` (default = 10) The interval, in seconds, between statistical logs. When it is less than or equal to 0, the statistical log is not printed.
* `sending_queue`  [details here](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/exporterhelper#configuration)
  * `enabled` (default = true)
  * `num_consumers` (default = 10) Number of consumers that dequeue batches; ignored if `enabled` is false.
  * `queue_size` (default = 1000) Maximum number of batches kept in memory before dropping; ignored if `enabled` is false.
* `retry_on_failure` [details here](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/exporterhelper#configuration)
  * `enabled` (default = true)
  * `initial_interval` (default = 5s) Time to wait after the first failure before retrying; ignored if `enabled` is false.
  * `max_interval` (default = 30s) The upper bound on backoff; ignored if `enabled` is false.
  * `max_elapsed_time` (default = 300s) The maximum amount of time spent trying to send a batch; ignored if `enabled` is false. If set to 0, the retries are never stopped.

The Doris exporter supports common [HTTP Configuration Settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md#http-configuration-settings), except for compression (all requests are uncompressed). As a consequence of supporting [confighttp](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md#http-configuration-settings), the Doris exporter also supports common [TLS Configuration Settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md#tls-configuration-settings).

The Doris exporter sets `timeout` (HTTP request timeout) to 60s by default. All other defaults are as defined by [confighttp](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md#http-configuration-settings).

Example:
```yaml
exporters:
  doris:
    endpoint: http://localhost:8030
    database: otel
    username: admin
    password: admin
    table:
      logs: otel_logs
      traces: otel_traces
      metrics: otel_metrics
    create_schema: true
    mysql_endpoint: localhost:9030
    history_days: 0
    create_history_days: 0
    replication_num: 1
    timezone: Asia/Shanghai
    timeout: 5s
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 1000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
```

## Notes

1. Time Zone

    The Doris exporter uses IANA Time Zone Database (known as tzdata) to handle time zones, so make sure tzdata is on your system.

    For example, when you use docker, you should add option `-v your/path/to/tzdata:/usr/share/zoneinfo` when running the container.
