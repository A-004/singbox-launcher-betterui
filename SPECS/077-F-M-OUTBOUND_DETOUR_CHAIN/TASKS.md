# TASKS 077 — Detour-цепочки

## Фаза 1 — модель + проброс + применение ✅
- [x] `Source.DetourTag` (state/connections.go)
- [x] `ProxySource.DetourTag` (configtypes/types.go)
- [x] Проброс в `ToProxySourceV4` (adapter_source.go), `sync_to_legacy.go`, `sync_to_connections.go`
- [x] `LoadNodesFromSource`: `applySourceDetour` проставляет `node.Outbound["detour"]` (skip wireguard / Jump)
- [x] Тесты: round-trip Source↔ProxySource (detour_mapping_test.go); applySourceDetour + server+detour end-to-end (detour_test.go). Эмиссия `"detour"` в JSON — существующий механизм GenerateNodeJSON.

## Фаза 2 — валидация (fail-open)
- [ ] `sanitizeDetours` в outbound_generator: висячий → drop+warn; self → drop+warn; цикл → разорвать+warn
- [ ] Тесты: каждый кейс + wg-skip + Jump-приоритет

## Фаза 3 — UI
- [ ] Дропдаун «Detour server» в source_edit_window (server + subscription), фильтр собственных тегов, «(none)»
- [ ] Хелпер применения в business; локали строк
- [ ] Round-trip выбора через закрытие/переоткрытие окна

## Фаза 4 — докуменация
- [ ] `docs/ParserConfig.md` + `docs/release_notes/upcoming.md`
- [ ] golden config-фрагмент (опц. через sing-box check)
- [ ] `IMPLEMENTATION_REPORT.md`
- [ ] `go build ./... && go test ./... && go vet ./...`
