package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Source https://stackoverflow.com/a/33404435
// Exit with return code 1 if env. var. is not provided
func TestApiTokenCrash(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestApiTokenCrash")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

var collector = newSocketCollector()
var ch = make(chan prometheus.Metric)

// Count that the number of metrics is greater than 1
func TestCollectAndCount(t *testing.T) {

	fetchMetrics()

	number := testutil.CollectAndCount(collector, "socket_score")
	if number < 1 {
		t.Fatalf("Less than 1 metric was returned. Only %d metrics returned", number)
	}

}

// Check linter on metrics
func TestCollectndLint(t *testing.T) {

	problem, err := testutil.CollectAndLint(collector, "socket_score")
	if err != nil {
		t.Errorf("%s", err.Error())
		t.Errorf("%s", problem)
	}

}
