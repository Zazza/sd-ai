package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"
	"go-sd/internal/serverclient"
)

var version = "dev"

func init() {
	if version == "dev" {
		info, ok := debug.ReadBuildInfo()
		if ok {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" {
					version = "dev-" + s.Value[:7]
					break
				}
			}
		}
	}
}

//go:embed all:frontend/dist
var assets embed.FS

//go:embed data/presets/*.json
var bundledPresets embed.FS

func main() {
	cfg := config.Load()

	presets, err := preset.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer presets.Close()

	if err := presets.SeedBundled(bundledPresets); err != nil {
		log.Printf("Warning: bundled presets seed failed: %v", err)
	}

	llmClient := llm.New(cfg.LLMUrl, cfg.LLMBackend)
	sdClient := sd.New(cfg.SDUrl)
	rembgClient := rembg.New("")
	srvClient := serverclient.NewClient()

	if v, _ := presets.GetSetting("llm_url"); v != "" {
		cfg.LLMUrl = v
		llmClient.SetURL(v)
	}
	if v, _ := presets.GetSetting("sd_url"); v != "" {
		cfg.SDUrl = v
		sdClient.SetURL(v)
	}
	if v, _ := presets.GetSetting("llm_model"); v != "" {
		cfg.LLMModel = v
	}
	if v, _ := presets.GetSetting("sd_prompt_model"); v != "" {
		cfg.SDPromptModel = v
	}
	if v, _ := presets.GetSetting("vision_model"); v != "" {
		cfg.VisionModel = v
	}
	if v, _ := presets.GetSetting("llm_backend"); v != "" {
		cfg.LLMBackend = v
		llmClient.SetBackend(v)
	}

	// Server mode: override URLs to proxy through server
	if mode, _ := presets.GetSetting("connection_mode"); mode == "server" {
		if serverURL, _ := presets.GetSetting("server_url"); serverURL != "" {
			srvClient.SetBaseURL(serverURL)
			sdURL, llmURL, rembgURL := srvClient.ProxyURLs()
			cfg.SDUrl = sdURL
			cfg.LLMUrl = llmURL
			sdClient.SetURL(sdURL)
			llmClient.SetURL(llmURL)
			llmClient.SetBackend("ollama")
			rembgClient.SetURL(rembgURL)
		}
	}

	var backendCfg llm.BackendConfig
	if v, _ := presets.GetSetting("llm_keep_alive"); v != "" {
		backendCfg.KeepAlive = v
	} else {
		backendCfg.KeepAlive = "5m"
	}
	if v, _ := presets.GetSetting("llm_num_ctx"); v != "" {
		fmt.Sscanf(v, "%d", &backendCfg.NumCtx)
	} else {
		backendCfg.NumCtx = 4096
	}
	if v, _ := presets.GetSetting("llm_num_gpu"); v != "" {
		fmt.Sscanf(v, "%d", &backendCfg.NumGPU)
	}
	llmClient.SetBackendConfig(backendCfg)

	app := NewApp(presets, llmClient, sdClient, rembgClient, srvClient, cfg)
	imgHandler := &imageFileHandler{db: presets, dataDir: filepath.Dir(cfg.DBPath)}

	if err := wails.Run(&options.App{
		Title:     "SD Studio",
		Width:     1280,
		Height:    800,
		Frameless: false,
		MinWidth:  900,
		MinHeight: 600,
		MaxWidth:  7680,
		MaxHeight: 4320,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: imgHandler,
		},
		OnStartup:  app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			WebviewGpuPolicy: linux.WebviewGpuPolicyAlways,
		},
	}); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

type imageFileHandler struct {
	db      *preset.DB
	dataDir string
}

func (h *imageFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	var dir string
	var idStr string

	if strings.HasPrefix(p, "/api/img/") {
		idStr = strings.TrimPrefix(p, "/api/img/")
		dir = "sessions"
	} else if strings.HasPrefix(p, "/api/thumb/") {
		idStr = strings.TrimPrefix(p, "/api/thumb/")
		dir = "thumbs"
	} else {
		http.NotFound(w, r)
		return
	}

	idStr = strings.TrimSuffix(idStr, ".jpg")
	idStr = strings.TrimSuffix(idStr, ".png")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	item, err := h.db.GetSessionItem(id)
	if err != nil || item == nil {
		http.NotFound(w, r)
		return
	}

	var fileName string
	if dir == "sessions" {
		fileName = item.FileName
	} else {
		fileName = item.ThumbName
	}
	if fileName == "" {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join(h.dataDir, dir, strconv.FormatInt(item.SessionID, 10), fileName)
	if _, err := os.Stat(filePath); err != nil {
		http.NotFound(w, r)
		return
	}

	contentType := "image/jpeg"
	if strings.HasSuffix(fileName, ".png") {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, filePath)
}
