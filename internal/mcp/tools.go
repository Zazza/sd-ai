package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-sd/internal/filebrowser"
	"go-sd/internal/generation"
	"go-sd/internal/preset"
)

func RegisterTools(s *Server) {
	s.Register(Tool{
		Name:        "status",
		Description: "Статус SD Studio: доступность SD/LLM, текущая модель SD, прогресс генерации, kids mode, путь к БД и папке артефактов.",
		InputSchema: props(nil),
		Handler: func(s *Server, _ map[string]any) (string, error) {
			res := map[string]any{
				"db":        s.deps.Cfg.DBPath,
				"out_dir":   s.deps.OutDir,
				"kids_mode": s.deps.Kids.IsActive(),
			}
			if err := s.deps.SD.HealthCheck(); err != nil {
				res["sd"] = map[string]any{"available": false, "error": err.Error()}
			} else {
				m := map[string]any{"available": true, "url": s.deps.Cfg.SDUrl}
				if opts, err := s.deps.SD.GetOptions(); err == nil {
					if v, ok := opts["sd_model_checkpoint"].(string); ok {
						m["model"] = v
					}
				}
				if prog, err := s.deps.SD.GetProgress(); err == nil {
					m["progress"] = map[string]any{
						"progress": prog.Progress,
						"eta":      prog.ETARelative,
						"job":      prog.State.Job,
						"steps":    prog.State.Sampling.Steps,
					}
				}
				res["sd"] = m
			}
			if err := s.deps.LLM.HealthCheck(); err != nil {
				res["llm"] = map[string]any{"available": false, "error": err.Error()}
			} else {
				res["llm"] = map[string]any{"available": true, "url": s.deps.Cfg.LLMUrl, "backend": s.deps.LLM.Backend()}
			}
			return toJSON(res), nil
		},
	})

	s.Register(Tool{
		Name:        "presets",
		Description: "Список пресетов компактными строками (id, имя, тип, теги, модель). Фильтры: query — подстрока в имени или тегах, tag — точный тег, type — тип пресета, limit (по умолчанию 50).",
		InputSchema: props(map[string]any{
			"query": prop("слова через пробел; каждое должно входить в имя или теги (AND); короткие запросы точнее длинных фраз", "string"),
			"tag":   prop("точный тег из tags", "string"),
			"type":  prop("тип пресета (preset_type)", "string"),
			"limit": prop("максимум строк (по умолчанию 50)", "integer"),
		}),
		Handler: func(s *Server, args map[string]any) (string, error) {
			items, err := s.deps.DB.List()
			if err != nil {
				return "", err
			}
			queryWords := strings.Fields(strings.ToLower(argString(args, "query")))
			tag := argString(args, "tag")
			ptype := argString(args, "type")
			limit := int(argInt(args, "limit"))
			if limit <= 0 {
				limit = 50
			}
			lines := make([]string, 0, 16)
			for _, p := range items {
				if len(queryWords) > 0 {
					lowerName := strings.ToLower(p.Name)
					lowerTags := strings.ToLower(p.Tags)
					matched := true
					for _, w := range queryWords {
						if !strings.Contains(lowerName, w) && !strings.Contains(lowerTags, w) {
							matched = false
							break
						}
					}
					if !matched {
						continue
					}
				}
				if tag != "" && !hasTag(p.Tags, tag) {
					continue
				}
				if ptype != "" && !strings.EqualFold(p.PresetType, ptype) {
					continue
				}
				line := fmt.Sprintf("%d  %s", p.ID, p.Name)
				if p.PresetType != "" {
					line += "  [" + p.PresetType + "]"
				}
				if p.Tags != "" {
					line += "  tags: " + p.Tags
				}
				if p.ModelName != "" {
					line += "  model: " + p.ModelName
				}
				lines = append(lines, line)
				if len(lines) >= limit {
					break
				}
			}
			if len(lines) == 0 {
				return "ничего не найдено", nil
			}
			return fmt.Sprintf("найдено %d (всего пресетов: %d)\n%s", len(lines), len(items), strings.Join(lines, "\n")), nil
		},
	})

	s.Register(Tool{
		Name:        "styles",
		Description: "Список стилей (типов пресетов) со счётчиком пресетов и примерами. Пресет = СТИЛЬ (реализм, аниме, мультфильм…), не сцена. Используй, чтобы показать пользователю выбор стилей перед генерацией.",
		InputSchema: props(nil),
		Handler: func(s *Server, _ map[string]any) (string, error) {
			types, err := s.deps.DB.ListPresetTypes()
			if err != nil {
				return "", err
			}
			items, err := s.deps.DB.List()
			if err != nil {
				return "", err
			}
			counts := map[int64]int{}
			examples := map[int64][]string{}
			untyped := 0
			for _, pr := range items {
				if pr.TypeID == nil {
					untyped++
					continue
				}
				counts[*pr.TypeID]++
				if len(examples[*pr.TypeID]) < 3 {
					examples[*pr.TypeID] = append(examples[*pr.TypeID], pr.Name)
				}
			}
			lines := make([]string, 0, len(types)+1)
			for _, t := range types {
				line := fmt.Sprintf("%d  %s (%d пресетов)", t.ID, t.Name, counts[t.ID])
				if t.Description != "" {
					line += " — " + t.Description
				}
				if ex := examples[t.ID]; len(ex) > 0 {
					line += "  примеры: " + strings.Join(ex, ", ")
				}
				lines = append(lines, line)
			}
			if untyped > 0 {
				lines = append(lines, fmt.Sprintf("без типа: %d пресетов", untyped))
			}
			if len(lines) == 0 {
				return "типов пресетов нет", nil
			}
			return strings.Join(lines, "\n"), nil
		},
	})

	s.Register(Tool{
		Name:        "preset_get",
		Description: "Полное тело пресета по id: prompt, negative_prompt, sampler, steps, cfg_scale, модель, теги, сид.",
		InputSchema: props(map[string]any{
			"id": prop("id пресета", "integer"),
		}, "id"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			id := argInt(args, "id")
			if id <= 0 {
				return "", fmt.Errorf("id обязателен")
			}
			p, err := s.deps.DB.Get(id)
			if err != nil {
				return "", fmt.Errorf("preset not found: %w", err)
			}
			return toJSON(p), nil
		},
	})

	s.Register(Tool{
		Name:        "preset_create",
		Description: "Создать пресет, возвращает id. Пишущий инструмент: требует confirm=true — сначала покажи пользователю имя и промпт и получи согласие. Сначала убедись, что подходящего пресета нет (presets/compound_list). Дефолты: sampler=Euler a, steps=20, cfg_scale=7.0.",
		InputSchema: props(map[string]any{
			"name":            prop("имя пресета", "string"),
			"prompt":          prop("положительный промпт (SD-теги или prose — по модели)", "string"),
			"negative_prompt": prop("негативный промпт", "string"),
			"model_name":      prop("модель SD (title из sd_meta)", "string"),
			"sampler":         prop("сэмплер (по умолчанию Euler a)", "string"),
			"steps":           prop("шаги (по умолчанию 20)", "integer"),
			"cfg_scale":       prop("CFG (по умолчанию 7.0)", "number"),
			"tags":            prop("теги для поиска", "array", map[string]any{"items": map[string]any{"type": "string"}}),
			"type_id":         prop("id типа пресета (preset_types из БД)", "integer"),
			"confirm":         prop("явное подтверждение пользователя на создание", "boolean"),
		}, "name", "prompt", "confirm"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			name := strings.TrimSpace(argString(args, "name"))
			prompt := strings.TrimSpace(argString(args, "prompt"))
			if name == "" || prompt == "" {
				return "", fmt.Errorf("name и prompt обязательны")
			}
			if !argBool(args, "confirm") {
				preview := prompt
				if r := []rune(preview); len(r) > 60 {
					preview = string(r[:60]) + "…"
				}
				return "", fmt.Errorf("подтвердите создание пресета %q (промпт: %s): получите согласие пользователя и повторите с confirm=true. Возможно, подходящий пресет уже есть — поищите через presets", name, preview)
			}
			p := preset.Preset{
				Name:           name,
				Prompt:         prompt,
				NegativePrompt: argString(args, "negative_prompt"),
				ModelName:      argString(args, "model_name"),
				Sampler:        argString(args, "sampler"),
				Steps:          int(argInt(args, "steps")),
				CfgScale:       argFloat(args, "cfg_scale"),
			}
			if p.Sampler == "" {
				p.Sampler = "Euler a"
			}
			if p.Steps <= 0 {
				p.Steps = 20
			}
			if p.CfgScale <= 0 {
				p.CfgScale = 7.0
			}
			if tags := argStringSlice(args, "tags"); len(tags) > 0 {
				p.Tags = strings.Join(tags, ", ")
			}
			if tid := argInt(args, "type_id"); tid > 0 {
				pt, err := s.deps.DB.GetPresetType(tid)
				if err != nil {
					return "", fmt.Errorf("type_id %d not found", tid)
				}
				p.TypeID = &tid
				p.PresetType = pt.Name
			}
			if err := s.deps.DB.Create(&p); err != nil {
				return "", err
			}
			return fmt.Sprintf("создан пресет id=%d (%s)", p.ID, p.Name), nil
		},
	})

	s.Register(Tool{
		Name:        "preset_delete",
		Description: "Удалить пресет по id. Деструктивное: требует confirm=true (спроси пользователя).",
		InputSchema: props(map[string]any{
			"id":      prop("id пресета", "integer"),
			"confirm": prop("явное подтверждение пользователя", "boolean"),
		}, "id", "confirm"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			id := argInt(args, "id")
			if id <= 0 {
				return "", fmt.Errorf("id обязателен")
			}
			if !argBool(args, "confirm") {
				name := ""
				if p, err := s.deps.DB.Get(id); err == nil {
					name = p.Name
				}
				return "", fmt.Errorf("подтвердите delete: preset %d %s", id, name)
			}
			if err := s.deps.DB.Delete(id); err != nil {
				return "", err
			}
			return fmt.Sprintf("пресет %d удалён", id), nil
		},
	})

	s.Register(Tool{
		Name:        "compound_list",
		Description: "Список составных пресетов (цепочек): id, имя, шаги с именами пресетов.",
		InputSchema: props(nil),
		Handler: func(s *Server, _ map[string]any) (string, error) {
			items, err := s.deps.DB.ListCompoundPresetsFull()
			if err != nil {
				return "", err
			}
			if len(items) == 0 {
				return "составных пресетов нет", nil
			}
			lines := make([]string, 0, len(items))
			for _, cp := range items {
				stepParts := make([]string, 0, len(cp.Steps))
				for _, st := range cp.Steps {
					name := fmt.Sprintf("preset #%d", st.PresetID)
					if st.Preset != nil {
						name = st.Preset.Name
					}
					stepParts = append(stepParts, fmt.Sprintf("%d:%s", st.StepOrder, name))
				}
				lines = append(lines, fmt.Sprintf("%d  %s  [%s]", cp.ID, cp.Name, strings.Join(stepParts, " → ")))
			}
			return strings.Join(lines, "\n"), nil
		},
	})

	s.Register(Tool{
		Name:        "compound_get",
		Description: "Составной пресет по id: полные шаги с телами пресетов и denoising_strength.",
		InputSchema: props(map[string]any{
			"id": prop("id составного пресета", "integer"),
		}, "id"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			id := argInt(args, "id")
			if id <= 0 {
				return "", fmt.Errorf("id обязателен")
			}
			cp, err := s.deps.DB.GetCompoundPresetFull(id)
			if err != nil {
				return "", fmt.Errorf("compound preset not found: %w", err)
			}
			return toJSON(cp), nil
		},
	})

	s.Register(Tool{
		Name:        "compound_generate",
		Description: "Прогнать пайплайн (составной пресет) целиком: description сцены конвертируется LLM по стилю первого шага и проводится через все шаги цепочки (txt2img → img2img → …). Медленнее одиночной генерации. Результат: PNG + sidecar в out_dir. Пайплайны — из compound_list.",
		InputSchema: props(map[string]any{
			"compound_preset_id": prop("id пайплайна из compound_list", "integer"),
			"description":        prop("сцена: что изобразить (стиль возьмётся из пресетов цепочки)", "string"),
			"negative":           prop("дополнительный негатив", "string"),
			"resolution_id":      prop("id разрешения", "integer"),
		}, "compound_preset_id"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			id := argInt(args, "compound_preset_id")
			if id <= 0 {
				return "", fmt.Errorf("compound_preset_id обязателен")
			}
			cp, err := s.deps.DB.GetCompoundPresetFull(id)
			if err != nil {
				return "", fmt.Errorf("compound preset not found: %w", err)
			}
			gp := generation.GenerateCompoundImageParams{
				CompoundPresetID:    id,
				ExtraPrompt:         argString(args, "description"),
				ExtraNegativePrompt: argString(args, "negative"),
			}
			if rid := optInt(args, "resolution_id"); rid != nil && *rid > 0 {
				gp.ResolutionID = rid
			}
			res, err := s.deps.Gen.GenerateCompoundImage(gp)
			if err != nil {
				return "", err
			}
			path, err := s.saveArtifact(cp.Name, "compound-"+slugify(cp.Name), res.Image, res.Info)
			if err != nil {
				return "", err
			}
			return toJSON(map[string]any{
				"path":   path,
				"prompt": res.EffectivePrompt,
				"steps":  len(cp.Steps),
			}), nil
		},
	})

	s.Register(Tool{
		Name: "generate",
		Description: "Сгенерировать изображение. Ветка 1: preset_id — стиль пресета, description (и negative) конвертируются LLM в SD-промпт, размер через resolution_id. " +
			"Ветка 2: prompt+model — прямой прогон без LLM (как Test), размер через width/height. " +
			"seed фиксирует сид. PNG + sidecar .json пишутся в out_dir; ответ: path, seed, prompt, is_preview. save=false — вернёт image_base64 вместо файла.",
		InputSchema: props(map[string]any{
			"preset_id":     prop("id пресета (ветка 1)", "integer"),
			"description":   prop("описание сцены для LLM-конвертации (ветка 1)", "string"),
			"negative":      prop("пользовательский негатив (ветка 1)", "string"),
			"prompt":        prop("готовый SD-промпт (ветка 2)", "string"),
			"model":         prop("модель SD title (ветка 2)", "string"),
			"seed":          prop("фиксированный сид", "integer"),
			"resolution_id": prop("id разрешения (ветка 1)", "integer"),
			"width":         prop("ширина (ветка 2, по умолчанию 512)", "integer"),
			"height":        prop("высота (ветка 2, по умолчанию 512)", "integer"),
			"sampler":       prop("сэмплер (ветка 2)", "string"),
			"steps":         prop("шаги (ветка 2)", "integer"),
			"cfg_scale":     prop("CFG (ветка 2)", "number"),
			"save":          prop("сохранить PNG+sidecar в out_dir (по умолчанию true)", "boolean"),
		}),
		Handler: func(s *Server, args map[string]any) (string, error) {
			save := optBool(args, "save", true)
			presetID := argInt(args, "preset_id")
			if presetID > 0 {
				p, err := s.deps.DB.Get(presetID)
				if err != nil {
					return "", fmt.Errorf("preset not found: %w", err)
				}
				extraPrompt, extraNeg := "", ""
				description := argString(args, "description")
				negative := argString(args, "negative")
				if description != "" || negative != "" {
					pr, err := s.deps.Gen.GenerateSDPrompt(generation.GenerateSDPromptParams{
						PresetID:    presetID,
						Description: description,
						Negative:    negative,
					})
					if err != nil {
						return "", err
					}
					extraPrompt, extraNeg = pr.Prompt, pr.NegativePrompt
				}
				gp := generation.GenerateImageParams{
					PresetID:            presetID,
					ExtraPrompt:         extraPrompt,
					ExtraNegativePrompt: extraNeg,
					Seed:                optInt(args, "seed"),
				}
				if rid := optInt(args, "resolution_id"); rid != nil && *rid > 0 {
					gp.ResolutionID = rid
				}
				res, err := s.deps.Gen.GenerateImage(gp)
				if err != nil {
					return "", err
				}
				out := map[string]any{
					"seed":            infoSeed(res.Info),
					"prompt":          res.EffectivePrompt,
					"negative_prompt": res.EffectiveNegativePrompt,
					"is_preview":      res.IsPreview,
				}
				if save {
					path, err := s.saveArtifact(p.Name, "gen-"+slugify(p.Name), res.Image, res.Info)
					if err != nil {
						return "", err
					}
					out["path"] = path
				} else {
					out["image_base64"] = res.Image
				}
				return toJSON(out), nil
			}

			prompt := argString(args, "prompt")
			model := argString(args, "model")
			if prompt == "" || model == "" {
				return "", fmt.Errorf("нужен preset_id или пара prompt+model")
			}
			tp := generation.TestGenerateParams{
				Mode:           "models",
				SelectedModels: []string{model},
				Prompt:         prompt,
				NegativePrompt: argString(args, "negative"),
				Sampler:        argString(args, "sampler"),
				Steps:          int(argInt(args, "steps")),
				CfgScale:       argFloat(args, "cfg_scale"),
				Width:          int(argInt(args, "width")),
				Height:         int(argInt(args, "height")),
				Seed:           optInt(args, "seed"),
			}
			if tp.Width <= 0 {
				tp.Width = 512
			}
			if tp.Height <= 0 {
				tp.Height = 512
			}
			if rid := optInt(args, "resolution_id"); rid != nil && *rid > 0 {
				tp.ResolutionID = rid
			}
			items, err := s.deps.Gen.TestGenerate(tp)
			if err != nil {
				return "", err
			}
			if len(items) == 0 || items[0].Error != "" {
				msg := "генерация не удалась"
				if len(items) > 0 {
					msg = items[0].Error
				}
				return "", fmt.Errorf("%s", msg)
			}
			item := items[0]
			steps := tp.Steps
			if steps <= 0 {
				steps = 20
			}
			cfg := tp.CfgScale
			if cfg <= 0 {
				cfg = 7
			}
			info, _ := json.Marshal(map[string]any{
				"prompt":          tp.Prompt,
				"negative_prompt": tp.NegativePrompt,
				"sampler_name":    item.Sampler,
				"scheduler":       item.ScheduleType,
				"seed":            item.Seed,
				"steps":           steps,
				"cfg_scale":       item.CfgScale,
				"width":           tp.Width,
				"height":          tp.Height,
				"clip_skip":       1,
				"sd_model_name":   item.ModelName,
			})
			out := map[string]any{
				"seed":       item.Seed,
				"prompt":     tp.Prompt,
				"model":      item.ModelName,
				"is_preview": false,
			}
			if save {
				path, err := s.saveArtifact(model, "gen-"+slugify(model), item.Image, info)
				if err != nil {
					return "", err
				}
				out["path"] = path
			} else {
				out["image_base64"] = item.Image
			}
			return toJSON(out), nil
		},
	})

	s.Register(Tool{
		Name: "generate_best_of",
		Description: "Веер best-of-N по пресету: description (опционально) конвертируется LLM один раз, затем n генераций с сидами base_seed+0..n-1 в полном размере (без preview-уменьшения). " +
			"Без description — веер чистого промпта пресета. Не используй пресеты с batch_size>1 (лишние картинки). " +
			"PNG + sidecar каждому, контактный лист bestof-{preset}-sheet.png. Ошибки отдельных картинок не роняют веер.",
		InputSchema: props(map[string]any{
			"preset_id":     prop("id пресета", "integer"),
			"description":   prop("описание сцены для LLM-конвертации", "string"),
			"negative":      prop("пользовательский негатив", "string"),
			"n":             prop("число картинок 1–12 (по умолчанию 4)", "integer"),
			"base_seed":     prop("базовый сид веера (по умолчанию случайный)", "integer"),
			"resolution_id": prop("id разрешения", "integer"),
		}, "preset_id"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			presetID := argInt(args, "preset_id")
			if presetID <= 0 {
				return "", fmt.Errorf("preset_id обязателен")
			}
			p, err := s.deps.DB.Get(presetID)
			if err != nil {
				return "", fmt.Errorf("preset not found: %w", err)
			}
			description := argString(args, "description")
			negative := argString(args, "negative")
			extraPrompt, extraNeg := "", ""
			if description != "" || negative != "" {
				pr, err := s.deps.Gen.GenerateSDPrompt(generation.GenerateSDPromptParams{
					PresetID:    presetID,
					Description: description,
					Negative:    negative,
				})
				if err != nil {
					return "", err
				}
				extraPrompt, extraNeg = pr.Prompt, pr.NegativePrompt
			}

			n := int(argInt(args, "n"))
			if n <= 0 {
				n = 4
			}
			if n > 12 {
				n = 12
			}
			base := argInt(args, "base_seed")
			if base == 0 {
				base = time.Now().UnixMilli() % 1_000_000_000
			}
			var resolutionID *int64
			if rid := optInt(args, "resolution_id"); rid != nil && *rid > 0 {
				resolutionID = rid
			}

			type fanItem struct {
				Path  string `json:"path,omitempty"`
				Seed  int64  `json:"seed"`
				Error string `json:"error,omitempty"`
			}
			images := make([]fanItem, 0, n)
			var paths, labels []string
			var lastEffPrompt, lastEffNeg string
			for i := 0; i < n; i++ {
				seed := base + int64(i)
				gp := generation.GenerateImageParams{
					PresetID:            presetID,
					ExtraPrompt:         extraPrompt,
					ExtraNegativePrompt: extraNeg,
					Seed:                &seed,
					FullSize:            true,
					ResolutionID:        resolutionID,
				}
				res, err := s.deps.Gen.GenerateImage(gp)
				if err != nil {
					images = append(images, fanItem{Seed: seed, Error: err.Error()})
					continue
				}
				actual := infoSeed(res.Info)
				if actual == 0 {
					actual = seed
				}
				path, err := s.saveArtifact(p.Name, fmt.Sprintf("bestof-%s-%d", slugify(p.Name), actual), res.Image, res.Info)
				if err != nil {
					images = append(images, fanItem{Seed: actual, Error: err.Error()})
					continue
				}
				images = append(images, fanItem{Path: path, Seed: actual})
				paths = append(paths, path)
				labels = append(labels, fmt.Sprintf("seed %d", actual))
				lastEffPrompt, lastEffNeg = res.EffectivePrompt, res.EffectiveNegativePrompt
			}

			if len(paths) == 0 {
				firstErr := "unknown"
				if len(images) > 0 && images[0].Error != "" {
					firstErr = images[0].Error
				}
				return "", fmt.Errorf("все %d генераций не удались: %s", n, firstErr)
			}
			usedPrompt, usedNeg := lastEffPrompt, lastEffNeg
			out := map[string]any{
				"images":          images,
				"prompt":          usedPrompt,
				"negative_prompt": usedNeg,
			}
			sheet := filepath.Join(s.deps.OutDir, fmt.Sprintf("bestof-%s-sheet.png", slugify(p.Name)))
			if err := buildContactSheet(paths, labels, sheet); err != nil {
				out["sheet_error"] = err.Error()
			} else {
				out["sheet"] = sheet
			}
			return toJSON(out), nil
		},
	})

	s.Register(Tool{
		Name:        "analyze",
		Description: "Разобрать изображение LLM-моделью: mode=tags — SD-теги для from_image, mode=describe — связное описание. Источник: image_path или image_base64.",
		InputSchema: props(map[string]any{
			"image_path":   prop("путь к изображению (png/jpg)", "string"),
			"image_base64": prop("картинка в base64", "string"),
			"mode":         prop("tags | describe (по умолчанию tags)", "string"),
		}),
		Handler: func(s *Server, args map[string]any) (string, error) {
			mode := argString(args, "mode")
			if mode == "" {
				mode = "tags"
			}
			if mode != "tags" && mode != "describe" {
				return "", fmt.Errorf("mode должен быть tags или describe")
			}
			b64 := argString(args, "image_base64")
			if b64 == "" {
				path := argString(args, "image_path")
				if path == "" {
					return "", fmt.Errorf("нужен image_path или image_base64")
				}
				var err error
				b64, err = filebrowser.ReadFileAsBase64(path)
				if err != nil {
					return "", err
				}
			}
			return s.deps.Gen.AnalyzeImage(b64, mode)
		},
	})

	s.Register(Tool{
		Name:        "upscale",
		Description: "Апскейл x2 (img2img, denoise 0.4) по image_path. Параметры генерации берутся из sidecar .json рядом с файлом (создаётся инструментами generate/best_of) — он обязателен. Результат: PNG + sidecar в out_dir.",
		InputSchema: props(map[string]any{
			"image_path": prop("путь к PNG со sidecar .json", "string"),
		}, "image_path"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			path := argString(args, "image_path")
			if path == "" {
				return "", fmt.Errorf("image_path обязателен")
			}
			b64, err := filebrowser.ReadFileAsBase64(path)
			if err != nil {
				return "", err
			}
			sidecar := sidecarInfoPath(path)
			if fi, err := os.Stat(sidecar); err == nil && fi.Size() > 1<<20 {
				return "", fmt.Errorf("sidecar %s слишком большой (%d байт)", sidecar, fi.Size())
			}
			infoBytes, err := os.ReadFile(sidecar)
			if err != nil {
				return "", fmt.Errorf("sidecar .json не найден (%s) — параметры генерации неизвестны: %w", sidecar, err)
			}
			res, err := s.deps.Gen.UpscaleImage(generation.UpscaleImageParams{
				ImageBase64: b64,
				GenInfo:     string(infoBytes),
			})
			if err != nil {
				return "", err
			}
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			outPath, err := s.saveArtifact(base, "upscale-"+slugify(base), res.Image, res.Info)
			if err != nil {
				return "", err
			}
			return toJSON(map[string]any{
				"path": outPath,
				"seed": infoSeed(res.Info),
			}), nil
		},
	})

	s.Register(Tool{
		Name:        "from_image",
		Description: "img2img или inpaint по image_path и пресету. tags (описание изменений) проходит LLM-конвертацию; для inpaint обязателен mask_path. Результат: PNG + sidecar в out_dir.",
		InputSchema: props(map[string]any{
			"image_path":         prop("путь к исходному изображению", "string"),
			"mode":               prop("img2img | inpaint (по умолчанию img2img)", "string"),
			"mask_path":          prop("путь к маске (обязателен для inpaint)", "string"),
			"preset_id":          prop("id пресета-стиля", "integer"),
			"tags":               prop("что изменить — теги или описание для LLM", "string"),
			"denoising_strength": prop("сила denoise 0..1 (по умолчанию 0.5)", "number"),
			"extra_negative":     prop("дополнительный негатив", "string"),
		}, "image_path", "preset_id"),
		Handler: func(s *Server, args map[string]any) (string, error) {
			path := argString(args, "image_path")
			if path == "" {
				return "", fmt.Errorf("image_path обязателен")
			}
			mode := argString(args, "mode")
			if mode == "" {
				mode = "img2img"
			}
			if mode != "img2img" && mode != "inpaint" {
				return "", fmt.Errorf("mode должен быть img2img или inpaint")
			}
			presetID := argInt(args, "preset_id")
			if presetID <= 0 {
				return "", fmt.Errorf("preset_id обязателен")
			}
			p, err := s.deps.DB.Get(presetID)
			if err != nil {
				return "", fmt.Errorf("preset not found: %w", err)
			}
			img, err := filebrowser.ReadFileAsBase64(path)
			if err != nil {
				return "", err
			}
			var mask string
			if mode == "inpaint" {
				maskPath := argString(args, "mask_path")
				if maskPath == "" {
					return "", fmt.Errorf("mask_path обязателен для inpaint")
				}
				mask, err = filebrowser.ReadFileAsBase64(maskPath)
				if err != nil {
					return "", err
				}
			}
			res, err := s.deps.Gen.GenerateFromImage(generation.GenerateFromImageParams{
				ImageBase64:         img,
				Mode:                mode,
				GenMode:             "preset",
				PresetID:            presetID,
				DenoisingStrength:   argFloat(args, "denoising_strength"),
				Tags:                argString(args, "tags"),
				ExtraNegativePrompt: argString(args, "extra_negative"),
				MaskBase64:          mask,
			})
			if err != nil {
				return "", err
			}
			outPath, err := s.saveArtifact(p.Name, "fromimg-"+slugify(p.Name), res.Image, res.Info)
			if err != nil {
				return "", err
			}
			return toJSON(map[string]any{
				"path":   outPath,
				"prompt": res.EffectivePrompt,
			}), nil
		},
	})

	s.Register(Tool{
		Name:        "sd_meta",
		Description: "Метаданные SD WebUI одним ответом: models, samplers, schedulers, vae, loras, upscalers. include — CSV-фильтр, например \"models,samplers\"; пусто — все.",
		InputSchema: props(map[string]any{
			"include": prop("CSV-список секций: models,samplers,schedulers,vae,loras,upscalers", "string"),
		}),
		Handler: func(s *Server, args map[string]any) (string, error) {
			want := map[string]bool{
				"models": true, "samplers": true, "schedulers": true,
				"vae": true, "loras": true, "upscalers": true,
			}
			if inc := argString(args, "include"); inc != "" {
				for k := range want {
					want[k] = false
				}
				for _, part := range strings.Split(inc, ",") {
					part = strings.TrimSpace(strings.ToLower(part))
					if _, ok := want[part]; ok {
						want[part] = true
					}
				}
			}
			res := map[string]any{}
			if want["models"] {
				if v, err := s.deps.SD.GetModels(); err != nil {
					res["models"] = map[string]string{"error": err.Error()}
				} else {
					out := make([]string, 0, len(v))
					for _, m := range v {
						out = append(out, m.Title)
					}
					res["models"] = out
				}
			}
			if want["samplers"] {
				if v, err := s.deps.SD.GetSamplers(); err != nil {
					res["samplers"] = map[string]string{"error": err.Error()}
				} else {
					out := make([]string, 0, len(v))
					for _, m := range v {
						out = append(out, m.Name)
					}
					res["samplers"] = out
				}
			}
			if want["schedulers"] {
				if v, err := s.deps.SD.GetSchedulers(); err != nil {
					res["schedulers"] = map[string]string{"error": err.Error()}
				} else {
					out := make([]string, 0, len(v))
					for _, m := range v {
						out = append(out, m.Name)
					}
					res["schedulers"] = out
				}
			}
			if want["vae"] {
				if v, err := s.deps.SD.GetVAEs(); err != nil {
					res["vae"] = map[string]string{"error": err.Error()}
				} else {
					out := make([]string, 0, len(v))
					for _, m := range v {
						out = append(out, m.ModelName)
					}
					res["vae"] = out
				}
			}
			if want["loras"] {
				if v, err := s.deps.SD.GetLoRAs(); err != nil {
					res["loras"] = map[string]string{"error": err.Error()}
				} else {
					out := make([]string, 0, len(v))
					for _, m := range v {
						out = append(out, m.Name)
					}
					res["loras"] = out
				}
			}
			if want["upscalers"] {
				if v, err := s.deps.SD.GetUpscalers(); err != nil {
					res["upscalers"] = map[string]string{"error": err.Error()}
				} else {
					out := make([]string, 0, len(v))
					for _, m := range v {
						out = append(out, m.Name)
					}
					res["upscalers"] = out
				}
			}
			return toJSON(res), nil
		},
	})

	s.Register(Tool{
		Name:        "interrupt",
		Description: "Прервать текущую генерацию SD. Forge общий с десктоп-приложением — прервёт и его генерацию; спроси пользователя, если в GUI что-то запущено.",
		InputSchema: props(nil),
		Handler: func(s *Server, _ map[string]any) (string, error) {
			if err := s.deps.Gen.InterruptGeneration(); err != nil {
				return "", err
			}
			return "прервано", nil
		},
	})
}

func infoSeed(info json.RawMessage) int64 {
	if len(info) == 0 {
		return 0
	}
	var v struct {
		Seed int64 `json:"seed"`
	}
	if err := json.Unmarshal(info, &v); err != nil {
		return 0
	}
	return v.Seed
}

func hasTag(tagsCSV, tag string) bool {
	for _, t := range strings.Split(tagsCSV, ",") {
		if strings.EqualFold(strings.TrimSpace(t), tag) {
			return true
		}
	}
	return false
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(" ", "-", "\t", "-").Replace(s)
	return filebrowser.SanitizeFilename(s)
}
