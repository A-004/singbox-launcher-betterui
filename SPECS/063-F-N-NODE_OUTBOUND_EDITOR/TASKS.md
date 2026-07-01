# TASKS 063-F-N: Per-node outbound JSON editor

## Phase 1: Domain layer (L2)
- [ ] 1.1 Добавить `NodeOverrides map[string]map[string]interface{}` в `ProxySource` (configtypes/types.go)
- [ ] 1.2 Добавить `NodeOverrides` в `Source` модель (wizardmodels)
- [ ] 1.3 Обновить `ToProxySourceV4()` и `applyProxyEditToSource()` для round-trip NodeOverrides
- [ ] 1.4 Реализовать `mergeOverride` в `GenerateNodeJSON` (outbound_generator.go)

## Phase 2: UI — Node Edit Dialog
- [ ] 2.1 Создать `ui/configurator/tabs/node_edit_dialog.go` — диалог raw JSON редактора
- [ ] 2.2 Добавить строки локализации в `internal/locale/en.json`

## Phase 3: UI — Preview list integration
- [ ] 3.1 Заменить `widget.NewList` в Preview табе на список с кнопкой ✏️
- [ ] 3.2 Подключить диалог к кнопке ✏️ для каждого узла
- [ ] 3.3 Сохранять overrides в `Source.NodeOverrides[tag]` при Save в диалоге

## Phase 4: State persistence
- [ ] 4.1 NodeOverrides авто-сериализуются через JSON (уже часть ProxySource/Source)
- [ ] 4.2 Проверить round-trip: save → restart → overrides сохранены

## Phase 5: Сборка и тесты
- [ ] 5.1 `go build ./...` без ошибок
- [ ] 5.2 `go vet ./...` без ошибок
- [ ] 5.3 `go test ./...` проходит
- [ ] 5.4 Ручная проверка: edit node → save → config.json содержит правки