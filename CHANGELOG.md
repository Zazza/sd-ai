# Changelog

All notable changes to SD Studio are documented here.

## [0.9.3] — 2026-10-06

### Changed
- **Searchable style picker in the pipeline editor**: the per-step preset `<select>` on the Pipelines page (300+ entries in one flat list) is replaced by a reusable `SearchableSelect` combobox — type to filter by preset name or style type (hint on the right), full keyboard navigation (arrows + Enter, Esc cancels and restores the chosen label, ✕ clears the step back to "no style"). Pure Vue, no dependencies; other selects are untouched.
