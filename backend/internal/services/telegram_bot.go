package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/models"
)

// TelegramBot long-polls Telegram when configured.
type TelegramBot struct {
	Store    *db.Store
	Ingestor *Ingestor
	Token    string
	ChatID   string
	Client   *http.Client
	UIBase   string
	offset   int64
}

func NewTelegramBot(store *db.Store, ing *Ingestor) *TelegramBot {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chat := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))
	if token == "" || chat == "" {
		return nil
	}
	ui := os.Getenv("DASHBOARD_URL")
	if ui == "" {
		ui = "http://localhost:3000"
	}
	return &TelegramBot{
		Store:    store,
		Ingestor: ing,
		Token:    token,
		ChatID:   chat,
		Client:   &http.Client{Timeout: 60 * time.Second},
		UIBase:   strings.TrimRight(ui, "/"),
	}
}

func (t *TelegramBot) Enabled() bool { return t != nil && t.Token != "" && t.ChatID != "" }

// NotifyHighMatch sends an alert with inline actions.
func (t *TelegramBot) NotifyHighMatch(job *models.Job, match *models.Match) {
	if !t.Enabled() || job == nil || match == nil {
		return
	}
	text := fmt.Sprintf(
		"High Match (%.0f%%)\n%s @ %s\nTrack: %s\n%s\n\n%s/jobs/%d",
		match.Score*100, job.Title, job.Company, match.Track, deref(match.ScoreReasons), t.UIBase, job.ID,
	)
	keyboard := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "Approve", "callback_data": fmt.Sprintf("approve:%d", job.ID)},
				{"text": "Discard", "callback_data": fmt.Sprintf("discard:%d", job.ID)},
			},
		},
	}
	_ = t.api("sendMessage", map[string]any{
		"chat_id":      t.ChatID,
		"text":         text,
		"reply_markup": keyboard,
	})
}

// Run starts long polling until ctx is done.
func (t *TelegramBot) Run(ctx context.Context) {
	if !t.Enabled() {
		return
	}
	log.Printf("telegram bot polling chat %s", t.ChatID)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		updates, err := t.getUpdates(ctx)
		if err != nil {
			log.Printf("telegram getUpdates: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			if u.UpdateID >= t.offset {
				t.offset = u.UpdateID + 1
			}
			t.handleUpdate(u)
		}
	}
}

type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
	CallbackQuery *struct {
		ID      string `json:"id"`
		Data    string `json:"data"`
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
}

func (t *TelegramBot) getUpdates(ctx context.Context) ([]tgUpdate, error) {
	q := url.Values{}
	q.Set("timeout", "25")
	q.Set("offset", strconv.FormatInt(t.offset, 10))
	var resp struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := t.apiGet(ctx, "getUpdates?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (t *TelegramBot) handleUpdate(u tgUpdate) {
	if u.CallbackQuery != nil {
		_ = t.api("answerCallbackQuery", map[string]any{"callback_query_id": u.CallbackQuery.ID})
		data := u.CallbackQuery.Data
		parts := strings.SplitN(data, ":", 2)
		if len(parts) != 2 {
			return
		}
		id, _ := strconv.ParseInt(parts[1], 10, 64)
		switch parts[0] {
		case "discard":
			status := "rejected"
			_, _ = t.Store.UpdateJob(id, &models.JobPatch{Status: &status})
			// The operator decided from Telegram: the eligibility engine must
			// not move this card again on the next rules re-check.
			_ = t.Store.ClearEligibilityApplied(id)
			t.reply(t.ChatID, fmt.Sprintf("Discarded job #%d", id))
		case "approve":
			status := "approved"
			_, _ = t.Store.UpdateJob(id, &models.JobPatch{Status: &status})
			_ = t.Store.ClearEligibilityApplied(id)
			t.reply(t.ChatID, fmt.Sprintf(
				"Job #%d marked approved. Submission dispatcher not implemented yet (Phase 5) — application was NOT sent.",
				id,
			))
		}
		return
	}
	if u.Message == nil {
		return
	}
	chatID := strconv.FormatInt(u.Message.Chat.ID, 10)
	if chatID != t.ChatID {
		// still allow configured chat only
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	switch {
	case strings.HasPrefix(text, "/stats"):
		st, err := t.Store.GetStats()
		if err != nil {
			t.reply(chatID, err.Error())
			return
		}
		t.reply(chatID, fmt.Sprintf(
			"Jobs: %d\nApplied: %d\nInterviews: %d\nBy status: %v",
			st.TotalJobs, st.Applied, st.Interviews, st.ByStatus,
		))
	case strings.HasPrefix(text, "/today"):
		limit := 8
		if raw := os.Getenv("DAILY_APPLICATION_LIMIT"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				limit = n
			}
		}
		applied, _ := t.Store.CountJobsByStatusToday("applied", "approved", "queued")
		t.reply(chatID, fmt.Sprintf("Today pipeline actions (applied/approved/queued): %d / daily limit %d", applied, limit))
	case strings.HasPrefix(text, "/status"):
		if t.Ingestor == nil {
			t.reply(chatID, "Ingestor not configured")
			return
		}
		st := t.Ingestor.Status()
		t.reply(chatID, fmt.Sprintf(
			"Ingest running=%v\n%s\ninserted=%d scored=%d drafted=%d\nerrors=%v",
			st.Running, st.Message, st.Inserted, st.Scored, st.Drafted, st.SourceErrors,
		))
	case strings.HasPrefix(text, "/start"), strings.HasPrefix(text, "/help"):
		t.reply(chatID, "Commands: /stats /today /status\nInline Approve/Discard on match alerts.")
	}
}

func (t *TelegramBot) reply(chatID, text string) {
	_ = t.api("sendMessage", map[string]any{"chat_id": chatID, "text": text})
}

func (t *TelegramBot) api(method string, payload map[string]any) error {
	body, _ := jsonMarshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+t.Token+"/"+method, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (t *TelegramBot) apiGet(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/bot"+t.Token+"/"+path, nil)
	if err != nil {
		return err
	}
	res, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return jsonDecode(res.Body, dest)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
