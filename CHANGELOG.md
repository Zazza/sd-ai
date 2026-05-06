# Changelog

All notable changes to SD Studio are documented here.

## [Unreleased]

## [0.5.2] — 2026-05-06

### Changed
- Refactored app.go God Object (5136 → 841 lines) into service modules: generation, session, settings, importexport, filebrowser, promptutil
- Business logic extracted to `internal/generation/` (service.go + analyze.go)
- Old `internal/analyze/` package removed, merged into `internal/generation/`

### Added
- 475 tests across 9 previously untested packages: promptutil (56), filebrowser (28), rembg (20), session (37), settings (32), importexport (51), generation (132), compositor (69), api (48)
- CI test gate — `go test` + `go vet` must pass before release build

## [0.5.1] — 2026-05-05

### Added
- SD generation progress polling — real-time progress bar with ETA and live preview
- Interrupt generation — cancel ongoing SD generation via UI button
- LLM status events — "thinking" indicator during prompt generation
- `useGenerationProgress` composable — shared progress logic across all generation pages
- `sd.GetProgress()` / `sd.Interrupt()` — new SD WebUI API methods
- Russian README (`README-ru.md`) with cross-language navigation
- CHANGELOG.md
- Bilingual docs — all `docs/` files split into `*-en.md` / `*-ru.md` with translations
- `docs/screenshots/` folder for README images
- App version in footer — injected via ldflags at build time
- GitHub Actions release workflow — cross-platform builds (macOS, Windows, Linux) on tag push

## [0.5.0] — 2025-05-05

### Added
- LLM prompt engineering — natural language description merged into SD prompt
- Smart Remove — AI-powered object removal with LLM vision context analysis
- Multi-scene composition — scene decomposition + multi-pass inpaint compositing
- Compound presets — chain txt2img → img2img → inpaint into pipelines
- Session management — project-based sessions with full generation history
- Kids mode — PIN protection with content filtering by category
- Image analysis — quick and deep chain mode via vision LLM
- Batch generation — generate N images with progress tracking
- File browser — thumbnail grid, fullscreen viewer
- Export — resize, convert (PNG/JPEG/WebP), quality/interpolation control
- Light/dark theme — system-aware with manual toggle
- Import/export presets — with model validation
- Mask editor — canvas with fullscreen mode, brush controls, undo, dilation, feathering
- SD WebUI retry with exponential backoff (3 attempts)
- Docker support
