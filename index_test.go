package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsElasticsearchPoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facilities.json")
	body := `{
	  "hits": {"hits": [{"_source": {
	    "id": "RGRN",
	    "name": "Greene St",
	    "isActive": true,
	    "hasLaunched": true,
	    "location": "POINT (-73.9966 40.7256)"
	  }}]}
	}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	facilities, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(facilities) != 1 || facilities[0].ID != "RGRN" {
		t.Fatalf("%+v", facilities)
	}
	if math.Abs(facilities[0].Lat-40.7256) > 0.0001 || math.Abs(facilities[0].Lon-(-73.9966)) > 0.0001 {
		t.Fatalf("coords %+v", facilities[0])
	}
}

func TestSearchTextFindsActiveGreene(t *testing.T) {
	idx := mustIndex(t, sampleFacilities())
	defer idx.Close()

	hits, err := idx.SearchText("Greene", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "RGRN" {
		t.Fatalf("hits: %+v", hits)
	}
}

func TestSearchNearOrdersByDistance(t *testing.T) {
	idx := mustIndex(t, sampleFacilities())
	defer idx.Close()

	hits, err := idx.SearchNear(40.7260, -73.9960, 25, "mi", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].ID != "RGRN" || hits[1].ID != "RBDY" {
		t.Fatalf("hits: %+v", hits)
	}
	if hits[0].Miles > hits[1].Miles {
		t.Fatalf("distance out of order: %+v", hits)
	}
}

func mustIndex(t *testing.T, facilities []Facility) *Index {
	t.Helper()
	idx, err := Open(facilities)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func sampleFacilities() []Facility {
	return []Facility{
		{ID: "RGRN", Name: "Greene St", StreetAddress: "121 Greene St", PostalCode: "10012", Locality: "New York", StoreNumber: "001", IsActive: true, HasLaunched: true, Type: "retail", Lat: 40.7256, Lon: -73.9966},
		{ID: "RBDY", Name: "Broadway", StreetAddress: "120 Broadway", PostalCode: "10006", Locality: "New York", StoreNumber: "002", IsActive: true, HasLaunched: true, Type: "retail", Lat: 40.7081, Lon: -74.0130},
		{ID: "RLAX", Name: "Los Angeles", StreetAddress: "100 Sunset Blvd", PostalCode: "90028", Locality: "Los Angeles", StoreNumber: "003", IsActive: true, HasLaunched: true, Type: "retail", Lat: 34.0522, Lon: -118.2437},
		{ID: "RLAB", Name: "Greene Lab", StreetAddress: "1 Greene St", PostalCode: "10012", Locality: "New York", StoreNumber: "004", IsActive: false, HasLaunched: true, Type: "lab", Lat: 40.72605, Lon: -73.99605},
	}
}
