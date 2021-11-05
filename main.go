package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/70k10/beat-exporter/collector"
	"github.com/70k10/beat-exporter/internal/service"
)

const (
	serviceName = "beat_exporter"
)

func main() {
	var (
		Name          = serviceName
		listenAddress = flag.String("web.listen-address", ":9479", "Address to listen on for web interface and telemetry.")
		tlsCertFile   = flag.String("tls.certfile", "", "TLS certs file if you want to use tls instead of http")
		tlsKeyFile    = flag.String("tls.keyfile", "", "TLS key file if you want to use tls instead of http")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		beatURI       = flag.String("beat.uri", "http://localhost:5066", "HTTP API address of beat.\n" +
			"Comma-separated for multiple URIs. Ex. \"http://localhost:5066,http://localhost:5067\"\n" +
			"Append semi-colon to URI followed by a name to modify the collector label. Ex. \"http://localhost:5066;servicefilebeat\"\n")
		beatTimeout   = flag.Duration("beat.timeout", 10*time.Second, "Timeout for trying to get stats from beat.")
		showVersion   = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Print(version.Print(Name))
		os.Exit(0)
	}

	log.SetLevel(log.DebugLevel)

	log.SetFormatter(&log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyMsg: "message",
		},
	})

	stopCh := make(chan bool)

	err := service.SetupServiceListener(stopCh, serviceName, log.StandardLogger())
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorf("Could not setup service listener: %v", err)
	}

	// version metric
	registry := prometheus.NewRegistry()
	versionMetric := version.NewCollector(Name)
	registry.MustRegister(versionMetric)
	var collectorLabel string

	for _, URI := range strings.Split(*beatURI,",") {
		if len(URI) > 0 {
			URI, collectorLabel = parseCollectorLabel(URI)
			parsedURL, err := url.Parse(URI)
			if err != nil {
				log.Fatalf("Failed to parse beat.uri, error: %v", err)
			}
			httpClient := &http.Client{
				Timeout: *beatTimeout,
			}

			if parsedURL.Scheme == "unix" {
				unixPath := parsedURL.Path
				parsedURL.Scheme = "http"
				parsedURL.Host = "localhost"
				parsedURL.Path = ""
				httpClient.Transport = &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						return (&net.Dialer{}).DialContext(ctx, "unix", unixPath)
					},
				}
			}

			log.WithFields(log.Fields{"URI": URI}).Info("Validating Beat URI accessible.")

			t := time.NewTicker(1 * time.Second)

		    beatValidation:
			for {
				select {
				case <-t.C:
					err = testBeatStats(httpClient, *parsedURL)
					if err != nil {
						log.Errorf("Failed to connect to Beat, with error: %v, retrying in 1s", err)
						continue
					}

					break beatValidation

				case <-stopCh:
					os.Exit(0) // signal received, stop gracefully
				}
			}

			t.Stop()

			parsedCollectorLabel, beatInfo, beatCollector := collector.NewMainCollector(httpClient, parsedURL, Name, collectorLabel)
			registry.MustRegister(beatCollector)

			log.WithFields(
				log.Fields{
					"beat":     beatInfo.Beat,
					"version":  beatInfo.Version,
					"name":     beatInfo.Name,
					"hostname": beatInfo.Hostname,
					"uuid":     beatInfo.UUID,
				}).Infof("%s: Target beat configuration loaded successfully!", parsedCollectorLabel)
		}
	}

	http.Handle(*metricsPath, promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			ErrorLog:           log.New(),
			DisableCompression: false,
			ErrorHandling:      promhttp.ContinueOnError}),
	)

	http.HandleFunc("/", IndexHandler(*metricsPath))


	go func() {
		defer func() {
			stopCh <- true
		}()

		log.WithFields(log.Fields{
			"addr": *listenAddress,
		}).Info("Starting listener")

		if *tlsCertFile != "" && *tlsKeyFile != "" {
			if err := http.ListenAndServeTLS(*listenAddress, *tlsCertFile, *tlsKeyFile, nil); err != nil {

				log.WithFields(log.Fields{
					"err": err,
				}).Errorf("tls server quit with error: %v", err)

			}
		} else {
			if err := http.ListenAndServe(*listenAddress, nil); err != nil {

				log.WithFields(log.Fields{
					"err": err,
				}).Errorf("http server quit with error: %v", err)

			}
		}
		log.Info("Listener exited")
	}()

	for {
		if <-stopCh {
			log.Info("Shutting down beats exporter")
			break
		}
	}
}

// IndexHandler returns a http handler with the correct metricsPath
func IndexHandler(metricsPath string) http.HandlerFunc {

	indexHTML := `
<html>
	<head>
		<title>Beat Exporter</title>
	</head>
	<body>
		<h1>Beat Exporter</h1>
		<p>
			<a href='%s'>Metrics</a>
		</p>
	</body>
</html>
`
	index := []byte(fmt.Sprintf(strings.TrimSpace(indexHTML), metricsPath))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(index)
	}
}

func parseCollectorLabel(URI string) (string,string) {
	splitURIandLabel := strings.Split(URI,";")
	if len(splitURIandLabel) > 1 && len(splitURIandLabel[1]) > 0 {
		return splitURIandLabel[0], splitURIandLabel[1]
	}
	return splitURIandLabel[0], ""
}

func testBeatStats(client *http.Client, url url.URL) error {
	beatInfo := &collector.BeatInfo{}

	response, err := client.Get(url.String())
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Errorf("Beat URL: %q status code: %d", url.String(), response.StatusCode)
		return err
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Can't read body of response")
		return err
	}

	err = json.Unmarshal(bodyBytes, &beatInfo)
	if err != nil {
		log.Error("Could not parse JSON response for target")
		return err
	}
	return nil
}
