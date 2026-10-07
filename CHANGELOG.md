# Changelog

All notable changes to SD Studio are documented here.

## [0.9.5] — 2026-10-07

### Fixed
- **Forge memory-manager crashes no longer kill generations**: Forge (Neo) nulls the weakref of a torn-down lazy model object while it stays in the loaded list, so the next garbage-collection pass crashes with a 500 — `TypeError: 'NoneType' object is not callable` or `AttributeError: 'NoneType' object has no attribute 'model_size'/'model'` — and every blind retry hits the same nail (typically on img2img steps of pipelines that switch checkpoints). The SD client now detects the whole `NoneType` family in SD error responses, recovers the server state (unload checkpoint → re-select the current checkpoint, forcing a clean reload) and replays the request once — covering txt2img, img2img and every pipeline step. The local Forge install is also patched at the root: a `_safe_real_model` helper on every direct call site, `is_dead` treats finalized objects as dead, and the size helpers return 0 for a missing model, so the broken object can never crash the GC again (patches are reinstall-on-upgrade, tracked in the project memory).

## [0.9.3] — 2026-10-06

### Changed
- **Searchable style picker in the pipeline editor**: the per-step preset `<select>` on the Pipelines page (300+ entries in one flat list) is replaced by a reusable `SearchableSelect` combobox — type to filter by preset name or style type (hint on the right), full keyboard navigation (arrows + Enter, Esc cancels and restores the chosen label, ✕ clears the step back to "no style"). Pure Vue, no dependencies; other selects are untouched.
