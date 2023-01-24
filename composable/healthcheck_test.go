package composable

import (
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestExecHealthcheckMethod_Passing(t *testing.T) {
	exec := ParseCmdString("ls /", ".")
	hc := NewExecHealthcheck(exec, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckPassing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestExecHealthcheckMethod_Failing(t *testing.T) {
	exec := ParseCmdString("rm /", ".")
	hc := NewExecHealthcheck(exec, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Passing(t *testing.T) {
	go createTestHttpServer(200)
	hg := DescribeHttpGet(Https, 8653, "/health")
	hc := NewHttpHealthcheck(hg, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_3XX(t *testing.T) {
	go createTestHttpServer(300)
	hg := DescribeHttpGet(Https, 8653, "/health")
	hc := NewHttpHealthcheck(hg, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_4XX(t *testing.T) {
	go createTestHttpServer(400)
	hg := DescribeHttpGet(Https, 8653, "/health")
	hc := NewHttpHealthcheck(hg, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_5XX(t *testing.T) {
	go createTestHttpServer(500)
	hg := DescribeHttpGet(Https, 8653, "/health")
	hc := NewHttpHealthcheck(hg, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_ConnectionRefused(t *testing.T) {
	hg := DescribeHttpGet(Https, 8653, "/health")
	hc := NewHttpHealthcheck(hg, 0, 10*time.Millisecond)
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func createTestHttpServer(responseStatus int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseStatus)
	})
	server := http.Server{
		Addr:    ":8653",
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Println("[ERROR] ")
	}
}
