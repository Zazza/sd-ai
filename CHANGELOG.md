# Changelog

All notable changes to SD Studio are documented here.

## [0.9.6] — 2026-10-08

### Added
- **Window icon**: the title-bar/taskbar window icon now uses the app logo (a dedicated 256×256 PNG embedded via `go:embed` and wired into the Wails Linux options — the 1024px packaging icon stays for builds/installers).
- **Local builds stamp their version**: `make build` passes `-ldflags "-X main.version=$(git describe --tags --abbrev=0)"`, so a locally built binary shows the real version (e.g. v0.9.5) instead of "vdev"; CI releases already stamped it.
- **Resolution picker in From Image (Remix)**: the params block (next to Denoising/Seed) now has a `ResolutionSelector` with a leading "Match original" option — `null` keeps the current behavior (backend stores the original size), picking a resolution sends `resolution_id` in `EnqueueFromImage` params (backend fits the original into the chosen resolution preserving aspect ratio, upscaling included). The choice is page-local and is not persisted.
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
