package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
)

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

type SocketAPI struct {
	token string
}

func NewSocketAPI(token string) *SocketAPI {
	api := new(SocketAPI)
	api.token = token
	return api
}

func (s SocketAPI) buildRequest(url string) *http.Request {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.token)))

	return req
}

func (s SocketAPI) FetchSocketScores(pack NpmPackage, client *http.Client) (SocketResponse, error) {
	var url = fmt.Sprintf("https://api.socket.dev/v0/npm/%s/%s/score", pack.Name, pack.Version)
	var req = s.buildRequest(url)

	logrus.Infof("Requesting package %s/%s scores to api.socket.dev", pack.Name, pack.Version)

	var result SocketResponse
	res, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Error requesting package %s to api.socket.dev: %s", pack.Name, err)
		return result, err
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		logrus.Errorf("Could not decode response body from api.socket.dev: %s", err)
		return result, err
	}

	logrus.Debugf("socket.dev package score response: %v", result)

	return result, nil
}
