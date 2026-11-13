package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	data := flag.String("data", "testdata/facilities.json", "JSON array of facilities")
	term := flag.String("term", "Greene", "text query")
	lat := flag.Float64("lat", 40.7260, "latitude for the distance search")
	lon := flag.Float64("lon", -73.9960, "longitude for the distance search")
	distance := flag.Float64("distance", 25, "distance filter")
	unit := flag.String("unit", "mi", "distance unit: mi or km")
	size := flag.Int("size", 10, "maximum hits")
	repeat := flag.Int("repeat", 50, "searches to time, after the index is built")
	serve := flag.Bool("serve", false, "listen for the same POST /v2/facilities/search calls")
	addr := flag.String("addr", "127.0.0.1:8787", "address used with -serve")
	flag.Parse()

	facilities, err := Load(*data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	started := time.Now()
	idx, err := Open(facilities)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer idx.Close()
	fmt.Printf("indexed %d facilities in %s\n", len(facilities), time.Since(started))

	if *serve {
		fmt.Printf("listening on http://%s/v2/facilities/search\n", *addr)
		if err := http.ListenAndServe(*addr, searchHandler(idx)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	textHits, textMin, textAvg, err := timeSearch(*repeat, func() ([]Hit, error) {
		return idx.SearchText(*term, *size)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("term %q hits=%d min=%s avg=%s\n", *term, len(textHits), textMin, textAvg)
	printHits(textHits, false)

	nearHits, nearMin, nearAvg, err := timeSearch(*repeat, func() ([]Hit, error) {
		return idx.SearchNear(*lat, *lon, *distance, *unit, *size)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("near %.5f,%.5f within %g%s hits=%d min=%s avg=%s\n", *lat, *lon, *distance, *unit, len(nearHits), nearMin, nearAvg)
	printHits(nearHits, true)
}

func timeSearch(n int, search func() ([]Hit, error)) (hits []Hit, min, avg time.Duration, err error) {
	if n < 1 {
		n = 1
	}
	var total time.Duration
	for i := 0; i < n; i++ {
		started := time.Now()
		hits, err = search()
		elapsed := time.Since(started)
		if err != nil {
			return nil, 0, 0, err
		}
		total += elapsed
		if i == 0 || elapsed < min {
			min = elapsed
		}
	}
	return hits, min, total / time.Duration(n), nil
}

func printHits(hits []Hit, withMiles bool) {
	for _, hit := range hits {
		if withMiles {
			fmt.Printf("  %s  %s  %.2f mi\n", hit.ID, hit.Name, hit.Miles)
			continue
		}
		fmt.Printf("  %s  %s\n", hit.ID, hit.Name)
	}
}
