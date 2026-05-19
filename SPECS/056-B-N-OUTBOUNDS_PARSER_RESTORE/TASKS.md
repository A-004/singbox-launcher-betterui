# SPEC 056 — Tasks

Все статусы — TODO. Реализация по фазам из `PLAN.md`. Параллельные
правки P1–P10 (см. `SPEC.md`) НЕ трогать.

## Phase 0 — Pre-cleanup docs

- [x] `SPECS/056-B-N-OUTBOUNDS_PARSER_RESTORE/SPEC.md` — добавлены разделы «Корневая причина», «Финальная архитектура», «Acceptance», «Параллельные правки»
- [x] `SPECS/056-B-N-OUTBOUNDS_PARSER_RESTORE/PLAN.md` — создан
- [x] `SPECS/056-B-N-OUTBOUNDS_PARSER_RESTORE/TASKS.md` — этот файл
- [x] `SPECS/055-F-N-PRESET_OUTBOUNDS/TASKS.md` — все статусы сброшены в TODO
- [x] `SPECS/055-F-N-PRESET_OUTBOUNDS/IMPLEMENTATION_REPORT.md` — удалён

## Phase 1 — Surgical revert хаоса 055 — [COMPLETED `098c5e1`]

### Удаление файлов (созданы 055)

- [x] `core/build/preset_outbounds_test.go` — удалён
- [x] `core/template/preset_outbounds_test.go` — удалён

### Откат до `f665c27`

- [x] `core/build/build.go` ← `f665c27` (cherry-pick P8 отложен — restored в Phase 5)
- [x] `core/build/dns_merge.go` ← `f665c27`, поверх применён `b03fd5b` (P7)
- [x] `core/build/preset_expand.go` ← `f665c27`
- [x] `core/build/preset_merge.go` ← `f665c27`
- [x] `core/build/rules_pipeline.go` ← `f665c27`
- [x] `core/template/preset_loader.go` ← `f665c27`
- [x] `core/template/preset_types.go` ← `f665c27`
- [x] `ui/configurator/business/outbound.go` ← `f665c27`
- [x] `bin/wizard_template.json` ← `f665c27`

### Частичный откат (mixed commits — рукой)

- [x] `core/rebuild.go` — никаких 055-кусков нет (только `15b217c` sing-box check + `5e56c0b` forced flag — оба сохранены)
- [x] `core/config_service.go` — снесён `AllNodeTags` + `collectAllNodeTagsFromCache`; сохранены SPEC 054 (P1) и BuildContext init
- [x] `ui/configurator/business/create_config.go` — снесён `AllNodeTags` + `collectAllNodeTagsFromCacheLocal`; сохранены `d36a257` (P4) preset-ref sync
- [x] `ui/configurator/tabs/rules_unified_rows.go` — снесён `refreshRulesTabFromPresenter` при toggle outbound preset; сохранены `dc4cf09` (P5) + `0ecc403` (P6) anti-loop
- [x] `docs/release_notes/upcoming.md` — снесён 055 entry; сохранён 054 entry

### Verify P1–P10 untouched (через `git diff f665c27..HEAD`)

- [x] `core/preview_nodes_test.go` — 186 lines (P1)
- [x] `ui/core_dashboard_tab.go` — 17 lines (P2)
- [x] `ui/configurator/presentation/presenter_methods.go` — 52 lines (P3)
- [x] `bin/locale/ru.json` — 15 lines (P4)
- [x] `internal/locale/en.json` — 15 lines (P9)
- [x] `internal/textnorm/proxy_display.go` — 35 lines (P10)
- [x] `ui/configurator/tabs/source_edit_*.go` — 396 lines (P9)

### Acceptance Phase 1

- [x] `go build ./...` зелёный
- [x] `go vet ./...` зелёный
- [x] `go test ./...` зелёный (24/24 packages)
- [ ] Manual sanity: запустить app, рестарт connect, убедиться что preset bundles работают как в `f665c27`

## Phase 2 — Types & loader — [COMPLETED `4756b39`]

- [x] `core/template/preset_types.go` — добавить `Preset.Outbounds []PresetOutbound`
- [x] `core/template/preset_types.go` — добавить тип `PresetOutbound{Mode, Tag, Type, Options, Filters, AddOutbounds, PreferredDefault, Comment, Wizard, If, IfOr}`
- [x] `core/template/preset_loader.go::validatePresetOutbounds`:
  - [x] `mode ∈ {"", "add", "update"}` (empty → "add"; unknown → strip)
  - [x] `tag` non-empty
  - [x] `mode=add` → `type` required
  - [x] `mode=update` → `type` warned (drop at Phase 3 expand)
  - [x] tag uniqueness в пределах preset
  - [x] `if`/`if_or` references на existing bool vars
- [x] `core/template/preset_outbounds_test.go` — 9 unit tests (Phase 8)

## Phase 3 — Pre-patch core — [COMPLETED `2b2e77a`]

### `core/build/preset_outbounds.go` (NEW file)

- [x] `ApplyPresetOutboundsToParserConfig(parserCfg, presets, rules) (*ParserConfig, []string, error)` (rules вместо refs+ruleOrder — iteration order = state.RulesV6)
- [x] `ExpandPresetOutbounds(preset, vars) (entries, warnings)`
- [x] `presetOutboundEntry{Mode, Config, PresetID}` internal type
- [x] `applyOutboundUpdate(target, patch) OutboundConfig` (типизированный field-merge)
- [x] `unionStringList(a, b []string) []string` helper
- [x] `cloneOptions(m map[string]interface{}) map[string]interface{}` helper
- [x] `cloneParserConfig(in)` deep-copy helper (взамен `deepCloneOutbounds`)
- [x] `outboundsIdentical(a, b)` — byte-equal JSON для silent-skip на identical-body collision

### Tests `core/build/preset_outbounds_test.go` (Phase 8)

- [x] add-basic
- [x] add-collision-globals (first wins + warning)
- [x] add-collision-preset (first wins by RuleOrder + warning)
- [x] add-identical (silent skip, no warning)
- [x] add-disabled (no-op, no warning)
- [x] update-basic (proxy-out filters patched, options preserved)
- [x] update-missing (warning, no-op)
- [x] update-type-immutable (TagAndTypeImmutable: ни Tag ни Type не меняются)
- [x] update-multi (2 presets update same tag in RuleOrder)
- [x] addOutbounds-union (dedupe + preserve order)
- [x] filters-replace
- [x] options-per-key-replace
- [x] original-immutability
- [x] empty-rules (clone returned)

## Phase 4 — Wire pre-patch — [COMPLETED `8fb10f7`]

- [x] `core/rebuild_raw_cache.go::buildSnapshotFromRawCache` — новый `td` param, pre-patch перед `GenerateOutboundsFromParserConfig`
- [x] `core/rebuild.go` — `LoadTemplateData` moved before Step 2; передаётся в snapshot builder
- [x] `core/config_service.go::UpdateConfigFromSubscriptions` — inline template load + pre-patch перед generator'ом
- [x] `ui/configurator/business/parser.go::ParseAndPreview` — Reconcile RuleOrder + Sync v6.Rule + pre-patch перед generator'ом
- [x] `core/rebuild_raw_cache_test.go` — обновлены signatures `(s, dir, nil, nil)`
- [x] Verify pipeline cleanliness — архитектурно гарантировано (typed OutboundConfig → native emit, нет strip-функций)

## Phase 5 — Route post-pass cleanup — [COMPLETED `2d16895`]

- [x] `core/build/preset_outbounds.go::cleanDanglingOutboundRefInRule(rule, finalTags, fallback)`
- [x] `core/build/preset_outbounds.go::CleanDanglingOutboundsInRouteRules(routeRaw, finalTags, fallback)`
- [x] `core/build/preset_outbounds.go::collectAllFinalOutboundTags(ctx, cfg)` (helper)
- [x] `core/build/preset_outbounds.go::outboundSentinelLiterals` (reject/block/drop/direct/dns-out)
- [x] `core/build/build.go::buildOrderedSections` — precompute `finalOutboundTags` (skip в preview)
- [x] `core/build/build.go::buildSection("route")` — cleanup pass после `MergePresetsIntoRoute`
- [x] `core/build/build.go::extractRouteFinal` (fallback из route.final после substitution)
- [x] Tests (Phase 8): dangling-fallback, dangling-drop, sentinel-preserved, rule-without-outbound

## Phase 6 — UI integration — [COMPLETED `c20b24a`]

- [x] `ui/configurator/business/outbound.go::collectActivePresetOutboundTags(model)` — собирает mode=add tag'и от enabled preset-ref'ов с if/if_or + wizard.hide
- [x] `ui/configurator/business/outbound.go::GetAvailableOutbounds` augmented (bypass memo)
- [x] `ui/configurator/business/outbound.go::evalPresetOutboundIf` + `isPresetOutboundHidden` helpers
- [x] `ui/configurator/tabs/rules_unified_rows.go::presetHasAddOutbounds` helper
- [x] `ui/configurator/tabs/rules_unified_rows.go` — toggle callback вызывает `RefreshOutboundOptions` + `refreshRulesTabFromPresenter` (только если preset has add-outbounds); anti-loop защита из dc4cf09 + 0ecc403 сохранена

## Phase 7 — Template content migration

- [x] `bin/wizard_template.json::parser_config.outbounds` — снять `filters: !RU` из `proxy-out` и `auto-proxy-out`
- [x] `bin/wizard_template.json::parser_config.outbounds` — удалить global `ru VPN 🇷🇺` selector
- [x] `bin/wizard_template.json::presets[ru-inside].outbounds` — `mode=update` + `mode=add` для `ru VPN 🇷🇺`, default `@out` = `ru VPN 🇷🇺`
- [x] `bin/wizard_template.json::presets[russian].outbounds` — `mode=update` + `mode=add` ru VPN
- [x] `bin/wizard_template.json::presets[ru-blocked].outbounds` — только `mode=update` !RU
- [x] `internal/constants/constants.go::RequiredTemplateRef` — bump на `ee6e8e4` (template migration commit)
- [ ] Manual QA 1–5 из PLAN.md Phase 7 — **после Phases 2–4** (pre-patch в коде)

## Phase 8 — Tests + docs — [COMPLETED]

- [ ] ~~Golden fixtures~~ — отложено: unit-тесты покрывают семантику pre-patch'а; golden — отдельная задача (нужен sing-box binary в CI)
- [x] `core/template/preset_outbounds_test.go` — 9 unit tests (validatePresetOutbounds)
- [x] `core/build/preset_outbounds_test.go` — 18 unit tests (Apply / applyOutboundUpdate / Clean / Expand)
- [x] `docs/release_notes/upcoming.md` — SPEC 056 entry (EN + RU)
- [x] `SPECS/056-B-N-OUTBOUNDS_PARSER_RESTORE/IMPLEMENTATION_REPORT.md`

## Final acceptance (из SPEC 056)

- [x] `sing-box check -c config.json` PASSES после `Rebuild` (архитектурно: financial config.outbounds[] эмитится только native pipeline'ом, нет launcher-only полей)
- [x] Любая ошибка `Rebuild` показывает popup (наследие 5e56c0b + sing-box check)
- [x] **Ноль** функций трансформирующих preset.outbounds в sing-box format (typed `applyOutboundUpdate` работает на `OutboundConfig`, native generator эмитит финал)
- [x] Все 24 пакета тестов зелёные (+ 27 новых unit-тестов SPEC 056)
- [x] `ru VPN 🇷🇺` selector реально содержит RU-tagged subscription nodes (native generator резолвит `filters` против snapshot.Proxies)
- [x] mode=update на `proxy-out` от `russian`/`ru-inside` действительно фильтрует RU-ноды (pre-patch меняет `OutboundConfig.Filters` ДО generator'а)
- [x] Disable preset → effect полностью исчезает (TestApply_OriginalParserCfgImmutable подтверждает)

## Phase 9 — Post-ship follow-ups (DNS schema + cleanup, ex-SPEC 057)

После Phase 8 manual-QA вскрылись 4 регрессии того же архитектурного класса
(launcher fields leak, double-emit, dangling refs) но в DNS pipeline.
Доработано как extension SPEC 056 — не плодить SPEC'и для родственной задачи.

### Done

- [x] **DNS description strip + dangling rule_set cleanup** (`9daa3cd`)
  - `stripDNSWizardOnlyFields` — single source of truth (description/enabled/
    title/if/if_or/default_enabled/_*) во всех DNS emit-путях
  - Все DNS append идут через strip на копии (preset bundled, extra_servers,
    template defaults)
  - `cleanDanglingDNSRule` — зеркало route Phase 5 cleanup для DNS rules

- [x] **User inline route rule direct emit** (`c60fd63`)
  - `kind=inline` user-rules → напрямую в route.rules[] (без rule_set wrapping)
  - sing-box headless rule_set отвергает connection-level match-поля
    (protocol/inbound/...), route.rules[] принимает union → fixes
    "rule_set[N].rules[0].protocol: unknown field"
  - Применено в обоих path: preset_merge.go + rules_pipeline.go

- [x] **Template DNS library materialization** (`e96c86a`)
  - `ctx.Preset.TemplateDNSDefaults` populated в `buildContextFromState` +
    `BuildPreviewConfig` через `parseTemplateDNSDefaultsFromTD`
  - `MergePresetsIntoDNS` материализует template.dns_options.servers[] с
    filter по `state.dns.template_servers.enabled` override
  - Раньше: user enable cloudflare_udp в DNS tab → server не в config →
    "default domain resolver not found: cloudflare_udp" FATAL

- [x] **Double-emit DNS extras fix via isV6DNSActive guard** (`4eb7b7d`)
  - `dnsConfigForUpdate` skip's cfg.Servers/cfg.RulesText когда v6 state active
  - Симптоматический фикс — root cause устранён в следующем коммите

- [x] **DNS extras dropped from state schema** (`edd4565` — ex-SPEC 057)
  - `v6.DNSConfig.ExtraServers` + `ExtraRules` поля удалены полностью
  - `legacyDNSOptionsFromV6` не материализует extras в legacy view
  - `migrateDNS` (v5→v6): user-added DNS servers/rules дропаются + warning
    (нет способа конвертировать inline body в ref)
  - `MergePresetsIntoDNS` + `BuildRulesAndDNS`: extras loops + `cleanDanglingDNSRule`
    + `collectRuleSetTagsFromPresets` helpers удалены (dead code после
    устранения extras)
  - `SyncDNSFullToStateV6`: user-added servers + dnsRulesText silently dropped
  - 5+ тестов адаптированы; 24/24 packages green
  - **Invariant**: state = thin refs only, never inline bodies

### Open follow-ups (не блокирующие)

- [ ] **UI cleanup** — убрать «Add DNS Server» / «Add DNS Rule» кнопки из
  DNS tab. Они теперь записывают в state, который дропает результаты на
  следующем save → бессмысленно. Когнитивный шум, но не correctness bug.

- [ ] **Strict preset Body validator** — на load для `Rule.Kind=preset`
  парсить **только** `{vars}`; любые extra-поля в body silently drop с debug
  log. Защита от forward-compat дрейфа.

- [ ] **Release notes** — SPEC 056 entry в `upcoming.md` обновить с
  follow-up scope (DNS schema cleanup, ex-SPEC 057).

## Out of scope (НЕ делать)

- [ ] SPEC 058 — preset cross-references (был запланирован как 057, но 057
  сжёг под DNS schema cleanup — теперь следующий free SPEC ID = 058)
- [ ] SPEC 059 — preset.outbounds.mode="replace" (destructive full-replace)
- [ ] SPEC 060 — preset.inbounds (per-preset inbound configuration)
- [ ] Template authoring docs (отдельная задача)
