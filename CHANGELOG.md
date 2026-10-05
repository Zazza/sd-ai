# Changelog

All notable changes to SD Studio are documented here.

## [0.9.0] — 2026-10-05

### Added
- **Preset overrides — override hidden preset parameters per generation**: the Generate tab's "More options" and From Image's parameter block gained a collapsible "Override preset parameters" panel (plain-preset mode only, hidden in kids mode): Model / Sampler / Scheduler selects with a leading "From preset" option, Steps / CFG / Clip skip numeric inputs (empty = from preset) and an "Ignore preset LoRAs" checkbox (sends `loras:"[]"` — preset LoRAs almost never survive a model swap). Overrides are one-shot: never persisted, reset on preset/mode change. Backend: new `generation.PresetOverrides` (nullable pointers, empty/zero values are no-ops, steps capped at 150) applied via `ApplyPresetOverrides` on a copy of the preset right after load in three entry points — `GenerateImage` (txt2img + queue), `GenerateFromImage` (remix/img2img/inpaint) and `GenerateSDPrompt` (so switching to a Flux-class model re-routes the LLM conversion to the prose instruction and Memory Guard sees the overridden model). Queue jobs carry overrides inside the existing params JSON (old pending jobs behave as before); MCP tools inherit the capability; compound pipelines intentionally don't (per-step presets carry their own settings).
- **Explicit Seed field on Generate and From Image**: "More options" on Generate and the parameter block on From Image gained a Seed input — empty means random (stated right in the placeholder and hint), a number pins the generation for reproduction or clean A/B prompt testing. A batch with a fixed seed fans out seed, seed+1, seed+2… (same scheme as the MCP best-of tool) instead of producing clones. Backend: `GenerateFromImageParams` gained a nullable `Seed` applied in both the remix (img2img/inpaint) and the txt2img-fallback request builders of the preset branch (compounds and remove-object keep their own seed semantics); the Generate path already supported it. The value is snapshotted before the batch loop like every other parameter.

### Fixed
- **«preset is required» when changing the preset after enqueuing a batch**: the Generate batch loop re-readed the live UI state (preset/mode/extra prompts/overrides) between enqueue awaits — changing the preset, type filter or mode mid-loop sent the remaining jobs with different or zero parameters, and the poisoned job (preset_id=0) only failed when the queue reached it, retrying 3× before pausing. All values are now snapshotted into constants before the loop (overrides — before the LLM conversion, so the prompt and the job parameters can't diverge either). The queue itself now validates job parameters on enqueue (fail-fast with the same error texts; the remove-object path legitimately runs without a preset and is exempt), and `GenerateCompoundImage` gained the same upfront guard. From Image was already safe (params built once before the loop).

### Changed
- **Bundled presets reworked around the "preset = style, not scene" concept**: the 25 bundled JSON files (313 presets, most of them per-model duplicates of the same category) are replaced by 24 categories × prompt families — **(SD)** tag-style, **(Pony)** anime-tag style with score_9 prefixes, **(Flux)** prose and **(Z-Image)** prose (24/8 steps, both CFG 1) — ~90 presets total. Prompts now describe only the style/medium/lighting; concrete subjects, colors and scenes are left to the user's description (LLM conversion), matching the sd-mcp semantics. Each family binds a representative model (realisticVision V60 / dreamshaper / riMix / leosams for 3D / flux / z-image) — the new override panel covers everything else. `wallpapers.json` merged into `wallpaper.json` with explicit Desktop/Phone variants; working databases pick the new set up automatically on next launch (bundled seeding only runs when no bundled presets exist). The `(Z-Image)` family requires a WebUI fork with Z-Image support — stock Forge does not ship that architecture (see `docs/setup` for a drop-in fork); every other family works on any supported backend.
- **MCP server `sd-mcp` for AI agents**: a separate stdio binary (`make mcp` → `build/bin/sd-mcp`) exposing SD Studio to MCP clients over line-delimited JSON-RPC (MCP 2024-11-05). 14 tools: `status`, `presets`/`preset_get`/`preset_create`/`preset_delete` (confirm-gated), `compound_list`/`compound_get`, `generate` (preset+LLM description conversion, or direct prompt+model via the Test path), `generate_best_of` (single LLM conversion, fan of seeds base+0..n-1 at full size, per-item errors don't abort, contact sheet), `analyze` (tags/describe), `upscale` (x2 img2img from a sidecar `.json`), `from_image` (img2img/inpaint), `sd_meta`, `interrupt`. Artifacts land as PNG + sidecar `.json` in `SD_MCP_OUT` (default `~/sd-mcp-out`). The binary reuses the desktop internals (generation.Service with no-op event/session adapters, settings restored from the shared DB, Memory Guard intact); DB path resolves via `-db` flag → `SD_MCP_DB` env → the standard config path, and `preset.Open` now sets `journal_mode(WAL)` + `busy_timeout(5000)` so MCP and the desktop app can work on the same DB concurrently. Desktop behavior is unchanged: the settings-restore block moved from `main.go` into `settings.Restore`, compound-preset enrichment moved into `preset.DB` (`ListCompoundPresetsFull`/`GetCompoundPresetFull`), `sanitizeFilename` moved to `filebrowser.SanitizeFilename`, and `GenerateImageParams` gained optional `Seed`/`FullSize` (zero values keep the old behavior). Hardened after logic+security review: requests are dispatched concurrently (goroutine per request, serialized writes) so `interrupt` and `ping` work while a generation is running — all other tools serialize on a mutex; `generate_best_of` skips the LLM conversion when no description/negative is given (previously duplicated the preset prompt); input lines are capped at 64 MiB with a JSON-RPC error instead of crashing the server; artifacts are written with `0600`/dir `0700`; LLM response bodies are no longer logged and SD error messages truncate the request body to 512 bytes. `preset_create` is confirm-gated like `preset_delete`: without `confirm=true` the tool refuses and suggests searching the existing library first — agents can no longer silently add presets. The `presets` search now matches per-word (AND across space-separated words in name/tags) instead of a whole-phrase substring — long translated queries no longer return empty results; an invalid `type_id` in `preset_create` is now an error instead of being silently dropped. New tools: `styles` (preset types with counts and example presets — the style menu an agent shows before generating) and `compound_generate` (run a whole pipeline: the scene description is LLM-converted in the first step's style and carried through the chain). Tool count: 16.

## [0.8.0] — 2026-10-02

### Added
- **UI language switcher (RU/EN)**: a language button next to the theme toggle in the sidebar switches the whole UI between English and Russian instantly. The Russian dictionary (`i18n/ru.js`) covers every UI string (503 keys, verified 1:1 with English); technical SD terms (img2img, LoRA, sampler parameters) stay untranslated. The choice persists in localStorage (`ui_locale`), defaulting to Russian on first launch. No vue-i18n dependency — a reactive locale ref drives the existing `t()` used in all templates. The dead duplicate `t()` export in `en.js` was removed, and the hardcoded ` [inverted]` mask suffix became a translatable key.
- **Model-aware prompt conversion — prose for smart models, tags for simple ones**: the LLM converter was tuned for SDXL ("SD understands only tags") and mangled scenes for Flux-class models whose T5 encoder follows natural language — an A/B test showed prose keeping 10/10 scene elements vs ~3/10 for weighted tag soup. Both converters (Generate, From Image) now pick the instruction by the preset's model: models matching the new `prose_models` CSV setting (default `flux,z-image,qwen-image,chroma,hunyuan`) get a prose instruction — one connected English paragraph preserving all spatial/logical relations, no weights, appearance variation woven in — editable in Settings as `sd_prompt_instruction_prose`; everything else keeps the existing tag instruction unchanged. Truncation limit for positives is mode-aware (2000 chars prose vs 1000 tags), the Test-compound page converts per-pipeline (mixed SDXL+Flux selections keep their own formats), and the inpaint tag append becomes a sentence in prose mode.
- **Memory Guard — conditional LLM unload before heavy generations**: loading a heavy checkpoint (flux & co) while the ollama LLM is warm (~10 GB resident) thrashes the 32 GB server. Before every model switch (choke point: `prepareSDContext` covering all generation flows, plus Test-generate model mode) a new `MemoryGuard` checks free VRAM/RAM via Forge `GET /sdapi/v1/memory` and resident ollama models via `GET /api/ps`; when there is a deficit that unloading would actually fix, it unloads the LLM (`keep_alive:0`) and waits up to 15 s for it to leave memory. Fail-open by design: any error or non-ollama backend → silent no-op, generation is never blocked, and light models (SDXL) make zero HTTP requests. New settings: `heavy_models` (CSV name substrings, default `flux,z-image,qwen-image,chroma,hunyuan`) and `llm_auto_unload` (default `true`).

### Fixed
- **No live preview/progress for images 2..N in batch generation**: after the first image of a batch arrived, both Generate and Remix (From Image) switched to the finished-image view — the remaining jobs generated silently (SD live preview events were flowing but not rendered). Generate's `session:added` handler ended the "generating" UI state after every job instead of only at batch end; Remix had no batch overlay in its template at all. Both pages now keep the batch state until the queue drains, showing the last completed image dimmed with a live-preview overlay, per-image progress ("n / N", %, ETA) and an Interrupt button. Also fixed along the way: Interrupt during a batch now cancels the whole remaining batch (was: only the current image, the next job started right after); a job paused permanently by the queue retry policy (max retries reached) now releases the batch overlay (was: stuck until restart — `queue:paused` wasn't handled); Generate's queue-restore on mount only picks up its own job types (txt2img/compound, was also claiming from_image jobs); Generate's event listeners are now properly unregistered on unmount (onUnmounted was registered inside an async onMounted after awaits, so tab switches leaked listeners).
- **Stale Saved-idea negative leaking into hand-written scenes**: the idea-sourced negative stayed in the field after the user detached from the idea by typing a new description (and could also be restored from the persisted session settings with a dead binding), silently riding along into the LLM conversion and generation. Detaching now clears the negative when it still equals the idea's (a hand-written negative is kept), and the session restore no longer resurrects the persisted negative when the last session's idea binding did not survive (description had diverged).
- **Settings save was silently dropping prompt/analyze instructions**: `sd_prompt_instruction`, `analyze_system_prompt`, `analyze_prompt` and `analyze_describe_prompt` were never added to the settings whitelist, so editing them in Settings was a no-op that reverted to defaults after restart — present since the whitelist gate was introduced. All four keys are now allowed through.
- **Editing a Saved idea no longer leaves the LLM prompt stale**: previously the update handler refreshed only the saved list — the live description field kept the old text, the dirty-flag stayed false, and the next generation silently reused the LLM prompt built from the previous wording. The page now tracks the currently loaded idea (persisted as `gen_desc_id` alongside the other Generate-page settings and restored on remount, including tab switches): editing that idea in the modal syncs the new text (and the negative when it was loaded from the idea and not hand-edited) into the live fields, and the watcher chain re-marks the prompt dirty so it regenerates. Hand edits detach the binding; deleting the bound idea clears it; saving a new description from the field binds it too. (The suspected cold-LLM/unload path was ruled out by code: a failed prompt generation aborts with an error instead of generating a wrong image.)
- **Re-import duplicating presets**: importing the same preset/pipeline file again unconditionally created new rows — 3 imports of one chain produced 9 exact copies in the production DB. Both import paths (`ImportPresets` via `ImportItems`, and pipeline steps in `ImportCompoundItems`) now look up an exact match first via the new `preset.DB.FindExactPreset` (name, preset_type, prompt, negative_prompt, sampler, schedule_type, steps, cfg_scale, model_name, loras — all `=` in SQL): a match is reused (`[import] reusing existing preset …` in the log; pipeline steps point at the existing preset ID) instead of inserting a copy; non-matching presets are still created. Plain import returns the existing presets in the result so the UI shows them as imported.

### Removed
- **Multi-Scene (Scene Editor / Multi-Pass scene generator)**: the feature is removed in its entirety — the Scene Editor page and navigation, the `DecomposeScene`/`GenerateMultiPass` Wails bindings, `internal/compositor` with the pass-by-pass generation and compositing, the `saved_scenes` table CRUD, and the `multipass:progress` event. DB migration v26 drops `saved_scenes` (`DROP TABLE`) — **saved scenes are lost irreversibly** on the first launch after the update. Single-image, batch, pipeline and From Image generation are untouched.
- **rembg infrastructure**: with no consumer left after the Multi-Scene removal, the rembg support is gone too — the `internal/rembg` client, the rembg flag in the service status (`CheckRembg`), the `rembg_url` setting (deleted from the DB by migration v26) and the Rembg section in Settings. The server-side rembg installation/proxying remains in the [sd-ai-server](https://github.com/Zazza/sd-ai-server) repository — cleaning it up there is a follow-up for that repo.

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
