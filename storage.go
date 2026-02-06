package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type Config struct {
	Theme  string `json:"theme"`
	Sound  bool   `json:"sound"`
	Music  bool   `json:"music"`
	Shadow bool   `json:"shadow"`
	Scale  int    `json:"scale"`
	Sync   bool   `json:"sync"`
	Volume int    `json:"volume"`
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
		Theme:  themes[0].Name,
		Sound:  true,
		Music:  true,
		Shadow: true,
		Scale:  1,
		Sync:   true,
		Volume: 70,
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
	if !bytes.Contains(data, []byte("\"sync\"")) {
		config.Sync = true
	}
	if !bytes.Contains(data, []byte("\"music\"")) {
		config.Music = true
	}
	if !bytes.Contains(data, []byte("\"shadow\"")) {
		config.Shadow = true
	}
	if !bytes.Contains(data, []byte("\"volume\"")) {
		config.Volume = 70
	}
	if config.Theme == "" {
		config.Theme = themes[0].Name
	}
	if config.Scale < 1 {
		config.Scale = 1
	}
	if !config.Shadow {
		config.Shadow = true
	}
	if config.Volume < 0 {
		config.Volume = 0
	}
	if config.Volume > 100 {
		config.Volume = 100
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
	if len(scores) > 50 {
		return scores[:50]
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
	if len(merged) > 50 {
		return merged[:50]
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

func formatAPITime(value string) string {
	if value == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Local().Format("2006-01-02 15:04")
}
