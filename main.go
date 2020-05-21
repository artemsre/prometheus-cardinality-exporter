package main

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// stat holds the information about individual cardinality.
type stat struct {
	Name  string `json:"name"`
	Value uint64 `json:"value"`
}

// tsdbStatus has information of cardinality statistics from postings.
type tsdbStatus struct {
	SeriesCountByMetricName     []stat `json:"seriesCountByMetricName"`
	LabelValueCountByLabelName  []stat `json:"labelValueCountByLabelName"`
	MemoryInBytesByLabelName    []stat `json:"memoryInBytesByLabelName"`
	SeriesCountByLabelValuePair []stat `json:"seriesCountByLabelValuePair"`
}

func main() {
	promURL := os.Getenv("PROMETHEUS")
	timeout := 10
	if len(os.Getenv("PROMETHEUS_TIMEOUT")) > 0 {
		tmp, err := strconv.Atoi(os.Getenv("PROMETHEUS_TIMEOUT"))
		if err == nil {
			timeout = tmp
		} else {
			print("Can't parse PROMETHEUS_TIMEOUT env:" + err.Error())
		}
	}
	series_count := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "prometheus_series_count",
		Help: "series count by series name",
	}, []string{"metric"})
	labels_count := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "prometheus_labels_count",
		Help: "labels count by label name",
	}, []string{"label"})
	prometheus.MustRegister(series_count)
	prometheus.MustRegister(labels_count)
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: time.Duration(timeout) * time.Second,
		}).Dial,
		TLSHandshakeTimeout: time.Duration(timeout) * time.Second,
	}
	var netClient = &http.Client{
		Timeout:   time.Second * time.Duration(timeout),
		Transport: netTransport,
	}
	for {
		res, err := netClient.Get(promURL + "/api/v1/status/tsdb")
		if err != nil {
			print("Can't connect to " + promURL + "/api/v1/status/tsdb " + err.Error())
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			print("Can't read from socket:" + err.Error())
		}
		var data tsdbStatus
		err = json.Unmarshal(body, &data)
		if err != nil {
			print("Can't parse json:" + err.Error())
		}
		for _, d := range data.SeriesCountByMetricName {
			print(d.Name + " " + strconv.Itoa(int(d.Value)) + "\n")
		}
		time.Sleep(60 * time.Second)
	}
}
