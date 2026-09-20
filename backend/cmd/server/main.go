package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/mueedx/job-bot/backend/internal/api"
	"github.com/mueedx/job-bot/backend/internal/db"
	"github.com/mueedx/job-bot/backend/internal/resumes"
	"github.com/mueedx/job-bot/backend/internal/services"
)

func main() {
	// Prefer repo-root .env, then backend/.env (cwd when running from backend/).
	_ = godotenv.Load(filepath.Join("..", ".env"))
	_ = godotenv.Load(".env")

	// resolveDataDir consults DATA_DIR itself, so a relative setting that does
	// not exist cannot silently split the database in two.
	dataDir := resolveDataDir()
	log.Printf("Data: reading %s (sqlite + editable YAML)", dataDir)

	sqlDB, err := db.Open(dataDir)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer sqlDB.Close()

		store := db.NewStore(sqlDB)
	drafter := &services.Drafter{Store: store, DataDir: dataDir}

	// ResumeAnalyzer: extracts keywords from resume PDFs via OpenAI.
	// Reuses the same BaseURL / API key / model as the Drafter so there's one
	// env-driven LLM configuration. Without OPENAI_API_KEY it stays inactive
	// and matching falls back to built-in keyword heuristics.
	analyzer := services.NewResumeAnalyzer(store, &services.OpenAIClient{
		BaseURL: services.LLMBaseURL(),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("LLM_MODEL"),
	})
	analyzer.Enable()

	ingestor := &services.Ingestor{Store: store, DataDir: dataDir, Analyzer: analyzer, Drafter: drafter}

	// Say out loud where resumes are read from: silent misconfiguration here
	// otherwise shows up much later as a wrong (or missing) PDF on an application.
	resumeStatus := resumes.Inspect()
	log.Printf("Resumes: reading %s (%d PDF file(s))", resumeStatus.Dir, len(resumeStatus.Files))
	if len(resumeStatus.Missing) > 0 {
		log.Printf("Resumes: no PDF for track(s) %s — add <track>.pdf or default.pdf in %s, see %s/README.md",
			strings.Join(resumeStatus.Missing, ", "), resumeStatus.Dir, resumes.DefaultDir)
	}

	bot := services.NewTelegramBot(store, ingestor)
	if bot != nil {
		ingestor.Notifier = bot.NotifyHighMatch
	}

		server := &api.Server{Store: store, Ingestor: ingestor, Drafter: drafter, Analyzer: analyzer, DataDir: dataDir}
	router := api.NewRouter(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	services.StartIngestCron(ingestor)
	if bot != nil {
		go bot.Run(ctx)
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8000"
	}

	httpServer := &http.Server{Addr: addr, Handler: router}
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()

	log.Printf("Job Agent API listening on http://localhost%s", addr)
	log.Printf("Swagger docs: http://localhost%s/docs", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

// resolveDataDir decides where the SQLite database and the editable YAML live.
//
// DATA_DIR wins when it is usable. A *relative* DATA_DIR that does not exist is
// a trap: starting the server from backend/ with DATA_DIR=./data used to create a
// second, empty database in backend/data, so the ingestor wrote to one database
// while the dashboard read another — and the board sources silently lost their
// target_companies.yaml. A missing relative path therefore falls back to
// detection, while absolute paths are honoured as given (Docker sets /data).
func resolveDataDir() string {
	if configured := strings.TrimSpace(os.Getenv("DATA_DIR")); configured != "" {
		if isDir(configured) || filepath.IsAbs(configured) {
			return configured
		}
		log.Printf("DATA_DIR=%q is not a directory; using the detected data directory instead", configured)
	}
	// Prefer the repository root's data/ so running from the repo root and from
	// backend/ share one database instead of quietly creating two.
	if root, ok := repoRoot(); ok {
		if dir := filepath.Join(root, "data"); isDir(dir) {
			return dir
		}
	}
	if isDir("data") {
		return "data"
	}
	return filepath.Join("..", "data")
}

// repoRoot walks up from the working directory looking for the repository root,
// which is the directory holding the Makefile.
func repoRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
