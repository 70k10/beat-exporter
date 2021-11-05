package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type mainCollector struct {
	Collectors map[string]prometheus.Collector
	Stats      *Stats
	client     *http.Client
	beatURL    *url.URL
	name       string
	targetDesc *prometheus.Desc
	targetUp   *prometheus.Desc
	metrics    exportedMetrics
	CollectorLabel string
	beatInfo   *BeatInfo
}

// NewMainCollector constructor
func NewMainCollector(client *http.Client, url *url.URL, name string, collectorLabel string) (string, BeatInfo, prometheus.Collector) {
	if collectorLabel == "" {
		collectorLabel = fmt.Sprintf("%s:%s", url.Hostname(), url.Port())
	}
	beat := &mainCollector{
		Collectors: make(map[string]prometheus.Collector),
		Stats:      &Stats{},
		client:     client,
		beatURL:    url,
		name:       name,
		CollectorLabel: collectorLabel,
		metrics:    exportedMetrics{},
		beatInfo: &BeatInfo{},
	}

	err := beat.loadBeatType(client, url)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorf("Failed to load beat type (%s): %v", beat.CollectorLabel, err)
	}

	beat.targetDesc = prometheus.NewDesc(
		prometheus.BuildFQName(name, "target", "info"),
		"target information",
		nil,
		prometheus.Labels{"version": beat.beatInfo.Version, "beat": beat.beatInfo.Beat, "collector": beat.CollectorLabel})

	beat.targetUp = prometheus.NewDesc(
		prometheus.BuildFQName("", beat.beatInfo.Beat, "up"),
		"Target up",
		nil,
		prometheus.Labels{"collector": beat.CollectorLabel})

	beat.Collectors["beat"] = NewBeatCollector(beat.beatInfo, beat.Stats, beat.CollectorLabel)
	beat.Collectors["libbeat"] = NewLibBeatCollector(beat.beatInfo, beat.Stats, beat.CollectorLabel)
	beat.Collectors["registrar"] = NewRegistrarCollector(beat.beatInfo, beat.Stats, beat.CollectorLabel)
	beat.Collectors["filebeat"] = NewFilebeatCollector(beat.beatInfo, beat.Stats, beat.CollectorLabel)
	beat.Collectors["metricbeat"] = NewMetricbeatCollector(beat.beatInfo, beat.Stats, beat.CollectorLabel)
	beat.Collectors["auditd"] = NewAuditdCollector(beat.beatInfo, beat.Stats, beat.CollectorLabel)

	return collectorLabel, beat.GetCollectorInfo(), beat
}

// Describe returns all descriptions of the collector.
func (b *mainCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- b.targetDesc
	ch <- b.targetUp

	for _, metric := range b.metrics {
		ch <- metric.desc
	}

	// standard collectors for all types of beats
	b.Collectors["beat"].Describe(ch)
	b.Collectors["libbeat"].Describe(ch)
	b.Collectors["auditd"].Describe(ch)

	// Customized collectors per beat type
	switch b.beatInfo.Beat {
	case "filebeat":
		b.Collectors["filebeat"].Describe(ch)
		b.Collectors["registrar"].Describe(ch)
	case "metricbeat":
		b.Collectors["metricbeat"].Describe(ch)
	}

}

// Collect returns the current state of all metrics of the collector.
func (b *mainCollector) Collect(ch chan<- prometheus.Metric) {

	err := b.fetchStatsEndpoint()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(b.targetUp, prometheus.GaugeValue, float64(0)) // set target down
		log.Errorf("Failed getting /stats endpoint of target: " + err.Error())
		return
	}

	ch <- prometheus.MustNewConstMetric(b.targetDesc, prometheus.GaugeValue, float64(1))
	ch <- prometheus.MustNewConstMetric(b.targetUp, prometheus.GaugeValue, float64(1)) // target up

	for _, i := range b.metrics {
		ch <- prometheus.MustNewConstMetric(i.desc, i.valType, i.eval(b.Stats))
	}

	// standard collectors for all types of beats
	b.Collectors["beat"].Collect(ch)
	b.Collectors["libbeat"].Collect(ch)
	b.Collectors["auditd"].Collect(ch)

	// Customized collectors per beat type
	switch b.beatInfo.Beat {
	case "filebeat":
		b.Collectors["filebeat"].Collect(ch)
		b.Collectors["registrar"].Collect(ch)
	case "metricbeat":
		b.Collectors["metricbeat"].Collect(ch)
	}

}

func (b *mainCollector) fetchStatsEndpoint() error {

	response, err := b.client.Get(b.beatURL.String() + "/stats")
	if err != nil {
		log.Errorf("Could not fetch stats endpoint of target: %v", b.beatURL.String())
		return err
	}

	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Can't read body of response")
		return err
	}

	err = json.Unmarshal(bodyBytes, &b.Stats)
	if err != nil {
		log.Error("Could not parse JSON response for target")
		return err
	}

	return nil
}

func (b *mainCollector) loadBeatType(client *http.Client, url *url.URL) error {
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

	err = json.Unmarshal(bodyBytes, &b.beatInfo)
	if err != nil {
		log.Error("Could not parse JSON response for target")
		return err
	}

	return nil
}

func (b *mainCollector) GetCollectorInfo() BeatInfo {
	bi := BeatInfo{b.beatInfo.Beat, b.beatInfo.Hostname, b.beatInfo.Name, b.beatInfo.UUID, b.beatInfo.Version}
    return bi
}