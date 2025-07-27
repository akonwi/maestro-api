package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type PredictionResponse struct {
	Response []PredictionData `json:"response"`
}

type PredictionData struct {
	Predictions Predictions `json:"predictions"`
}

type Predictions struct {
	Advice string `json:"advice"`
}

type PredGoals struct {
	Home string `json:"home"`
	Away string `json:"away"`
}

type MatchPrediction struct {
	Advice string
}

func fetchPrediction(fixtureID int) (*MatchPrediction, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("https://v3.football.api-sports.io/predictions?fixture=%d", fixtureID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header - you'll need to set this environment variable
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable not set")
	}

	req.Header.Set("X-RapidAPI-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var predResponse PredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&predResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(predResponse.Response) == 0 {
		return nil, fmt.Errorf("no predictions found for fixture %d", fixtureID)
	}

	pred := predResponse.Response[0].Predictions
	return &MatchPrediction{
		Advice: pred.Advice,
	}, nil
}
