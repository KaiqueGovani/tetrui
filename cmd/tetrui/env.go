package main

import "os"

var (
	defaultScoreAPIURL string
	defaultScoreAPIKey string
)

func loadEmbeddedEnv() {
	if defaultScoreAPIURL != "" {
		if _, exists := os.LookupEnv("TETRUI_SCORE_API_URL"); !exists {
			_ = os.Setenv("TETRUI_SCORE_API_URL", defaultScoreAPIURL)
		}
	}
	if defaultScoreAPIKey != "" {
		if _, exists := os.LookupEnv("TETRUI_SCORE_API_KEY"); !exists {
			_ = os.Setenv("TETRUI_SCORE_API_KEY", defaultScoreAPIKey)
		}
	}
}
