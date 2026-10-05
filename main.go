package main

import (
	"embed"
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
	"go-sd/internal/sd"
	"go-sd/internal/serverclient"
	"go-sd/internal/settings"
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

// иконка окна под Linux (Windows/macOS берут её из build/ при сборке)
//
//go:embed build/appicon.png
var appIcon []byte

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
	srvClient := serverclient.NewClient()

	settings.Restore(presets, cfg, llmClient, sdClient, srvClient)

	app := NewApp(presets, llmClient, sdClient, srvClient, cfg)
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
			Icon:             appIcon,
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
