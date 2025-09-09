package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/brensch/passengerprincess/pkg/maps"
)

type CircleResult struct {
	Circle      maps.Circle `json:"circle"`
	ErrorsCount int         `json:"errors_count"`
	PlaceIDs    []string    `json:"place_ids"`
}

func main() {
	// Fixed bounds for a 5km x 5km area around Mountain View
	latMin := 37.2
	latMax := 37.9
	lonMin := -122.6
	lonMax := -121.8

	radius := 1000

	targets, err := maps.CreateMesh(latMin, latMax, lonMin, lonMax, radius)
	if err != nil {
		panic(err)
	}

	if len(targets) == 0 {
		panic("CreateMesh returned no targets")
	}

	// Generate HTML using VisualiseMeshHTML
	html := maps.VisualiseMeshHTML(latMax, lonMin, targets)

	// Write HTML to file
	err = os.WriteFile("mesh.html", []byte(html), 0644)
	if err != nil {
		panic(err)
	}

	// get all the tesla superchargers in a mountain view
	// (just their ids)
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		panic("MAPS_API_KEY environment variable not set")
	}

	query := "tesla supercharger"

	fmt.Printf("running %d searches for superchargers\n", len(targets))

	// run all the searches concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []CircleResult

	for _, target := range targets {
		wg.Add(1)
		go func(target maps.Circle) {
			defer wg.Done()
			var placeIDs []string
			errorsCount := 0
			for {
				ids, err := maps.GetPlaceIDsViaTextSearch(apiKey, query, target)
				if err != nil {
					slog.Error("GetPlaceIDsViaTextSearch failed", "error", err, "circle", target)
					errorsCount++
					continue
				}
				placeIDs = ids
				break
			}
			mu.Lock()
			results = append(results, CircleResult{
				Circle:      target,
				ErrorsCount: errorsCount,
				PlaceIDs:    placeIDs,
			})
			mu.Unlock()
		}(target)
	}
	wg.Wait()

	// Print results
	for _, res := range results {
		fmt.Printf("Circle: %+v\n", res.Circle)
		fmt.Printf("Errors encountered: %d\n", res.ErrorsCount)
		fmt.Printf("IDs within that circle: %v\n\n", res.PlaceIDs)
	}

	// Write JSON to file
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("scraper_results.json", jsonData, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("Results written to scraper_results.json")

}
