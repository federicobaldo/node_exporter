package collector

import (
	"os"
	"testing"
)

func TestSockStat(t *testing.T) {
	file, err := os.Open("fixtures/sockstat")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	sockStat, err := parseSockStat(file)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := 5.0, memInfo["tcp_inuse"]; want != got {
		t.Errorf("want tcp_inuse %f, got %f", want, got)
	}

	if want, got := 8.0, memInfo["udp_inuse"]; want != got {
		t.Errorf("want udp_inuse %f, got %f", want, got)
	}
}
