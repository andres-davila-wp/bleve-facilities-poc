package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchHandlerTerm(t *testing.T) {
	idx := mustIndex(t, sampleFacilities())
	defer idx.Close()

	req := httptest.NewRequest(http.MethodPost, "/v2/facilities/search?term=Greene&size=10", nil)
	rec := httptest.NewRecorder()
	searchHandler(idx).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Hits) != 1 || body.Hits[0].ID != "RGRN" {
		t.Fatalf("hits %+v", body.Hits)
	}
}

func TestSearchHandlerNear(t *testing.T) {
	idx := mustIndex(t, sampleFacilities())
	defer idx.Close()

	req := httptest.NewRequest(http.MethodPost, "/v2/facilities/search?lat=40.726&long=-73.996&distanceFilter=25&distanceMeasure=mi&size=10", nil)
	rec := httptest.NewRecorder()
	searchHandler(idx).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Hits) != 2 || body.Hits[0].ID != "RGRN" || body.Hits[1].ID != "RBDY" {
		t.Fatalf("hits %+v", body.Hits)
	}
}
