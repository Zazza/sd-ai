# Changelog

All notable changes to SD Studio are documented here.

## [Unreleased]

## [0.5.4] — 2026-05-06

### Added
- Fast Save button on generated image — modal dialog with filename input, saves directly to FileBrowser folder
- Fast Save format selector: JPG (default) / PNG
- FileBrowser: directory path now persisted between sessions
- Clearing description/negative triggers prompt regeneration via LLM (or returns preset base prompt)
- Changing preset or pipeline also triggers prompt regeneration

### Changed
- Removed WebP support across entire app (export, file browser, DB defaults, dependencies)

### Fixed
- Fast Save: auto-detects image format from SD (PNG or JPEG) instead of assuming PNG

## [0.5.3] — 2026-05-06

### Added
- SD Forge compatibility — automatic fallback when Hires Fix fails (retry without Hires Fix)
- Hires Fix skipped warning in UI when fallback triggers
- Generate page: immediate spinner/status on Generate button click (covers LLM prompt phase)
- Batch page: LLM prompt generation — Description and Negative fields now go through LLM before batch SD generation
- Pipeline mode: description is now sent to LLM for prompt generation (was ignored before)
- Setup docs: two installation variants — A1111 (standard) and Forge (faster, with known limitations note)
- README: Forge support mention, pre-built releases download link

### Fixed
- SD Forge: Hires Fix causing `NoneType` error or connection reset — graceful fallback without Hires Fix
- Generate page: no visual feedback during LLM prompt generation phase
- Batch page: raw description was copied into prompt field instead of being processed by LLM
- Pipeline mode on Generate page: description and negative were ignored, previous prompt sent directly to SD

### Tests
- 3 regression tests for Hires Fix fallback (fallback success, no fallback without HiresFix, fallback still fails)

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
