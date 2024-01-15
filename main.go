package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var initializing bool = true
var client = http.Client{
	Timeout: 5 * time.Second,
}
var exportedMetrics = []map[string]interface{}{}

var token string
var periodEnvVar string
var period time.Duration
var port = 9101

// Define a struct for you collector that contains pointers
// to prometheus descriptors for each metric you wish to expose.
// Note you can also include fields of other types if they provide utility
// but we just won't be exposing them as metrics.
type socketCollector struct {
	socketMetric *prometheus.Desc
}

type NpmPackage struct {
	Name    string
	Version string
}

type NpmObject struct {
	Package NpmPackage
}

type NpmResponse struct {
	Objects []NpmObject `json:"objects"`
}

type SupplyChainRiskResponse struct {
	Score float64 `json:"score"`
}

type QualityResponse struct {
	Score float64 `json:"score"`
}

type MaintenanceResponse struct {
	Score float64 `json:"score"`
}

type VulnerabilityResponse struct {
	Score float64 `json:"score"`
}

type LicenseResponse struct {
	Score float64 `json:"score"`
}

type MiscellaneousResponse struct {
	Score float64 `json:"score"`
}

type SocketResponse struct {
	Supplychainrisk SupplyChainRiskResponse `json:"supplyChainRisk"`
	Quality         QualityResponse         `json:"quality"`
	Maintenance     MaintenanceResponse     `json:"maintenance"`
	Vulnerability   VulnerabilityResponse   `json:"vulnerability"`
	License         LicenseResponse         `json:"license"`
	Miscellaneous   MiscellaneousResponse   `json:"miscellaneous"`
}

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func newSocketCollector() *socketCollector {
	return &socketCollector{
		socketMetric: prometheus.NewDesc("socket_score",
			"Shows socket.dev packages scores",
			[]string{"package", "version", "score"}, nil,
		),
	}
}

// Each and every collector must implement the Describe function.
// It essentially writes all descriptors to the prometheus desc channel.
func (collector *socketCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.socketMetric
}

// Collect implements required collect function for all promehteus collectors
func (collector *socketCollector) Collect(ch chan<- prometheus.Metric) {

	logrus.Info("Received HTTP request")
	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.
	logrus.Info(fmt.Sprintf("Sending metrics to Prometheus channel"))
	for i := range exportedMetrics {
		s, err := strconv.ParseFloat(fmt.Sprintf("%v", exportedMetrics[i]["value"]), 64)
		if err != nil {
			logrus.Error(fmt.Sprintf("Error converting metric value: %s", err))
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			collector.socketMetric,
			prometheus.GaugeValue,
			s,
			fmt.Sprintf("%v", exportedMetrics[i]["name"]),
			fmt.Sprintf("%v", exportedMetrics[i]["version"]),
			fmt.Sprintf("%v", exportedMetrics[i]["score"]),
		)
	}

}

func updateMetrics() {

	var npmResponse NpmResponse

	logrus.Info("Sending request to registry.npmjs.org")
	res, err := client.Get("https://registry.npmjs.org/-/v1/search?text=scope:celo&size=100")
	if err != nil {
		logrus.Error(fmt.Sprintf("Error making http request to registry.npmjs.org: %s", err))
		return
	}

	err = json.NewDecoder(res.Body).Decode(&npmResponse)
	if err != nil {
		logrus.Error(fmt.Sprintf("Could not decode response body from registry.npmjs.org: %s", err))
		return
	}

	for i := range npmResponse.Objects {
		logrus.Info(fmt.Sprintf("Requesting package %s/%s scores to api.socket.dev", npmResponse.Objects[i].Package.Name, npmResponse.Objects[i].Package.Version))

		req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.socket.dev/v0/npm/%s/%s/score", npmResponse.Objects[i].Package.Name, npmResponse.Objects[i].Package.Version), nil)
		req.Header.Add("accept", "application/json")
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(token)))
		res, err := client.Do(req)
		if err != nil {
			logrus.Error(fmt.Sprintf("Error requesting package %s to api.socket.dev: %s", npmResponse.Objects[i].Package.Name, err))
			continue
		}

		var socketResponse SocketResponse
		err = json.NewDecoder(res.Body).Decode(&socketResponse)
		if err != nil {
			logrus.Error(fmt.Sprintf("Could not decode response body from api.socket.dev: %s", err))
			continue
		}

		logrus.Debug(fmt.Sprintf("Socket supply chain risk score: %f", socketResponse.Supplychainrisk.Score))
		logrus.Debug(fmt.Sprintf("Socket quality score: %f", socketResponse.Quality.Score))
		logrus.Debug(fmt.Sprintf("Socket maintenance score: %f", socketResponse.Maintenance.Score))
		logrus.Debug(fmt.Sprintf("Socket vulnerability score: %f", socketResponse.Vulnerability.Score))
		logrus.Debug(fmt.Sprintf("Socket license score: %f", socketResponse.License.Score))
		logrus.Debug(fmt.Sprintf("Socket miscellaneous score: %f", socketResponse.Miscellaneous.Score))

		metricSupplyChainRisk := map[string]interface{}{"name": npmResponse.Objects[i].Package.Name, "version": npmResponse.Objects[i].Package.Version, "score": "supplychainrisk", "value": socketResponse.Supplychainrisk.Score}
		metricQuality := map[string]interface{}{"name": npmResponse.Objects[i].Package.Name, "version": npmResponse.Objects[i].Package.Version, "score": "quality", "value": socketResponse.Quality.Score}
		metricMaintenance := map[string]interface{}{"name": npmResponse.Objects[i].Package.Name, "version": npmResponse.Objects[i].Package.Version, "score": "maintenance", "value": socketResponse.Maintenance.Score}
		metricVulnerability := map[string]interface{}{"name": npmResponse.Objects[i].Package.Name, "version": npmResponse.Objects[i].Package.Version, "score": "vulneravility", "value": socketResponse.Vulnerability.Score}
		metricLicense := map[string]interface{}{"name": npmResponse.Objects[i].Package.Name, "version": npmResponse.Objects[i].Package.Version, "score": "license", "value": socketResponse.License.Score}
		metricMiscellaneous := map[string]interface{}{"name": npmResponse.Objects[i].Package.Name, "version": npmResponse.Objects[i].Package.Version, "score": "miscellaneous", "value": socketResponse.Miscellaneous.Score}

		exportedMetrics = append(exportedMetrics, metricSupplyChainRisk)
		exportedMetrics = append(exportedMetrics, metricQuality)
		exportedMetrics = append(exportedMetrics, metricMaintenance)
		exportedMetrics = append(exportedMetrics, metricVulnerability)
		exportedMetrics = append(exportedMetrics, metricLicense)
		exportedMetrics = append(exportedMetrics, metricMiscellaneous)
	}

}

func periodicLogic() {
	if initializing {
		updateMetrics()
		logrus.Info("Finished initialization")
		initializing = false
		return
	} else {
		logrus.Info("Getting metrics for socket.dev in a loop")
		for {
			logrus.Info(fmt.Sprintf("Sleeping %f hours", period.Hours()))
			time.Sleep(period)
			updateMetrics()
		}
	}
}

func main() {
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		lvl = "info"
	}
	level, err := logrus.ParseLevel(lvl)
	if err != nil {
		level = logrus.DebugLevel
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	token, ok = os.LookupEnv("API_TOKEN")
	if !ok {
		logrus.Error("Could not read env. var. API_TOKEN with Socket.dev API token")
		os.Exit(1)
	}

	periodEnvVar, ok = os.LookupEnv("PERIOD")
	if !ok {
		logrus.Error("Could not read env. var. PERIOD. Setting it to 24 hours.")
		period, err = time.ParseDuration("24h")
	} else {
		period, err = time.ParseDuration(fmt.Sprintf("%sh", periodEnvVar))
		if err != nil {
			logrus.Error(fmt.Sprintf("Could not parse PERIOD env. var. to time.Duration: %s", err))
			os.Exit(1)
		}
	}

	socketCollector := newSocketCollector()
	prometheus.MustRegister(socketCollector)
	http.Handle("/metrics", promhttp.Handler())

	logrus.Info("Initializing, getting metrics for socket.dev")
	periodicLogic()
	logrus.Info("Start go rutine to get metrics for socket.dev")
	go periodicLogic()

	logrus.Info(fmt.Sprintf("Listening on port %d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
