package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
)

var (
	errBadCoord    = errors.New("lat and long must be numbers")
	errBadDistance = errors.New("distanceFilter must be a number")
	errBadSize     = errors.New("size must be a positive integer")
)

type searchHit struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Miles float64 `json:"miles,omitempty"`
}

type searchResponse struct {
	TookMs float64     `json:"tookMs"`
	Hits   []searchHit `json:"hits"`
}

func searchHandler(idx *Index) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/facilities/search" {
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		size, err := querySize(query.Get("size"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		term := query.Get("term")
		latRaw, longRaw := query.Get("lat"), query.Get("long")
		started := time.Now()
		var hits []Hit
		switch {
		case latRaw != "" || longRaw != "":
			hits, err = searchNearQuery(idx, query.Get("distanceFilter"), query.Get("distanceMeasure"), latRaw, longRaw, size)
		case term != "":
			hits, err = idx.SearchText(term, size)
		default:
			http.Error(w, "term or lat and long are required", http.StatusBadRequest)
			return
		}
		elapsed := time.Since(started)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		out := searchResponse{TookMs: float64(elapsed.Microseconds()) / 1000, Hits: []searchHit{}}
		for _, hit := range hits {
			item := searchHit{ID: hit.ID, Name: hit.Name}
			if latRaw != "" {
				item.Miles = hit.Miles
			}
			out.Hits = append(out.Hits, item)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}

func searchNearQuery(idx *Index, distanceRaw, unit, latRaw, longRaw string, size int) ([]Hit, error) {
	lat, err := strconv.ParseFloat(latRaw, 64)
	if err != nil {
		return nil, errBadCoord
	}
	lon, err := strconv.ParseFloat(longRaw, 64)
	if err != nil {
		return nil, errBadCoord
	}
	distance := 25.0
	if distanceRaw != "" {
		distance, err = strconv.ParseFloat(distanceRaw, 64)
		if err != nil {
			return nil, errBadDistance
		}
	}
	if unit == "" {
		unit = "mi"
	}
	return idx.SearchNear(lat, lon, distance, unit, size)
}

func querySize(raw string) (int, error) {
	if raw == "" {
		return 10, nil
	}
	size, err := strconv.Atoi(raw)
	if err != nil || size < 1 {
		return 0, errBadSize
	}
	return size, nil
}
