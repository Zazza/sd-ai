package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"go-sd/internal/config"
	"go-sd/internal/generation"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/mcp"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
	"go-sd/internal/serverclient"
	"go-sd/internal/settings"
)

func main() {
	log.SetFlags(0)

	dbPath := flag.String("db", "", "путь к presets.db (по умолчанию env SD_MCP_DB, затем config.Load)")
	flag.Parse()

	cfg := config.Load()
	if *dbPath != "" {
		cfg.DBPath = *dbPath
	} else if v := os.Getenv("SD_MCP_DB"); v != "" {
		cfg.DBPath = v
	}

	db, err := preset.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	llmClient := llm.New(cfg.LLMUrl, cfg.LLMBackend)
	sdClient := sd.New(cfg.SDUrl)
	srvClient := serverclient.NewClient()

	settings.Restore(db, cfg, llmClient, sdClient, srvClient)

	logSvc := logger.New(nil)
	kidsMgr := kids.NewManager(db)
	settingsSvc := settings.New(db, llmClient, sdClient, cfg, logSvc, srvClient)
	gen := generation.New(
		db, llmClient, sdClient, cfg,
		filepath.Dir(cfg.DBPath),
		mcp.NoopEmitter{}, kidsMgr, mcp.NoopSessions{}, settingsSvc, logSvc,
	)

	outDir := os.Getenv("SD_MCP_OUT")
	if outDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("home dir: %v", err)
		}
		outDir = filepath.Join(home, "sd-mcp-out")
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		log.Fatalf("create out dir: %v", err)
	}

	srv := mcp.NewServer(mcp.Deps{
		DB:       db,
		Gen:      gen,
		SD:       sdClient,
		LLM:      llmClient,
		Settings: settingsSvc,
		Kids:     kidsMgr,
		Cfg:      cfg,
		OutDir:   outDir,
	})
	mcp.RegisterTools(srv)

	if err := srv.Serve(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("mcp serve: %v", err)
	}
}
