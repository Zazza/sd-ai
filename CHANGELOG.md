# Changelog

All notable changes to SD Studio are documented here.

## [0.7.11] — 2026-09-20

### Changed
- **php-chat preset pack renamed to wallpapers**: the bundled pack (22 presets) recreates a set of desktop wallpapers — the old "php-chat" type name referenced only the chat where the original run happened and was confusing. File `data/presets/php-chat.json` → `data/presets/wallpapers.json`, preset type `php-chat` → `wallpapers`, origin mention dropped from tags. Affects fresh DBs and re-imports; existing databases keep the old category name (rename it in the app's category editor if desired).

### Fixed
- **Same default face on every generation (checkpoint face attractor)**: when the user scene mentioned a person without appearance details, the LLM converter emitted no face tags (Rule 4 forbade inventing), so the SD checkpoint rendered its averaged "default face" — on epiCRealism XL every generation converged to the same Bella-Ramsey-like face. The default instruction now has an `APPEARANCE VARIATION` section: if the scene has a person but no appearance, the LLM must fill seven abstract slots (`(age range:1.2)`, `(ethnicity or heritage:1.2)`, `(face shape:1.2)`, `(hair color and texture:1.2)`, `(body type:1.1)`, two distinctive-feature slots) with concrete different values every call; Rules 4 and 6 carve this out explicitly. Slots are abstract placeholders only (no concrete tokens — example-leak guard invariant), so verbatim slot copies are subtracted by the existing runtime filter while invented values survive. `GenerateSDPrompt` temperature raised 0.4 → 0.7 for the roulette to actually vary. From Image is protected: img2img/inpaint converters get an explicit "the base image defines the person — do NOT invent appearance" line, so reference photos are never overwritten. If you had customized `sd_prompt_instruction` in Settings, reset it to the new default to pick this up.

### Changed
- **LLM JSON parse retry**: both prompt converters (Generate, From Image) made no retry when the LLM returned invalid JSON — output silently degraded to raw text (more likely at the higher temperature). New `llmGenerateAndParse` retries once before falling back to the raw-text path.

### Removed
- **Preset recommendation feature («Подобрать стиль» / RecommendPreset)**: the ✨ suggest-style button on Generate and the automatic "Recommended" card shown after Tags analysis in From Image are gone — the feature was unused and was the root cause of the recent "sql: no rows in result set" bug class (an LLM-hallucinated preset ID flowing into generation). Analysis itself is untouched: both From Image analysis buttons (Tags/Text) keep working exactly as before and now run one LLM call faster (no follow-up recommendation round-trip); the "AI Prompt" button on Generate is separate and unaffected. Backend `RecommendPreset`/fallback, the Wails binding, and the transitively dead `llm.ChatJSON`/`ResponseFormat` client surface are removed; wailsjs bindings regenerated. Note: the earlier `RecommendPreset` ID-validation fix from this Unreleased section went out with the feature itself.

### Fixed
- **"sql: no rows in result set" on generation**: a stale or LLM-hallucinated `preset_id` flowing into `GenerateImage` propagated the raw `sql.ErrNoRows` straight into the queue job error. `GenerateImage`, `UpscaleImage` and `UpscalePreview` wrap lookup failures as `preset not found: …` and reject `preset_id ≤ 0` upfront; `GetSessionItem` returns `(nil, nil)` on missing rows (matching `GetActiveItem`), making item deletion idempotent. Frontend defense: Generate and From Image pages verify preset/compound IDs against the loaded lists before assigning them — from persisted settings and cross-page shared state — and From Image now awaits its preset list before restoring settings (validation previously ran against an empty list). (The primary trigger — the recommend feature returning unvalidated LLM IDs — is removed outright, see Removed above.)

## [0.7.10] — 2026-09-18

### Fixed
- **LLM instruction example leak (random bearded man / glowing blue mushrooms)**: the default prompt-converter instruction carried vivid weighted example tags in its conversion guide (`(thick beard:1.3)`, `(giant luminescent mushrooms:1.2)`, `(blue glow from artifact on face:1.2)`). Weak local LLMs occasionally copy guide examples into the response instead of deriving tags from the user scene — and because the examples carry the highest weights (1.2–1.3), SD rendered them dominantly, most often on short descriptions where the model has little real content to convert. Two-layer fix: (1) the guide now uses abstract format placeholders plus an explicit "NEVER copy guide examples" rule; (2) a runtime guard (`promptutil.ExtractExampleTags` + `Service.filterInstructionExamples`) parses `(tag:weight)` examples out of the *active* instruction — including user-customized ones in Settings — and subtracts them from the LLM's prompt **and** negative prompt in both converters (Generate, From Image), while never subtracting a tag the user actually requested. Covers custom instructions verbatim, weight variations of a leaked tag, and short generic placeholders unconditionally.
- **History → Remix now updates an already-open From Image page**: picking an image in the History footer panel and pressing Remix only worked when the Remix page wasn't already mounted — the page loaded the active session item solely in `onMounted`, so nothing reloaded it and the image silently "landed" in Generate instead (GeneratePage restores the active item on its own mount). A new `session:selected` event is now emitted only on explicit user selection (`SetActiveSessionItem`, `SetLastImage` — including File Browser "Send to Remix"); the From Image page listens and reloads with a latest-wins guard against out-of-order responses. Generation-time `session:active`/`session:added` remain non-auto-loading so a running edit/mask is never clobbered. Also fixed a pre-existing `ReferenceError` on leaving the page: Wails `EventsOn` off-functions were declared inside `onMounted` but called from `onUnmounted` (different scope), leaking listeners on every visit (same latent bug still present in `ExportPage.vue`).

## [0.7.9] — 2026-09-15

### Changed
- **Analyze modes reworked (Remix / From Image)**: the Quick/Deep toggle was cosmetic — both UI modes called the same `AnalyzeImage` with identical behavior (chain vs single decided by a global setting, chain on by default). Now the mode is a real parameter: **Tags** (single vision call → SD tags, fixed prompt that asks for tags immediately instead of "describe then convert") and **Text** (former Deep, single call → literary description of the photo in Russian, editable prompt in Settings). The 4-step analysis chain (`analyze_use_chain`, `analyze_chain_1..4` settings, `analyze:step` events) is removed — it was the main source of repeated garbage output. The field is now an "instruction" of any format (tags or prose); on generate it still flows through the existing LLM converter (`generateLLMPromptFromTags`) which handles both.
- **Analyze anti-repeat hardening**: vision requests now send `frequency_penalty: 0.3` / `presence_penalty: 0.2`, and tag output is globally deduplicated (`promptutil.DedupeTags`, weight- and case-aware). Verified against a live degenerate loop from qwen2.5vl (103 tags → 38 after dedup, loop tags collapse to one each). Root cause of repeats was the vision model looping on tag generation, not only the chain.
- **Preset style-tag leak fix**: the LLM converter copied STYLE REFERENCE tags from the preset into the result despite prompt rules (verified live: "gold accents, vintage illustration style..." from a Dark Custom preset leaked into a winter-city scene). Converter output is now filtered with `promptutil.RemoveTags(result, preset.Prompt)` — preset style tags are subtracted before generation.

### Added
- **PHP-chat wallpaper preset pack** (`data/presets/php-chat.json`): 22 presets reconstructing the php-community chat wallpaper run (2026-09-08 session) as reusable presets — 7 SD themes × DPM++ 2M Karras: minimal silk-ribbon gradients (6 colors, zavychromaxl_v100, 28 steps / CFG 6), cyberpunk night city (6 scenes), programmer desk, space, mountains, macro (juggernautXL_ragnarokBy, 30 / 6) and retrowave synthwave (ghostmix_v20Bakedvae, 28 / 7). Shared negative `text, watermark, signature, letters, people, jpeg artifacts, low quality, oversaturated`. Model names match Forge `model_name` (no `.safetensors`) so import validation passes clean. New preset type `php-chat` (auto-created on import and by `SeedBundled` on fresh DB). The terminal theme from the original run is intentionally not included — it was a native PIL render of real PHP code, not reproducible via SD presets. Original DAT x4 upscale/phone-crop post-processing is out of preset scope; use the Upscale button after generation.

## [0.7.8] — 2026-07-20

### Added
- **Remix (From Image) batch generation**: the From Image page now generates multiple variants in one run, matching Generate's batch UX. A count input (1–100) sits next to the Generate button; on submit the page enqueues N jobs through the existing `EnqueueFromImage` queue, so all N variants land in the active session as separate items with their own progress. No backend change — count is purely client-side and the queue already produces one session item per job. For inpaint/remove modes the mask is captured once before the loop so every variant uses the identical mask. Interrupt behavior is unchanged (aborts only the current job, like Generate); the batch count is persisted in settings (`fi_count`).

## [0.7.7] — 2026-06-29

### Fixed
- **"My images" → Remix landed on Generate page stuck in Remix view**: the File Browser's "Send to Remix" navigated to `{ page: 'generate', tab: 'from-image' }`, which both opened the wrong page (Generate, not Remix) and left `generateTab` pinned to `from-image`. Since `UnifiedGeneratePage` has no tab-switch UI and initializes its active tab from `initialTab` once, the Generate page stayed on the from-image/Remix sub-view forever — switching Generate/Remix in the sidebar always showed Remix. Now it navigates to `{ page: 'remix' }`, consistent with the session panel's Remix and the sidebar link (`FileBrowserPage.sendToFromImage()`). The image still loads via `setLastImage → AddToSession → SetActiveItem`, picked up by the Remix page's `useLastImage()`.
- **Remix (From Image) → inpaint with Workflow gave a brand-new image**: in compound/workflow mode the first step dispatched `txt2img` for any `mode != "img2img"`, so `inpaint` silently fell through to txt2img — the init image and mask were ignored and SD generated an unrelated image from the prompt. Now the compound first step runs `img2img` with the mask for `inpaint` mode (`runFromImageCompoundFirstStep`), matching the single-preset inpaint path.

## [0.7.6] — 2026-06-14

### Fixed
- **Compare By Image → txt2img instead of img2img**: a stale prompt persisted in settings (`test_prompt`) leaked into the hidden prompt field on the Image tab, so `analyzeImage` was skipped. Generation then ran img2img on the unrelated stale text instead of the pasted image — results looked like txt2img from a description. Reproduced only when `test_prompt` was non-empty. Now always analyzes the actual pasted image when an init image is present (`TestPage.generate()`).

### Removed
- Dead code cleanup: orphaned `internal/api/` package (`handler.go` + `handler_test.go`, ~44 KB — HTTP server extracted to sd-ai-server, zero imports), `enqueueCompare()` in `TestPage.vue` (never called), `llm.DefaultURL`, `queue.decodeBase64Image`, `generation.saveLastImage` + its test (superseded by session storage). Verified with `go build`, `go vet`, `go test ./...`, and `deadcode` (zero unreachable funcs remaining).

## [0.7.5] — 2026-06-06

### Changed
- Compare page preserves state across navigation (v-show instead of v-if)
- Compare preset list refreshes on navigation to the page
- Compare allows generating without text prompt (presets have their own prompts)
- Added preset type filter in Compare mode
- Removed duplicate Workflow button in Compare

### Fixed
- **Critical: Wails EventsOff global bug** — replaced all `EventsOff()` calls with cancel functions from `EventsOn()` across 8 files. `EventsOff('name')` was killing ALL subscribers for that event globally, causing History to stop updating and Compare to lose results
- History panel now auto-scrolls to show new images when already open
- Session select updates correctly when creating a new group
- `selectAll()` in Compare respects active preset type filter

## [0.7.4] — 2026-06-04

### Changed
- Replaced `any` types with concrete types (`string`, `json.RawMessage`) in `GenerateImageResult`
- Extracted `prepareSDContext` and `doHiresFallback` helpers — eliminates 8+ duplicated SD call patterns
- Extracted `processGeneration` in queue processor — 3 identical methods reduced to thin wrappers
- Extracted compound generation helpers — `GenerateCompoundImage` 220→90 lines, `generateFromImageCompound` 180→92 lines
- Extracted clipboard logic into `internal/clipboard/` package
- Extracted `usePresets` and `useKidsMode` frontend composables — shared preset state between pages
- Replaced magic numbers with named constants (`MaxImageBase64Len`, `MaxImageBytes`, `MAX_IMAGE_SIZE`)
- Removed dead parameter from `resolveHires`

### Fixed
- Added `io.LimitReader` to all HTTP clients (sd, llm, serverclient, rembg) to prevent OOM
- Added path traversal protection in serverclient API calls
- Added URL scheme validation (`http`/`https` only) in SD and LLM `SetURL` methods

## [0.7.3] — 2026-06-04

### Changed
- Hires profile now applied on the last workflow step instead of the first, preventing upscale degradation through intermediate img2img passes
- History panel auto-scrolls to bottom on open
- Preset lists auto-refresh on tab navigation, removed Refresh buttons from Generate and Remix pages

### Added
- Delete button in history lightbox with confirmation (button + Del key)

## [0.7.1] — 2026-06-01

### Changed
- LLM system prompt rewritten: detail preservation rule forces extraction of every visual element from user description
- Default `maxTokens` increased 256→512 for longer, more complete SD prompts
- Response length instruction changed from restrictive limit to "include ALL elements"
- History lightbox with metadata bar and action buttons

### Fixed
- LLM losing details (moss, confident, face lighting, cracks) when converting descriptions to SD tags

## [0.7.0] — 2026-05-30

### Added
- Queue system with retry, pause, persistent storage
- Multi-scene compositor with background removal
- Compare page (side-by-side generation)
- Smart Remove (LLM vision + inpaint)
- Server mode integration (serverclient, discovery)
- Kids mode filtering
- Session management enhancements
- Resolution and hires profiles
- Pipeline export/import
- Model catalog

### Changed
- User-scene-first prompt generation — LLM generates only user scene tags with SD weights, preset style added after BREAK separator for better CLIP attention
- User prompt placed first in final SD prompt (gets full first-chunk attention), preset quality/style after BREAK
- System prompt labels changed to STYLE REFERENCE / USER SCENE to prevent LLM from duplicating preset tags
- Server component extracted to separate repository: [sd-ai-server](https://github.com/Zazza/sd-ai-server)

### Fixed
- Various bug fixes and UI improvements

## [0.6.2] — 2026-05-20

### Added
- Pipeline (compound preset) export/import — select pipelines, export to JSON, import from file with preset auto-creation
- Resolution and hires profile selectors on Batch and Compare pages

### Fixed
- Hires upscale applied only on last compound pipeline step (was applying on every step)
- Resolution and hires profile selections persisted in Generate, Batch, and Test pages on unmount
- Resolution/hires keys added to AllowedSettings whitelist and saved via watch
- Hires upscaler defaults improved, neural pre-upscale added for Forge fallback
- VAE reset to Automatic when preset has no VAE set

## [0.6.1] — 2026-05-13

### Fixed
- Remove tool: fixed inpaint parameters — use Original fill + denoising 0.6 for proper object removal instead of generating new content

### Changed
- Removed `width`/`height` from pipeline (compound preset) steps — resolution now comes from the resolution selector on generation page, same as regular presets
- DB migration v18: drops `width`/`height` columns from `compound_preset_steps` table

### Added
- Duplicate button for pipelines (Presets → Pipelines)

## [0.6.0] — 2026-05-13

### Added
- Resolution profiles — independent resolution entities stored in DB (`resolutions` table), selectable via `ResolutionSelector` component
- Hires profiles — independent hires fix entities stored in DB (`hires_profiles` table), selectable via `HiresProfileSelector` component
- Resolution and hires profile persistence — saved in settings (`gen_resolution_id`, `gen_hires_profile_id`), restored on app restart and page transitions
- Default resolution: first from list when no saved value; hires disabled by default
- Pipeline JSON import/export — sample pipeline files in `data/pipelines/` (anime-realistic, photo-stylize, style-transfer)

### Changed
- Removed `width`/`height` fields from presets — resolution now managed via independent resolution profiles
- Removed `hires_*` fields from preset form — hires now managed via independent hires profiles
- Generation flow uses selected resolution and hires profile instead of preset-embedded values

## [0.5.6] — 2026-05-11

### Added
- Manual hires upscale fallback: when Forge API hires fix fails, generates at base resolution then upscales via img2img with denoising — transparent to UI, no UI changes needed
- `HiresFixManual` flag in generation result (UI can show "manual upscale" badge)

### Changed
- Hires fix fallback now attempts manual upscale before falling back to base image

## [0.5.5] — 2026-05-08

### Fixed
- Remove tool: LLM vision analysis now produces background/surface tags instead of full scene descriptions
- Added prose detection — scene descriptions are rejected, fallback prompt used instead
- Lowered denoising strength (0.75 → 0.6) for cleaner object removal with fewer artifacts
- Post-processing with `CleanTags` + `StripJunk` to strip non-tag output from LLM

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
