package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

//Filebeat json structure
type Filebeat struct {
	Events struct {
		Active float64 `json:"active"`
		Added  float64 `json:"added"`
		Done   float64 `json:"done"`
	} `json:"events"`

	Harvester struct {
		Closed    float64 `json:"closed"`
		OpenFiles float64 `json:"open_files"`
		Running   float64 `json:"running"`
		Skipped   float64 `json:"skipped"`
		Started   float64 `json:"started"`
	} `json:"harvester"`

	Input struct {
		Log struct {
			Files struct {
				Renamed   float64 `json:"renamed"`
				Truncated float64 `json:"truncated"`
			} `json:"files"`
		} `json:"log"`
		Netflow struct {
			Flows float64 `json:"flows"`
			Packets struct {
			    Dropped float64 `json:"dropped"`
				Received float64 `json:"received"`
			} `json:"packets"`
		} `json:"netflow"`
	} `json:"input"`
}

type filebeatCollector struct {
	beatInfo *BeatInfo
	stats    *Stats
	metrics  exportedMetrics
}

// NewFilebeatCollector constructor
func NewFilebeatCollector(beatInfo *BeatInfo, stats *Stats, collectorLabel string) prometheus.Collector {
	return &filebeatCollector{
		beatInfo: beatInfo,
		stats:    stats,
		metrics: exportedMetrics{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "events"),
					"filebeat.events",
					nil, prometheus.Labels{"event": "active", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Events.Active },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "events"),
					"filebeat.events",
					nil, prometheus.Labels{"event": "added", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Events.Added },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "events"),
					"filebeat.events",
					nil, prometheus.Labels{"event": "done", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Events.Done },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "harvester"),
					"filebeat.harvester",
					nil, prometheus.Labels{"harvester": "closed", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Harvester.Closed },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "harvester"),
					"filebeat.harvester",
					nil, prometheus.Labels{"harvester": "open_files", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Harvester.OpenFiles },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "harvester"),
					"filebeat.harvester",
					nil, prometheus.Labels{"harvester": "running", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Harvester.Running },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "harvester"),
					"filebeat.harvester",
					nil, prometheus.Labels{"harvester": "skipped", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Harvester.Skipped },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "harvester"),
					"filebeat.harvester",
					nil, prometheus.Labels{"harvester": "started", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Harvester.Started },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "input_log"),
					"filebeat.input_log",
					nil, prometheus.Labels{"files": "renamed", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Input.Log.Files.Renamed },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "input_log"),
					"filebeat.input_log",
					nil, prometheus.Labels{"files": "truncated", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Input.Log.Files.Truncated },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "input_netflow_flows"),
					"filebeat.input_netflow",
					nil, prometheus.Labels{"collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Input.Netflow.Flows },
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "input_netflow"),
					"filebeat.input_netflow",
					nil, prometheus.Labels{"packets": "dropped", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Input.Netflow.Packets.Dropped},
				valType: prometheus.UntypedValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(beatInfo.Beat, "filebeat", "input_netflow"),
					"filebeat.input_netflow",
					nil, prometheus.Labels{"packets": "received", "collector": collectorLabel},
				),
				eval:    func(stats *Stats) float64 { return stats.Filebeat.Input.Netflow.Packets.Received},
				valType: prometheus.UntypedValue,
			},
		},
	}
}

// Describe returns all descriptions of the collector.
func (c *filebeatCollector) Describe(ch chan<- *prometheus.Desc) {

	for _, metric := range c.metrics {
		ch <- metric.desc
	}

}

// Collect returns the current state of all metrics of the collector.
func (c *filebeatCollector) Collect(ch chan<- prometheus.Metric) {

	for _, i := range c.metrics {
		ch <- prometheus.MustNewConstMetric(i.desc, i.valType, i.eval(c.stats))
	}

}
