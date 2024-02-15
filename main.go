package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var initializing bool = true

var retries int
var timeout time.Duration

var exportedMetrics = []Metric{}

var token string
var period time.Duration
var port = 9101

type Metric map[string]interface{}

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

func (s *SocketResponse) ToMetrics(packageName string, packageVersion string) []Metric {
	metrics := []Metric{
		Metric{"score": "supplychainrisk", "value": s.Supplychainrisk.Score},
		Metric{"score": "quality", "value": s.Quality.Score},
		Metric{"score": "maintenance", "value": s.Maintenance.Score},
		Metric{"score": "vulnerability", "value": s.Vulnerability.Score},
		Metric{"score": "license", "value": s.License.Score},
		Metric{"score": "miscellaneous", "value": s.Miscellaneous.Score},
	}

	for _, metric := range metrics {
		metric["name"] = packageName
		metric["version"] = packageVersion
	}

	return metrics
}

func fetchMetrics() ([]Metric, error) {
	var npmResponse NpmResponse
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = retries
	retryClient.Logger = log.New(ioutil.Discard, "", log.LstdFlags)

	var result = []Metric{}
	retryClient.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, attempt int) {
		logrus.WithFields(logrus.Fields{
			"host":    req.URL.Host,
			"path":    req.URL.Path,
			"attempt": attempt,
		}).Info("Sending request")
	}

	client := retryClient.StandardClient() // *http.Client
	client.Timeout = timeout

	logrus.Info("Sending request to registry.npmjs.org")
	res, err := client.Get("https://registry.npmjs.org/-/v1/search?text=scope:celo&size=100")
	if err != nil {
		logrus.Errorf("Error making http request to registry.npmjs.org: %s", err)
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&npmResponse)
	if err != nil {
		logrus.Errorf("Could not decode response body from registry.npmjs.org: %s", err)
		return nil, err
	}

	celoPackages := npmResponse.Objects
	var socketAPI = NewSocketAPI(token)

	for _, object := range celoPackages {
		var currentPackage = object.Package

		var socketResponse, err = socketAPI.FetchSocketScores(currentPackage, client)
		if err != nil {
			logrus.WithFields(logrus.Fields{"package": currentPackage.Name}).Error("Failed to fetch score for package")
			continue
		}
		socketScoreMetrics := socketResponse.ToMetrics(currentPackage.Name, currentPackage.Version)

		//todo: implement npm download fetch

		result = append(result, socketScoreMetrics...)
	}

	return result, nil
}

func periodicLogic() {
	logrus.Info("Getting metrics for socket.dev in a loop")
	for {
		metrics, err := fetchMetrics()
		if err != nil {
			logrus.Errorf("Error upon fetching metrics %e", err)
			continue
		}
		exportedMetrics = append(exportedMetrics, metrics...)

		logrus.Infof("Sleeping %f hours", period.Hours())
		time.Sleep(period)
	}
}

func initializeConfig() {
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

	periodEnvVar, ok := os.LookupEnv("PERIOD")
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

	retriesEnvVar, ok := os.LookupEnv("RETRIES")
	if !ok {
		logrus.Error("Could not read env. var. RETRIES. Setting it to 5.")
		retries = 5
	} else {
		retries, err = strconv.Atoi(retriesEnvVar)
		if err != nil {
			logrus.Error(fmt.Sprintf("Could not parse RETRIES env. var. to int: %s", err))
			os.Exit(1)
		}
	}

	timeoutEnvVar, ok := os.LookupEnv("TIMEOUT")
	if !ok {
		logrus.Error("Could not read env. var. TIMEOUT. Setting it to 15 seconds.")
		timeout = 15 * time.Second
	} else {
		timeoutInt, err := strconv.Atoi(timeoutEnvVar)
		if err != nil {
			logrus.Error(fmt.Sprintf("Could not parse TIMEOUT env. var. to int: %s", err))
			os.Exit(1)
		}
		timeout = time.Duration(timeoutInt) * time.Second
	}
}

func main() {
	initializeConfig()

	socketCollector := newSocketCollector()
	prometheus.MustRegister(socketCollector)
	http.Handle("/metrics", promhttp.Handler())

	logrus.Info("Initializing, getting metrics for socket.dev")
	metrics, err := fetchMetrics()
	if err != nil {
		logrus.Fatalf("Error upon initializing metrics %e", err)
	}
	exportedMetrics = append(exportedMetrics, metrics...)

	logrus.Info("Start go routine to get metrics for socket.dev")
	go periodicLogic()

	logrus.Infof("Listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
