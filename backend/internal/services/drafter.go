package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/resumes"
	"gopkg.in/yaml.v3"
)

type projectBank struct {
	Projects []struct {
		Name       string   `yaml:"name"`
		TrackHints []string `yaml:"track_hints"`
		Bullets    []string `yaml:"bullets"`
	} `yaml:"projects"`
	Candidate struct {
		Name      string   `yaml:"name"`
		Portfolio string   `yaml:"portfolio"`
		Strengths []string `yaml:"strengths"`
	} `yaml:"candidate"`
}

// Drafter generates cover letters via OpenAI using only project_bank facts.
type Drafter struct {
	Store   *db.Store
	DataDir string
	Client  *http.Client
	bank    *projectBank
}

func (d *Drafter) Enabled() bool {
	return strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != ""
}

// defaultLLMBaseURL is used when OPENAI_API_BASE_URL is not set.
const defaultLLMBaseURL = "https://api.openai.com/v1"

// LLMBaseURL resolves the OpenAI-compatible API base URL.
// Accepts either a base (https://host/v1) or a full chat-completions endpoint.
func LLMBaseURL() string {
	base := strings.TrimSpace(os.Getenv("OPENAI_API_BASE_URL"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	}
	if base == "" {
		base = defaultLLMBaseURL
	}
	return strings.TrimRight(base, "/")
}

// chatCompletionsURL builds the full chat completions endpoint.
func chatCompletionsURL() string {
	base := LLMBaseURL()
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func (d *Drafter) loadBank() error {
	if d.bank != nil {
		return nil
	}
	path := filepath.Join(d.DataDir, "project_bank.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w — copy project_bank.yaml.example to project_bank.yaml and add your projects", path, err)
	}
	var bank projectBank
	if err := yaml.Unmarshal(b, &bank); err != nil {
		return err
	}
	d.bank = &bank
	return nil
}

// DraftAndSave writes a cover letter (+ optional Q&A) into applications.
func (d *Drafter) DraftAndSave(ctx context.Context, job *models.Job, match *models.Match) error {
	if !d.Enabled() {
		return fmt.Errorf("OPENAI_API_KEY not set")
	}
	if err := d.loadBank(); err != nil {
		return err
	}

	track := "fullstack"
	if match != nil && match.Track != "" {
		track = match.Track
	}
	resumePath, err := resumes.Path(track)
	if err != nil {
		resumePath, _ = resumes.Path("fullstack")
	}

	letter, qa, err := d.callOpenAI(ctx, job, match, track)
	if err != nil {
		return err
	}
	_, err = d.Store.UpsertApplication(job.ID, &letter, &resumePath)
	if err != nil {
		return err
	}
	if qa != "" {
		_ = d.Store.SetApplicationQA(job.ID, qa)
	}
	return nil
}

func (d *Drafter) callOpenAI(ctx context.Context, job *models.Job, match *models.Match, track string) (string, string, error) {
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	facts := d.bankFacts(track)
	// Address the candidate by the name in their facts file; fall back to a
	// neutral phrasing when it is missing or still the example placeholder.
	who := d.bank.Candidate.Name
	if who == "" || who == "Your Name" {
		who = "the candidate"
	}
	sys := fmt.Sprintf(`You write concise, professional cover letters for %s.
Use ONLY the project facts provided. Do not invent employers, metrics, or technologies.
Return JSON: {"cover_letter":"markdown string","custom_qa":{"Why this company?":"...","Relevant experience":"..."}}`, who)

	user := fmt.Sprintf(
		"Track: %s\nRole: %s @ %s\nMatch notes: %s\n\nJob description:\n%s\n\nProject facts:\n%s\nPortfolio: %s",
		track, job.Title, job.Company, matchReason(match), truncate(job.Description, 6000), facts, d.bank.Candidate.Portfolio,
	)

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": sys},
			{"role": "user", "content": user},
		},
		"temperature":     0.4,
		"response_format": map[string]string{"type": "json_object"},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL(), bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode >= 400 {
		return "", "", fmt.Errorf("openai HTTP %d: %s", res.StatusCode, truncate(string(raw), 300))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", "", err
	}
	if len(parsed.Choices) == 0 {
		return "", "", fmt.Errorf("openai empty choices")
	}
	content := parsed.Choices[0].Message.Content
	var out struct {
		CoverLetter string          `json:"cover_letter"`
		CustomQA    json.RawMessage `json:"custom_qa"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return content, "", nil
	}
	qa := ""
	if len(out.CustomQA) > 0 {
		qa = string(out.CustomQA)
	}
	return out.CoverLetter, qa, nil
}

func (d *Drafter) bankFacts(track string) string {
	var b strings.Builder
	b.WriteString("Candidate: ")
	b.WriteString(d.bank.Candidate.Name)
	b.WriteString("\nStrengths: ")
	b.WriteString(strings.Join(d.bank.Candidate.Strengths, "; "))
	b.WriteString("\n")
	for _, p := range d.bank.Projects {
		relevant := len(p.TrackHints) == 0
		for _, h := range p.TrackHints {
			if h == track {
				relevant = true
				break
			}
		}
		if !relevant && track != "fullstack" {
			// still include briefly
		}
		b.WriteString("\n## ")
		b.WriteString(p.Name)
		b.WriteString("\n")
		for _, bullet := range p.Bullets {
			b.WriteString("- ")
			b.WriteString(bullet)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func matchReason(m *models.Match) string {
	if m == nil || m.ScoreReasons == nil {
		return ""
	}
	return *m.ScoreReasons
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
