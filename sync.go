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
		req, err := http.NewRequest(http.MethodGet, s.baseURL+"/scores?limit=10", nil)
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
		var scores []ScoreEntry
		if err := json.NewDecoder(resp.Body).Decode(&scores); err != nil {
			return scoresLoadedMsg{err: err}
		}
		return scoresLoadedMsg{scores: scores}
	}
}

func (s *ScoreSync) UploadScoreCmd(entry ScoreEntry) tea.Cmd {
	return func() tea.Msg {
		if s == nil || !s.enabled {
			return scoreUploadedMsg{}
		}
		payload, err := json.Marshal(entry)
		if err != nil {
			return scoreUploadedMsg{err: err}
		}
		req, err := http.NewRequest(http.MethodPost, s.baseURL+"/scores", bytes.NewReader(payload))
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
