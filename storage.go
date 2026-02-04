package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

type Config struct {
	Theme string `json:"theme"`
	Sound bool   `json:"sound"`
	Scale int    `json:"scale"`
}

type ScoreEntry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Lines int    `json:"lines"`
	Level int    `json:"level"`
	When  string `json:"when"`
}

func loadConfig() (Config, error) {
	config := Config{
		Theme: themes[0].Name,
		Sound: true,
		Scale: 1,
	}
	path, err := configPath()
	if err != nil {
		return config, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config, nil
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	if config.Theme == "" {
		config.Theme = themes[0].Name
	}
	if config.Scale < 1 {
		config.Scale = 1
	}
	return config, nil
}

func saveConfig(config Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadScores() ([]ScoreEntry, error) {
	path, err := scoresPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []ScoreEntry{}, nil
	}
	var scores []ScoreEntry
	if err := json.Unmarshal(data, &scores); err != nil {
		return []ScoreEntry{}, err
	}
	return scores, nil
}

func saveScores(scores []ScoreEntry) error {
	path, err := scoresPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func insertScore(scores []ScoreEntry, entry ScoreEntry) []ScoreEntry {
	scores = append(scores, entry)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].When > scores[j].When
		}
		return scores[i].Score > scores[j].Score
	})
	if len(scores) > 10 {
		return scores[:10]
	}
	return scores
}

func mergeScores(local []ScoreEntry, remote []ScoreEntry) []ScoreEntry {
	merged := make([]ScoreEntry, 0, len(local)+len(remote))
	seen := make(map[string]struct{})
	for _, entry := range append(local, remote...) {
		key := entry.Name + "|" + entry.When + "|" + strconv.Itoa(entry.Score)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, entry)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Score == merged[j].Score {
			return merged[i].When > merged[j].When
		}
		return merged[i].Score > merged[j].Score
	})
	if len(merged) > 10 {
		return merged[:10]
	}
	return merged
}

func configPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "tetrui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func scoresPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "tetrui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "scores.json"), nil
}
