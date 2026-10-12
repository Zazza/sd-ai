# Changelog

All notable changes to SD Studio are documented here.

## [0.9.8] — 2026-10-12

### Fixed
- **Memory guard now checks free VRAM for all models, not just the heavy white-list**: the pre-generation ollama unload only triggered for models from `heavy_models` (flux, z-image, …) before even looking at actual VRAM, so an SDXL-class checkpoint (~7 GB) sharing a 16 GB card with a warm 14B LLM was never relieved (incident 2026-10-12, riMix + ollama); free VRAM/RAM is now checked for every model — 8 GiB VRAM headroom for regular models, 12 GiB for the heavy list.

### Added
- **Image viewer: collapsible info panel + zoom**: the bottom metadata/generation-chain panel no longer covers the image — it is collapsed by default and toggled by an "ℹ" button (Remix/Export/Delete stay available without expanding, in a compact row). New zoom controls (`[−] [Fit|NN%] [+]`): fit-to-window by default (as before), 30/50/100% presets via the indicator, ±10% steps (10–500%), double-click toggles fit/100%, Ctrl+wheel zooms; when the image overflows the window it can be dragged (grab cursor, edges stay clamped to the window, bar-aware). Switching images resets to fit; zoom clamps and fit-percentage react to window resize.
- **Generation metadata in the session history**: every image in the footer history now remembers what created it — the model, LoRAs, the preset or pipeline name (Custom-mode runs show as "Custom"), and for multi-step pipelines the full per-step chain (preset, model, denoise, LoRAs; quality mode shows its base → upscale → detail chain with skipped steps omitted). Stored as a single `session_items.meta` JSON column (migration V28; old items simply have none). The model is taken from the generation context with the SD response's `sd_model_name` as fallback; oversized metadata degrades (drops LoRAs, then the chain) instead of failing. UI: thumbnail tooltip in the footer (label/model/seed/step count) plus a step-count badge, and a "Generation chain" panel in the image viewer; entries without metadata degrade to the existing fields. No new bindings — the metadata travels through the existing `GetSessionItems`.
- **Custom mode (pipeline constructor) on the Generate page**: a third generation mode (Style | Pipeline | Custom) where a multi-step pipeline is assembled inline instead of being pre-built on the Pipelines page — each step picks a preset, a denoising strength (0.05–1, steps 2+; 0 is rejected rather than silently coerced to the 0.5 default) and its own LoRA set (untouched = inherit the step preset's LoRAs, editable with the same LoRA UI as the overrides panel, "disable all" supported), up to 8 steps. The Seed field is available in Custom mode and is applied to every step, so the intended workflow is: fix the seed, then iterate on the presets/denoise of the second and later steps until the result is right. Generation goes through the existing compound engine (queue, progress, hires on the last step, interrupt all work), including batch (`seed + i`, clamped to MAX_SAFE_INTEGER on both Generate and From Image). The assembled configuration can be saved as a regular pipeline ("Save as pipeline" with name/description) — per-step LoRAs are persisted via a new `compound_preset_steps.loras` column (empty = inherit, `[]` = off; the Pipelines editor and duplicate preserve them, and they apply in all compound entry points: Generate, From Image, Test), and are included in export/import (invalid/oversized LoRA payloads are dropped on import). The editor state persists between launches (`gen_custom_steps`); untouched steps re-sync their LoRA rows from the current preset on restore (library changes don't leave stale rows). `"custom"` never leaks into From Image via shared state (sanitized to `"preset"`). Removed the now-dead "Using multi-step workflows?" toggle link.

## [0.9.7] — 2026-10-09

### Added
- **Quality mode in From Image (Remix)**: a "Quality mode" checkbox (visible for preset + img2img with a resolution selected) runs a 3-step pipeline instead of stretching the source image to the target size: img2img at the source's native size (downscale-only fit into the chosen resolution box, never upscaled) → ESRGAN neural upscale (R-ESRGAN 4x+, Catmull-Rom fallback) into the box preserving the source aspect ratio → a detail img2img pass at denoise 0.35. If the upscale or the detail pass fails, the last good image is returned with a visible notice ("Upscale skipped…"/"Detail pass skipped…"). Validation rejects quality mode outside preset+img2img or without a resolution (enqueue + entry + MCP). The mode works in batch too (3 SD operations per image), persists per page (`fi_quality_mode`), and is exposed to MCP via `from_image` props `quality`/`resolution_id`.

### Fixed
- **Hires profile is now applied in From Image Pipeline mode**: the compound (pipeline) path on From Image ignored the selected hires profile — the last step's image is now upscaled + re-denoised via the profile (same manual hires mechanism as the Generate page pipelines), including single-step pipelines; on hires failure (or when no profile is selected) the last good image is returned with its metadata (session item info / job result info) preserved.
- **Fields no longer vanish when the gen-mode toggle is on "Pipeline"**: on From Image, the Seed field and the "Quality mode" checkbox used to disappear entirely in Pipeline mode (looking like the feature doesn't exist — the choice is also persisted between launches). They now stay visible but dimmed and disabled with an "Available in Style mode only" hint; the 3-operations hint only shows when the checkbox is actually usable. Regression-tested (6 vitest cases).
- **Huge results are no longer silently dropped from the session**: `AddToSession` re-encodes images over the 50 MB base64 cap to JPEG (Q95) instead of quietly discarding them — reachable via quality mode at 4K resolutions; a failed re-encode now logs a warning before dropping.

## [0.9.6] — 2026-10-08

### Added
- **Window icon**: the title-bar/taskbar window icon now uses the app logo (a dedicated 256×256 PNG embedded via `go:embed` and wired into the Wails Linux options — the 1024px packaging icon stays for builds/installers).
- **Local builds stamp their version**: `make build` passes `-ldflags "-X main.version=$(git describe --tags --abbrev=0)"`, so a locally built binary shows the real version (e.g. v0.9.5) instead of "vdev"; CI releases already stamped it.
- **Resolution picker in From Image (Remix)**: the params block (next to Denoising/Seed) now has a `ResolutionSelector` with a leading "Match original" option — `null` keeps the current behavior (backend stores the original size), picking a resolution sends `resolution_id` in `EnqueueFromImage` params (backend fits the original into the chosen resolution preserving aspect ratio, upscaling included). The choice is page-local and is not persisted.

### Changed
- **Safer destructive buttons**:
  - Queue "Cancel All" now requires a second click ("Cancel?" confirm state, resets after 3 s), same as "Clear Done".
  - Queue footer buttons got a stable `min-width` (wider for the morphing clear button) plus `white-space: nowrap`, so the "…→ confirm?" text morph no longer changes button size or shifts neighbors.
  - Session "Delete History" in the footer shows a native `window.confirm()` dialog ("Delete the session and all its files? This action cannot be undone.") after the inline two-click confirm, before files are removed; the button width is also fixed.
  - New i18n keys in both RU and EN: `fi.resolution_original`, `queue.confirm_cancel`, `footer.confirm_delete_dialog`.

## [0.9.5] — 2026-10-07

### Fixed
- **Forge memory-manager crashes no longer kill generations**: Forge (Neo) nulls the weakref of a torn-down lazy model object while it stays in the loaded list, so the next garbage-collection pass crashes with a 500 — `TypeError: 'NoneType' object is not callable` or `AttributeError: 'NoneType' object has no attribute 'model_size'/'model'` — and every blind retry hits the same nail (typically on img2img steps of pipelines that switch checkpoints). The SD client now detects the whole `NoneType` family in SD error responses, recovers the server state (unload checkpoint → re-select the current checkpoint, forcing a clean reload) and replays the request once — covering txt2img, img2img and every pipeline step. The local Forge install is also patched at the root: a `_safe_real_model` helper on every direct call site, `is_dead` treats finalized objects as dead, and the size helpers return 0 for a missing model, so the broken object can never crash the GC again (patches are reinstall-on-upgrade, tracked in the project memory).

## [0.9.3] — 2026-10-06

### Changed
- **Searchable style picker in the pipeline editor**: the per-step preset `<select>` on the Pipelines page (300+ entries in one flat list) is replaced by a reusable `SearchableSelect` combobox — type to filter by preset name or style type (hint on the right), full keyboard navigation (arrows + Enter, Esc cancels and restores the chosen label, ✕ clears the step back to "no style"). Pure Vue, no dependencies; other selects are untouched.
