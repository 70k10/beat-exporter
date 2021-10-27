# beat-exporter for Prometheus ![](https://github.com/70k10/beat-exporter/workflows/test-and-build/badge.svg)

Exposes multiple instances of (file|metric)beat statistics from beats statistics endpoint to prometheus format, automatically configuring collectors for appropriate beat type.

Current coverage
-

 * filebeat
 * metricbeat
 * packetbeat - _partial_
 * auditbeat - _partial_

Setup
-

Edit your *beat configuration and add following:

```
http:
  enabled: true
  host: localhost
  port: 5066
```

This will expose `(file|metrics|*)beat` http endpoint at given port.

Run beat-exporter:
```
$ ./beat-exporter
```

beat-exported default port for prometheus is: `9479`

Point your Prometheus to `0.0.0.0:9479/metrics`

Configuration reference
-
```
$ ./beat-exporter -help
Usage of ./beat-exporter:
  -beat.timeout duration
    	Timeout for trying to get stats from beat. (default 10s)
  -beat.uri string
    	HTTP API address of beat. (default "http://localhost:5066")
  -tls.certfile string
    	TLS certs file if you want to use tls instead of http
  -tls.keyfile string
    	TLS key file if you want to use tls instead of http
  -version
    	Show version and exit
  -web.listen-address string
    	Address to listen on for web interface and telemetry. (default ":9479")
  -web.telemetry-path string
    	Path under which to expose metrics. (default "/metrics")
```

Contribution
-
Please use pull requests, issues
