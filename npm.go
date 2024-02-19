package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
)

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

type NpmDownloadCounts struct {
	Downloads int    `json:"downloads"`
	Day       string `json:"day"`
}

type NpmDownloadCountResponse struct {
	Start     string              `json:"start"`
	End       string              `json:"end"`
	Package   string              `json:"package"`
	Downloads []NpmDownloadCounts `json:"downloads"`
}

func GetDownloadCountForCeloNpmPackage(currentPackage NpmPackage, client *http.Client) (NpmDownloadCountResponse, error) {
	var url = fmt.Sprintf("https://api.npmjs.org/downloads/range/last-day/@celo/%s", currentPackage.Name)
	res, err := client.Get(url)

	if err != nil {
		logrus.Errorf("Error making http request to registry.npmjs.org: %s", err)
		return nil, err
	}

	var result NpmDownloadCountResponse

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		logrus.Errorf("Could not decode download count response body from npmjs.org: %s", err)
		return result, err
	}

	logrus.Debugf("npm download count response: %v", result)

	return result, nil
}

func GetCeloNPMPackages(client *http.Client) (NpmResponse, error) {
	var npmResponse NpmResponse
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

	return npmResponse, nil
}
