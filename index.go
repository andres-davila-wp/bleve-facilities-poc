package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
)

type Facility struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	StreetAddress string  `json:"streetAddress"`
	PostalCode    string  `json:"postalCode"`
	Locality      string  `json:"locality"`
	StoreNumber   string  `json:"storeNumber"`
	CountryCode   string  `json:"countryCode"`
	Type          string  `json:"type"`
	IsActive      bool    `json:"isActive"`
	HasLaunched   bool    `json:"hasLaunched"`
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
}

type Hit struct {
	ID    string
	Name  string
	Miles float64
}

type Index struct {
	bleve bleve.Index
	byID  map[string]Facility
}

func Load(path string) ([]Facility, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("no facilities in %s", path)
	}
	if trimmed[0] == '[' {
		var facilities []Facility
		if err := json.Unmarshal(trimmed, &facilities); err != nil {
			return nil, err
		}
		if len(facilities) == 0 {
			return nil, fmt.Errorf("no facilities in %s", path)
		}
		return facilities, nil
	}
	var response struct {
		Hits struct {
			Hits []struct {
				Source pointFacility `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(trimmed, &response); err != nil {
		return nil, err
	}
	facilities := make([]Facility, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		facility, err := hit.Source.toFacility()
		if err != nil {
			return nil, err
		}
		facilities = append(facilities, facility)
	}
	if len(facilities) == 0 {
		return nil, fmt.Errorf("no facilities in %s", path)
	}
	return facilities, nil
}

type pointFacility struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StreetAddress string `json:"streetAddress"`
	PostalCode    string `json:"postalCode"`
	Locality      string `json:"locality"`
	StoreNumber   string `json:"storeNumber"`
	CountryCode   string `json:"countryCode"`
	Type          string `json:"type"`
	IsActive      bool   `json:"isActive"`
	HasLaunched   bool   `json:"hasLaunched"`
	Location      string `json:"location"`
}

func (source pointFacility) toFacility() (Facility, error) {
	lat, lon, err := parsePoint(source.Location)
	if err != nil {
		return Facility{}, fmt.Errorf("facility %s: %w", source.ID, err)
	}
	return Facility{
		ID:            source.ID,
		Name:          source.Name,
		StreetAddress: source.StreetAddress,
		PostalCode:    source.PostalCode,
		Locality:      source.Locality,
		StoreNumber:   source.StoreNumber,
		CountryCode:   source.CountryCode,
		Type:          source.Type,
		IsActive:      source.IsActive,
		HasLaunched:   source.HasLaunched,
		Lat:           lat,
		Lon:           lon,
	}, nil
}

func parsePoint(value string) (lat, lon float64, err error) {
	n, scanErr := fmt.Sscanf(value, "POINT (%f %f)", &lon, &lat)
	if scanErr != nil || n != 2 {
		return 0, 0, fmt.Errorf("location %q is not POINT (lon lat)", value)
	}
	return lat, lon, nil
}

func Open(facilities []Facility) (*Index, error) {
	if len(facilities) == 0 {
		return nil, fmt.Errorf("no facilities to index")
	}
	idx, err := bleve.NewMemOnly(newMapping())
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Facility, len(facilities))
	for _, facility := range facilities {
		if facility.ID == "" {
			_ = idx.Close()
			return nil, fmt.Errorf("facility is missing id")
		}
		if err := idx.Index(facility.ID, indexedDoc(facility)); err != nil {
			_ = idx.Close()
			return nil, err
		}
		byID[facility.ID] = facility
	}
	return &Index{bleve: idx, byID: byID}, nil
}

func (idx *Index) Close() error {
	return idx.bleve.Close()
}

func (idx *Index) SearchText(term string, size int) ([]Hit, error) {
	if term == "" {
		return nil, fmt.Errorf("term is empty")
	}
	return idx.search(withActive(textQuery(term)), size, nil, 0, 0)
}

func (idx *Index) SearchNear(lat, lon, distance float64, unit string, size int) ([]Hit, error) {
	if distance <= 0 {
		return nil, fmt.Errorf("distance must be greater than 0")
	}
	if unit != "mi" && unit != "km" {
		return nil, fmt.Errorf("unit must be mi or km")
	}
	geoQuery := bleve.NewGeoDistanceQuery(lon, lat, fmt.Sprintf("%g%s", distance, unit))
	geoQuery.SetField("location")
	geoSort, err := search.NewSortGeoDistance("location", unit, lon, lat, false)
	if err != nil {
		return nil, err
	}
	return idx.search(withActive(geoQuery), size, search.SortOrder{geoSort}, lat, lon)
}

func (idx *Index) search(q query.Query, size int, sort search.SortOrder, lat, lon float64) ([]Hit, error) {
	req := bleve.NewSearchRequest(q)
	req.Size = normalizeSize(size)
	if len(sort) > 0 {
		req.SortByCustom(sort)
	}
	result, err := idx.bleve.Search(req)
	if err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		facility := idx.byID[hit.ID]
		item := Hit{ID: hit.ID, Name: facility.Name}
		if len(sort) > 0 {
			item.Miles = milesBetween(lat, lon, facility.Lat, facility.Lon)
		}
		hits = append(hits, item)
	}
	return hits, nil
}

type latLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type indexed struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StreetAddress string `json:"streetAddress"`
	PostalCode    string `json:"postalCode"`
	Locality      string `json:"locality"`
	StoreNumber   string `json:"storeNumber"`
	CountryCode   string `json:"countryCode"`
	Type          string `json:"type"`
	IsActive      bool   `json:"isActive"`
	HasLaunched   bool   `json:"hasLaunched"`
	Location      latLon `json:"location"`
}

func indexedDoc(facility Facility) indexed {
	return indexed{
		ID:            facility.ID,
		Name:          facility.Name,
		StreetAddress: facility.StreetAddress,
		PostalCode:    facility.PostalCode,
		Locality:      facility.Locality,
		StoreNumber:   facility.StoreNumber,
		CountryCode:   facility.CountryCode,
		Type:          facility.Type,
		IsActive:      facility.IsActive,
		HasLaunched:   facility.HasLaunched,
		Location:      latLon{Lat: facility.Lat, Lon: facility.Lon},
	}
}

func newMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()
	doc := bleve.NewDocumentMapping()
	text := bleve.NewTextFieldMapping()
	for _, field := range []string{"id", "name", "streetAddress", "postalCode", "locality", "storeNumber", "countryCode", "type"} {
		doc.AddFieldMappingsAt(field, text)
	}
	flag := bleve.NewBooleanFieldMapping()
	doc.AddFieldMappingsAt("isActive", flag)
	doc.AddFieldMappingsAt("hasLaunched", flag)
	doc.AddFieldMappingsAt("location", bleve.NewGeoPointFieldMapping())
	indexMapping.DefaultMapping = doc
	return indexMapping
}

func textQuery(term string) query.Query {
	fields := []string{"id", "name", "streetAddress", "postalCode", "locality", "storeNumber", "countryCode"}
	parts := make([]query.Query, 0, len(fields))
	for _, field := range fields {
		match := bleve.NewMatchQuery(term)
		match.SetField(field)
		parts = append(parts, match)
	}
	return bleve.NewDisjunctionQuery(parts...)
}

func withActive(q query.Query) query.Query {
	active := bleve.NewBoolFieldQuery(true)
	active.SetField("isActive")
	launched := bleve.NewBoolFieldQuery(true)
	launched.SetField("hasLaunched")
	combined := bleve.NewBooleanQuery()
	combined.AddMust(active, launched, q)
	return combined
}

func normalizeSize(size int) int {
	if size < 1 {
		return 10
	}
	return size
}

func milesBetween(lat1, lon1, lat2, lon2 float64) float64 {
	const earthMiles = 3958.8
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)
	lat1 = degreesToRadians(lat1)
	lat2 = degreesToRadians(lat2)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthMiles * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
