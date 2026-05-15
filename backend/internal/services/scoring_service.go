package services

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cuckoo/backend/internal/config"
)

type ScoringService interface {
	Score(ctx context.Context, input ScoringInput) ScoringResult
}

type ScoringInput struct {
	Text        string
	Units       int
	MaxUnits    int
	TimeTakenMs int
	TimeLimitMs int
}

type ScoringResult struct {
	Compliance int    `json:"compliance"`
	Time       int    `json:"time"`
	Fluency    int    `json:"fluency"`
	Total      int    `json:"total"`
	Explain    string `json:"explain"`
}

type LocalScoringService struct {
	aiServiceURL string
	client       *http.Client
}

func NewScoringService(cfg config.Config) ScoringService {
	return &LocalScoringService{
		aiServiceURL: strings.TrimRight(cfg.AIServiceURL, "/"),
		client:       &http.Client{Timeout: 1500 * time.Millisecond},
	}
}

func (s *LocalScoringService) Score(ctx context.Context, input ScoringInput) ScoringResult {
	compliance := 50
	if input.Units > input.MaxUnits {
		compliance = 0
	}
	timeScore := 30
	if input.TimeLimitMs > 0 {
		timeScore = 30 - min(input.TimeTakenMs*30/input.TimeLimitMs, 30)
	} else {
		timeScore = 30 - min(input.TimeTakenMs/2000, 30)
	}
	fluency, explain := s.judgeFluency(ctx, input.Text)
	total := compliance + timeScore + fluency
	return ScoringResult{Compliance: compliance, Time: timeScore, Fluency: fluency, Total: total, Explain: explain}
}

func (s *LocalScoringService) judgeFluency(ctx context.Context, text string) (int, string) {
	if s.aiServiceURL == "" {
		return 20, "local placeholder fluency score"
	}
	body, _ := json.Marshal(map[string]interface{}{"text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.aiServiceURL+"/judge", bytes.NewReader(body))
	if err != nil {
		return 20, "local fallback after judge request build failed"
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return 20, "local fallback after judge request failed"
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return 20, "local fallback after judge returned non-2xx"
	}
	var payload struct {
		Fluency int    `json:"fluency"`
		Explain string `json:"explain"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return 20, "local fallback after judge response decode failed"
	}
	if payload.Fluency < 0 || payload.Fluency > 20 {
		return 20, "local fallback after judge returned invalid fluency"
	}
	if payload.Explain == "" {
		payload.Explain = "AI stub fluency score"
	}
	return payload.Fluency, payload.Explain
}
