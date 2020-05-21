package main

import (
	"bytes"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type alertAddCmd struct {
	annotations  []string
	generatorURL string
	labels       []string
	start        string
	end          string
}

type AlertType struct {
	Annotations  map[string]string
	GeneratorURL string
	Labels       map[string]string
	EndsAt       time.Time
}

func pushAlert(alert AlertType) error {
	alertURL := os.Getenv("ALERTMANAGER")
	if len(alertURL) < 5 {
		return nil
	}
	url := alertURL + "/api/v1/alerts"
	var alerts []AlertType
	alerts = append(alerts, alert)
	reqBody, err := json.Marshal(alerts)
	if err != nil {
		print("Can't create json body: " + err.Error())
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 5}
	resp, err := client.Do(req)
	if err != nil {
		print(err.Error())
		return err
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	//fmt.Println("response Body:", string(body))
	return err
}

// stat holds the information about individual cardinality.
type stat struct {
	Name  string `json:"name"`
	Value uint64 `json:"value"`
}

// tsdbStatus has information of cardinality statistics from postings.
type tsdbStatus struct {
	Data struct {
		SeriesCountByMetricName     []stat `json:"seriesCountByMetricName"`
		LabelValueCountByLabelName  []stat `json:"labelValueCountByLabelName"`
		MemoryInBytesByLabelName    []stat `json:"memoryInBytesByLabelName"`
		SeriesCountByLabelValuePair []stat `json:"seriesCountByLabelValuePair"`
	} `json:"data"`
}

func main() {
	promURL := os.Getenv("PROMETHEUS")
	if len(promURL) < 5 {
		print("You should define PROMETHEUS env url")
		os.Exit(1)
	}
	timeout := 10
	if len(os.Getenv("PROMETHEUS_TIMEOUT")) > 0 {
		tmp, err := strconv.Atoi(os.Getenv("PROMETHEUS_TIMEOUT"))
		if err == nil {
			timeout = tmp
		} else {
			print("Can't parse PROMETHEUS_TIMEOUT env:" + err.Error())
		}
	}
	series_count := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "SeriesCountByMetricName",
		Help: "series count by series name",
	}, []string{"metric"})
	labels_count := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "LabelValueCountByLabelName",
		Help: "labels count by label name",
	}, []string{"label"})
	memory_count := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "MemoryInBytesByLabelName",
		Help: "Memory count by label name",
	}, []string{"label"})
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
	errCount := 0
	activeAlert := 0
	for {
		a := AlertType{
			Labels:       map[string]string{"alertname": "Prometheus unresponsible", "severity": "medium", "env": "prod"},
			GeneratorURL: promURL,
			EndsAt:       time.Now().Add(time.Hour * 1),
		}
		if errCount > 5 {
			err := pushAlert(a)
			if err == nil {
				activeAlert = 1
			}
		}
		res, err := netClient.Get(promURL + "/api/v1/status/tsdb")
		if err != nil {
			print("Can't connect to " + promURL + "/api/v1/status/tsdb " + err.Error())
			errCount = errCount + 1
			time.Sleep(time.Second * 1)
			continue
		} else {
			errCount = 0
		}
		if activeAlert > 0 {
			if errCount == 0 {
				//resolve alerts
				a.EndsAt = time.Now()
				pushAlert(a)
			}
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
		for _, d := range data.Data.SeriesCountByMetricName {
			series_count.WithLabelValues(d.Name).Set(float64(d.Value))
		}
		for _, d := range data.Data.LabelValueCountByLabelName {
			labels_count.WithLabelValues(d.Name).Set(float64(d.Value))
		}
		for _, d := range data.Data.MemoryInBytesByLabelName {
			memory_count.WithLabelValues(d.Name).Set(float64(d.Value))
		}
		time.Sleep(2 * time.Second)
	}
}
