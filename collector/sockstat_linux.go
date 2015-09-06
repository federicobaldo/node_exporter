// +build !nosockstat

package collector

import (
	"bufio"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	procNetSockStat   = "/proc/net/sockstat"
	sockStatSubsystem = "sockstat"
)

type sockStatCollector struct {
	metrics map[string]prometheus.Gauge
}

func init() {
	Factories["sockstat"] = NewSockStatCollector
}

func NewSockStatCollector() (Collector, error) {
	return &sockStatCollector{
		metrics: map[string]prometheus.Gauge{},
	}, nil
}

func (c *sockStatCollector) Update(ch chan<- prometheus.Metric) (err error) {
	sockStat, err := getSockStat()
	if err != nil {
		return fmt.Errorf("Couldn't get sockstat %s", err)
	}
	log.Debugf("Set sockstat: %#v", sockStat)
	for k, v := range sockStat {
		if _, ok := c.metrics[k]; !ok {
			c.metrics[k] = prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: sockStatSubsystem,
				Name:      k,
				Help:      k + " from /proc/socketstat.",
			})
		}
		c.metrics[k].Set(v)
		c.metrics[k].Collect(ch)
	}
	return err
}

func getSockStat() (map[string]float64, error) {
	file, err := os.Open(procNetSockStat)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseSockStat(file)
}

func parseSockStat(r io.Reader) (map[string]float64, error) {
	var (
		sockStat = map[string]float64{}
		scanner  = bufio.NewScanner(r)

		re = regexp.MustCompile("sockets: used \\d+\n" +
			"TCP: inuse (?P<tcp_inuse>\\d+) orphan (?P<orphans>\\d+)" +
			" tw (?P<tw_count>\\d+) alloc (?P<tcp_sockets>\\d+)" +
			" mem (?P<tcp_pages>\\d+)\n" +
			"UDP: inuse (?P<udp_inuse>\\d+)" +
			//UDP memory accounting was added in v2.6.25-rc1
			"(?: mem (?P<udp_pages>\\d+))?\n" +
			//UDP-Lite (RFC 3828) was added in v2.6.20-rc2
			"(?:UDPLITE: inuse (?P<udplite_inuse>\\d+)\n)?" +
			"RAW: inuse (?P<raw_inuse>\\d+)\n" +
			"FRAG: inuse (?P<ip_frag_nqueues>\\d+)" +
			" memory (?P<ip_frag_mem>\\d+)\n")
	)
	names := re.SubexpNames()
	bytesArray, err := ReadAll(scanner)

	match := re.FindSubmatch(bytesArray)

	for k, v := range match {
		sockStat[names[k]] = float64(v)
	}

	return sockStat, err
}
