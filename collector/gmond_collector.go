// +build ganglia

package collector

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"regexp"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/node_exporter/collector/ganglia"
)

const (
	gangliaAddress       = "127.0.0.1:8649"
	gangliaProto         = "tcp"
	gangliaTimeout       = 30 * time.Second
	gangliaMetricsPrefix = "ganglia_"
)

type gmondCollector struct {
	Metrics map[string]*prometheus.GaugeVec
	config  Config
}

func init() {
	Factories["gmond"] = NewGmondCollector
}

var illegalCharsRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Takes a config struct and prometheus registry and returns a new Collector scraping ganglia.
func NewGmondCollector(config Config) (Collector, error) {
	c := gmondCollector{
		config:  config,
		Metrics: map[string]*prometheus.GaugeVec{},
	}

	return &c, nil
}

func (c *gmondCollector) setMetric(name, cluster string, metric ganglia.Metric) {
	if _, ok := c.Metrics[name]; !ok {
		var desc string
		var title string
		for _, element := range metric.ExtraData.ExtraElements {
			switch element.Name {
			case "DESC":
				desc = element.Val
			case "TITLE":
				title = element.Val
			}
			if title != "" && desc != "" {
				break
			}
		}
		glog.V(1).Infof("Register %s: %s", name, desc)
		gv := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: gangliaMetricsPrefix,
				Name:      name,
				Help:      desc,
			},
			[]string{"cluster"},
		)
		c.Metrics[name] = prometheus.MustRegisterOrGet(gv).(*prometheus.GaugeVec)
	}
	glog.V(1).Infof("Set %s{cluster=%q}: %f", name, cluster, metric.Value)
	c.Metrics[name].WithLabelValues(cluster).Set(metric.Value)
}

func (c *gmondCollector) Update() (updates int, err error) {
	conn, err := net.Dial(gangliaProto, gangliaAddress)
	glog.V(1).Infof("gmondCollector Update")
	if err != nil {
		return updates, fmt.Errorf("Can't connect to gmond: %s", err)
	}
	conn.SetDeadline(time.Now().Add(gangliaTimeout))

	ganglia := ganglia.Ganglia{}
	decoder := xml.NewDecoder(bufio.NewReader(conn))
	decoder.CharsetReader = toUtf8

	err = decoder.Decode(&ganglia)
	if err != nil {
		return updates, fmt.Errorf("Couldn't parse xml: %s", err)
	}

	for _, cluster := range ganglia.Clusters {
		for _, host := range cluster.Hosts {

			for _, metric := range host.Metrics {
				name := illegalCharsRE.ReplaceAllString(metric.Name, "_")

				c.setMetric(name, cluster.Name, metric)
				updates++
			}
		}
	}
	return updates, err
}

func toUtf8(charset string, input io.Reader) (io.Reader, error) {
	return input, nil //FIXME
}