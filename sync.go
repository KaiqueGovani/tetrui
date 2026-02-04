package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ScoreSync struct {
	enabled bool
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewScoreSyncFromEnv() *ScoreSync {
	baseURL := strings.TrimSpace(os.Getenv("TETRUI_SCORE_API_URL"))
	apiKey := strings.TrimSpace(os.Getenv("TETRUI_SCORE_API_KEY"))
	enabled := strings.EqualFold(strings.TrimSpace(os.Getenv("TETRUI_SCORE_SYNC")), "true")
	if baseURL == "" || !enabled {
		return nil
	}
	return &ScoreSync{
		enabled: true,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

func (s *ScoreSync) Enabled() bool {
	return s != nil && s.enabled
}

func (s *ScoreSync) FetchScoresCmd() tea.Cmd {
	return func() tea.Msg {
		if s == nil || !s.enabled {
			return scoresLoadedMsg{}
		}
		req, err := http.NewRequest(http.MethodGet, s.baseURL, nil)
		if err != nil {
			return scoresLoadedMsg{err: err}
		}
		if s.apiKey != "" {
			req.Header.Set("X-Api-Key", s.apiKey)
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return scoresLoadedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return scoresLoadedMsg{err: errUnexpectedStatus(resp.StatusCode)}
		}
		var payload []apiScore
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return scoresLoadedMsg{err: err}
		}
		scores := make([]ScoreEntry, 0, len(payload))
		for _, entry := range payload {
			scores = append(scores, entry.ToScoreEntry())
		}
		return scoresLoadedMsg{scores: scores}
	}
}

func (s *ScoreSync) UploadScoreCmd(entry ScoreEntry) tea.Cmd {
	return func() tea.Msg {
		if s == nil || !s.enabled {
			return scoreUploadedMsg{}
		}
		payload, err := json.Marshal(uploadScore{
			Name:  entry.Name,
			Score: entry.Score,
			Lines: entry.Lines,
			Level: entry.Level,
		})
		if err != nil {
			return scoreUploadedMsg{err: err}
		}
		req, err := http.NewRequest(http.MethodPost, s.baseURL, bytes.NewReader(payload))
		if err != nil {
			return scoreUploadedMsg{err: err}
		}
		req.Header.Set("Content-Type", "application/json")
		if s.apiKey != "" {
			req.Header.Set("X-Api-Key", s.apiKey)
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return scoreUploadedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return scoreUploadedMsg{err: errUnexpectedStatus(resp.StatusCode)}
		}
		return scoreUploadedMsg{}
	}
}

type statusError int

func (s statusError) Error() string {
	return "unexpected status: " + http.StatusText(int(s))
}

func errUnexpectedStatus(code int) error {
	return statusError(code)
}

type apiScore struct {
	Name      string `json:"name"`
	Score     int    `json:"score"`
	Lines     int    `json:"lines"`
	Level     int    `json:"level"`
	CreatedAt string `json:"createdAt"`
}

type uploadScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Lines int    `json:"lines"`
	Level int    `json:"level"`
}

func (s apiScore) ToScoreEntry() ScoreEntry {
	return ScoreEntry{
		Name:  s.Name,
		Score: s.Score,
		Lines: s.Lines,
		Level: s.Level,
		When:  formatAPITime(s.CreatedAt),
	}
}
