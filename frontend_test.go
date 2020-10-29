package main

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestGetState(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo/state", nil)
	w := httptest.NewRecorder()
	state(w, req)

	res := w.Result()
	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != 200 {
		t.Error(res.StatusCode)
	}
	if data := string(body); data != "{}" {
		t.Error(data)
	}
}

func TestGetState_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "http://foo/state", nil)
	w := httptest.NewRecorder()
	state(w, req)

	res := w.Result()

	if res.StatusCode != 405 {
		t.Error(res.StatusCode)
	}
}
