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

func NewScoreSyncFromEnv(enabled bool) *ScoreSync {
	baseURL := strings.TrimSpace(os.Getenv("TETRUI_SCORE_API_URL"))
	apiKey := strings.TrimSpace(os.Getenv("TETRUI_SCORE_API_KEY"))
	if baseURL == "" {
		DebugLogf("score sync disabled: missing TETRUI_SCORE_API_URL")
		return nil
	}
	DebugLogf("score sync enabled=%v url=%s", enabled, baseURL)
	return &ScoreSync{
		enabled: enabled,
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

func (s *ScoreSync) SetEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.enabled = enabled
}

func (s *ScoreSync) FetchScoresCmd() tea.Cmd {
	return func() tea.Msg {
		if s == nil || !s.enabled {
			return scoresLoadedMsg{}
		}
		DebugLogf("scores fetch start url=%s", s.baseURL)
		req, err := http.NewRequest(http.MethodGet, s.baseURL, nil)
		if err != nil {
			return scoresLoadedMsg{err: err}
		}
		if s.apiKey != "" {
			req.Header.Set("X-Api-Key", s.apiKey)
		}
		resp, err := s.client.Do(req)
		if err != nil {
			DebugLogf("scores fetch request error: %v", err)
			return scoresLoadedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			DebugLogf("scores fetch status=%d", resp.StatusCode)
			return scoresLoadedMsg{err: errUnexpectedStatus(resp.StatusCode)}
		}
		var payload []apiScore
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			DebugLogf("scores fetch decode error: %v", err)
			return scoresLoadedMsg{err: err}
		}
		DebugLogf("scores fetch ok count=%d", len(payload))
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
		DebugLogf("score upload start name=%s score=%d", entry.Name, entry.Score)
		DebugLogf("score upload url=%s", s.baseURL)
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
			DebugLogf("score upload request error: %v", err)
			return scoreUploadedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			DebugLogf("score upload status=%d", resp.StatusCode)
			return scoreUploadedMsg{err: errUnexpectedStatus(resp.StatusCode)}
		}
		DebugLogf("score upload ok")
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
