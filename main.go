package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	censusAPIBase = "https://api.census.gov/data/2021/acs/acs5"
)

type Location struct {
	GeoID      string
	Name       string
	Type       string
	Population string
}

func main() {
	fmt.Println("Fetching US Census population data...")

	locations := []Location{}

	// Fetch state data
	states, err := fetchStates()
	if err != nil {
		fmt.Printf("Error fetching states: %v\n", err)
		os.Exit(1)
	}
	locations = append(locations, states...)
	fmt.Printf("Fetched %d states\n", len(states))

	// Fetch county data for each state
	for _, state := range states {
		counties, err := fetchCounties(state.GeoID)
		if err != nil {
			fmt.Printf("Warning: Error fetching counties for %s: %v\n", state.Name, err)
			continue
		}
		locations = append(locations, counties...)
	}
	fmt.Printf("Total locations: %d\n", len(locations))

	// Write to CSV
	err = writeCSV("census_population.csv", locations)
	if err != nil {
		fmt.Printf("Error writing CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully wrote data to census_population.csv")
}

func fetchStates() ([]Location, error) {
	url := fmt.Sprintf("%s?get=NAME,B01003_001E&for=state:*", censusAPIBase)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data [][]string
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Skip header row
	locations := make([]Location, 0, len(data)-1)
	for i := 1; i < len(data); i++ {
		if len(data[i]) < 3 {
			continue
		}
		locations = append(locations, Location{
			GeoID:      data[i][2], // state code
			Name:       data[i][0],
			Type:       "state",
			Population: data[i][1],
		})
	}

	return locations, nil
}

func fetchCounties(stateCode string) ([]Location, error) {
	url := fmt.Sprintf("%s?get=NAME,B01003_001E&for=county:*&in=state:%s", 
		censusAPIBase, stateCode)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data [][]string
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Skip header row
	locations := make([]Location, 0, len(data)-1)
	for i := 1; i < len(data); i++ {
		if len(data[i]) < 4 {
			continue
		}
		// Construct full GEOID: state + county
		geoID := data[i][2] + data[i][3]
		locations = append(locations, Location{
			GeoID:      geoID,
			Name:       strings.TrimSpace(data[i][0]),
			Type:       "county",
			Population: data[i][1],
		})
	}

	return locations, nil
}

func writeCSV(filename string, locations []Location) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"geoid", "name", "type", "population"}); err != nil {
		return err
	}

	// Write data
	for _, loc := range locations {
		if err := writer.Write([]string{loc.GeoID, loc.Name, loc.Type, loc.Population}); err != nil {
			return err
		}
	}

	return nil
}
