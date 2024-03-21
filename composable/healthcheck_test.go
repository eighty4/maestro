package composable

import (
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestExecHealthcheckMethod_Passing(t *testing.T) {
	hc := NewExecHealthcheck(ParseCmdString("ls /", "."), 0, 10*time.Millisecond)
	defer hc.Stop()
	go hc.Start()
	assert.Equal(t, HealthcheckPassing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestExecHealthcheckMethod_Failing(t *testing.T) {
	hc := NewExecHealthcheck(ParseCmdString("rm /", "."), 0, 10*time.Millisecond)
	defer hc.Stop()
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Passing(t *testing.T) {
	go createTestHttpServer(200)
	hc := NewHttpHealthcheck(DescribeHttpGet(Https, 8653, "/health"), 0, 10*time.Millisecond)
	defer hc.Stop()
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_3XX(t *testing.T) {
	go createTestHttpServer(300)
	hc := NewHttpHealthcheck(DescribeHttpGet(Https, 8653, "/health"), 0, 10*time.Millisecond)
	defer hc.Stop()
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_4XX(t *testing.T) {
	go createTestHttpServer(400)
	hc := NewHttpHealthcheck(DescribeHttpGet(Https, 8653, "/health"), 0, 10*time.Millisecond)
	defer hc.Stop()
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_5XX(t *testing.T) {
	go createTestHttpServer(500)
	hc := NewHttpHealthcheck(DescribeHttpGet(Https, 8653, "/health"), 0, 10*time.Millisecond)
	defer hc.Stop()
	go hc.Start()
	assert.Equal(t, HealthcheckFailing, <-hc.StatusC)
	hc.Stop()
	assert.Equal(t, HealthcheckStopped, hc.Status)
}

func TestHttpHealthcheckMethod_Failing_ConnectionRefused(t *testing.T) {
	hc := NewHttpHealthcheck(DescribeHttpGet(Https, 8653, "/health"), 0, 10*time.Millisecond)
	defer hc.Stop()
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
