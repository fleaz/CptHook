package main

import (
	"net/http/httptest"
	"net/http"
	"testing"
)


func TestPrometheusHandler(t *testing.T){
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(prometheusHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
				status, http.StatusOK)
	}
}