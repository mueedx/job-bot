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

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = resolveDataDir()
	}

	sqlDB, err := db.Open(dataDir)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer sqlDB.Close()

	store := db.NewStore(sqlDB)
	drafter := &services.Drafter{Store: store, DataDir: dataDir}
	ingestor := &services.Ingestor{Store: store, DataDir: dataDir, Drafter: drafter}

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

	server := &api.Server{Store: store, Ingestor: ingestor, Drafter: drafter}
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

// resolveDataDir prefers ./data and falls back to ../data, so the server finds
// the same directory whether it is started from the repo root or from backend/.
func resolveDataDir() string {
	if isDir("data") {
		return "data"
	}
	parent := filepath.Join("..", "data")
	if isDir(parent) {
		return parent
	}
	return "data"
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
