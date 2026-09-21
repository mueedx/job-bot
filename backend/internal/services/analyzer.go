package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
)

type ResumeAnalyzer struct {
	store     *db.Store
	openaiKey string
	client    HTTPClient
	model     string
	once      bool
	enabled   bool
}

// OpenAIClient is the default HTTPClient implementation for OpenAI-compatible
// endpoints. It mirrors how Drafter.callOpenAI makes requests.
type OpenAIClient struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// PostJSON implements HTTPClient by calling an OpenAI-compatible chat endpoint.
func (c *OpenAIClient) PostJSON(ctx context.Context, url string, body any) (*HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &HTTPResponse{Status: resp.StatusCode, Body: respBytes}, nil
}

type HTTPClient interface {
	PostJSON(ctx context.Context, url string, body any) (*HTTPResponse, error)
}

type HTTPResponse struct {
	Status int
	Body   []byte
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type ExtractedProfile struct {
	ResumePath  string
	ContentHash string
	Track       string
	Skills      *string
	Keywords    *string
	Seniority   string
	Summary     string
	Source      string
}

func NewResumeAnalyzer(store *db.Store, client HTTPClient) *ResumeAnalyzer {
	return &ResumeAnalyzer{store: store, client: client}
}

func (a *ResumeAnalyzer) Enable() {
	if a.once {
		return
	}
	a.once = true
	a.openaiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	a.enabled = a.openaiKey != "" && a.client != nil
}

func (a *ResumeAnalyzer) Enabled() bool {
	a.Enable()
	return a.enabled
}

const resumePrompt = `Read this resume text and return ONLY a JSON object with exactly these fields:
{
  "track": one of "fullstack" | "blockchain" | "fde",
  "skills": array of specific technologies/tools/languages actually mentioned,
  "keywords": array of domain terms and phrases (e.g. "smart contract", "mcp", "rag", "evm"),
  "seniority": one of "junior" | "mid" | "senior" | "staff",
  "summary": one sentence describing the candidate's primary specialty
}
If something is absent, use an empty array for skills/keywords and "unknown" for other fields.
Do not include any explanation, only the JSON object.`

func (a *ResumeAnalyzer) AnalyzePDF(ctx context.Context, resumePath string) (*ExtractedProfile, error) {
	text, hash, err := extractPDFText(resumePath)
	if err != nil {
		return nil, fmt.Errorf("read resume %s: %w", resumePath, err)
	}

	existing, _ := a.store.GetResumeProfile(resumePath)
	if existing != nil && existing.ContentHash == hash {
		return &ExtractedProfile{
			ResumePath:  existing.ResumePath,
			ContentHash: existing.ContentHash,
			Track:       existing.Track,
			Skills:      existing.Skills,
			Keywords:    existing.Keywords,
			Seniority:   existing.Seniority,
			Summary:     existing.Summary,
			Source:      "cached",
		}, nil
	}

	if !a.Enabled() {
		rel := relpath(resumePath)
		track := inferTrackFromPath(rel)
		profile := &models.ResumeProfile{
			ResumePath:  rel,
			ContentHash: hash,
			Track:       track,
			Skills:      nil,
			Keywords:    nil,
			Seniority:   "unknown",
			Summary:     "",
			ExtractedAt: time.Now().UTC(),
		}
		if err := a.store.UpsertResumeProfile(profile); err != nil {
			return nil, fmt.Errorf("store fallback profile: %w", err)
		}
		return &ExtractedProfile{
			ResumePath:  rel,
			ContentHash: hash,
			Track:       track,
			Source:      "fallback",
		}, nil
	}

	extracted, err := a.callOpenAI(ctx, text)
	if err != nil {
		track := inferTrackFromPath(resumePath)
		profile := &models.ResumeProfile{
			ResumePath:  resumePath,
			ContentHash: hash,
			Track:       track,
			Skills:      nil,
			Keywords:    nil,
			Seniority:   "unknown",
			Summary:     "",
			ExtractedAt: time.Now().UTC(),
		}
		_ = a.store.UpsertResumeProfile(profile)
		return &ExtractedProfile{
			ResumePath:  resumePath,
			ContentHash: hash,
			Track:       track,
			Source:      "fallback",
		}, nil
	}

	skillsJSON, _ := json.Marshal(extracted.Skills)
	keywordsJSON, _ := json.Marshal(extracted.Keywords)
	skillsStr := string(skillsJSON)
	keywordsStr := string(keywordsJSON)
	profile := &models.ResumeProfile{
		ResumePath:  resumePath,
		ContentHash: hash,
		Track:       extracted.Track,
		Skills:      &skillsStr,
		Keywords:    &keywordsStr,
		Seniority:   extracted.Seniority,
		Summary:     extracted.Summary,
		ExtractedAt: time.Now().UTC(),
	}
	if err := a.store.UpsertResumeProfile(profile); err != nil {
		return nil, fmt.Errorf("persist profile: %w", err)
	}

	extracted.ResumePath = resumePath
	extracted.ContentHash = hash
	extracted.Source = "openai"
	return extracted, nil
}

func (a *ResumeAnalyzer) callOpenAI(ctx context.Context, text string) (*ExtractedProfile, error) {
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": resumePrompt},
			{"role": "user", "content": text},
		},
		"temperature": 0.1,
		"max_tokens":  1500,
	}

	resp, err := a.client.PostJSON(ctx, chatCompletionsURL(), payload)
	if err != nil {
		return nil, fmt.Errorf("openai call: %w", err)
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("openai returned status %d", resp.Status)
	}

	var oaiResp OpenAIResponse
	if err := json.Unmarshal(resp.Body, &oaiResp); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}
	if oaiResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", oaiResp.Error.Message)
	}
	if len(oaiResp.Choices) == 0 {
		return nil, errors.New("openai returned no choices")
	}

	content := strings.TrimSpace(oaiResp.Choices[0].Message.Content)
	content = stripJSONFence(content)

	var extracted struct {
		Track     string   `json:"track"`
		Skills    []string `json:"skills"`
		Keywords  []string `json:"keywords"`
		Seniority string   `json:"seniority"`
		Summary   string   `json:"summary"`
	}
	if err := json.Unmarshal([]byte(content), &extracted); err != nil {
		return nil, fmt.Errorf("parse extracted JSON: %w (raw: %s)", err, truncate(content, 200))
	}

	skillsJSON, _ := json.Marshal(extracted.Skills)
	keywordsJSON, _ := json.Marshal(extracted.Keywords)
	skillsStr := string(skillsJSON)
	keywordsStr := string(keywordsJSON)
	return &ExtractedProfile{
		Skills:    &skillsStr,
		Keywords:  &keywordsStr,
		Track:     extracted.Track,
		Seniority: extracted.Seniority,
		Summary:   extracted.Summary,
	}, nil
}

// MatchJobWithProfiles scores a job against all resume profiles. When AI is
// enabled, it uses extracted keywords per profile; otherwise it delegates to
// MatchJob heuristics. Best-scoring profile's track wins.
func MatchJobWithProfiles(job *models.Job, profiles []ExtractedProfile) MatchResult {
	if len(profiles) == 0 {
		return MatchJob(job)
	}

	best := MatchResult{}
	for _, p := range profiles {
		r := matchAgainstProfile(job, &p)
		if r.Score > best.Score {
			best = r
		}
	}
	return best
}

// matchAgainstProfile scores a job against one resume profile using AI keywords
// when available, plus the built-in heuristics.
func matchAgainstProfile(job *models.Job, p *ExtractedProfile) MatchResult {
	blob := strings.ToLower(job.Title + "\n" + job.Description + "\n" + nullStr(job.Location))

	// Base heuristic score from built-in signal.
	base := MatchJob(job)

	// Augment with AI-extracted keywords if present.
	var profileKeywords []string
	if p.Keywords != nil && *p.Keywords != "" {
		_ = json.Unmarshal([]byte(*p.Keywords), &profileKeywords)
	}

	aiHits := 0
	for _, kw := range profileKeywords {
		if kw != "" && strings.Contains(blob, strings.ToLower(kw)) {
			aiHits++
		}
	}

	if aiHits > 0 {
		aiBoost := minFloat(0.25, float64(aiHits)*0.05)
		base.Score += aiBoost
		if base.Score > 1 {
			base.Score = 1
		}
		base.MatchedSkills = append(base.MatchedSkills, profileKeywords...)
		base.MatchedSkills = unique(base.MatchedSkills)
		base.ScoreReasons = fmt.Sprintf(
			"%s Plus AI keyword signal (%d hits) added %.0f%%.",
			base.ScoreReasons, aiHits, aiBoost*100,
		)
	}

	// Prefer the AI-extracted track when it provided keywords.
	if p.Track != "" && p.Track != "unknown" && len(profileKeywords) > 0 {
		base.Track = p.Track
	}

	return base
}

func extractPDFText(path string) (text string, hash string, err error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	reader, err := r.GetPlainText()
	if err != nil {
		return "", "", err
	}
	buf, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}

	text = string(buf)
	h := sha256.Sum256([]byte(text))
	hash = hex.EncodeToString(h[:])
	return text, hash, nil
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func relpath(path string) string {
	rel, err := filepath.Rel("resumes", path)
	if err != nil {
		return filepath.Base(path)
	}
	return "resumes/" + rel
}

func inferTrackFromPath(path string) string {
	// Token-based matching so "ai" does not match "email" or "available".
	tokens := map[string]bool{}
	for _, tok := range strings.FieldsFunc(strings.ToLower(path), func(r rune) bool {
		return r == '/' || r == '_' || r == '-' || r == '.' || r == ' '
	}) {
		tokens[tok] = true
	}

	switch {
	case tokens["blockchain"] || tokens["web3"] || tokens["solidity"] || tokens["evm"] || tokens["defi"]:
		return "blockchain"
	case tokens["fde"] || tokens["agentic"] || tokens["mcp"] || tokens["rag"] || tokens["ai"] || tokens["solutions"]:
		return "fde"
	default:
		return "fullstack"
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
