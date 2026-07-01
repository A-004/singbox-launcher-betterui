# IMPLEMENTATION REPORT — SPEC 063-F-N: Per-node outbound JSON editor

## Статус: реализовано

## Изменённые файлы

| Файл | Что сделано |
|------|-------------|
| `core/config/configtypes/types.go` | `ProxySource.NodeOverrides` + `ParsedNode.Override` |
| `core/config/outbound_generator.go` | `GenerateNodeJSON` — shallow-merge override на Outbound |
| `core/state/connections.go` | `Source.NodeOverrides` |
| `core/state/adapter_source.go` | `ToProxySourceV4` — round-trip NodeOverrides |
| `ui/configurator/models/wizard_state_file.go` | `Source` — алиас на corestate.Source (уже содержит NodeOverrides) |
| `ui/configurator/tabs/node_edit_dialog.go` | **Новый** — диалог raw JSON редактора узла |
| `ui/configurator/tabs/source_edit_window.go` | Preview таб: кнопки ✏️ + интеграция диалога + apply override |
| `internal/locale/en.json` | Строки для node edit dialog |

## Как это работает

1. Source Edit → Preview таб → каждый узел имеет кнопку ✏️
2. Клик ✏️ → `showNodeEditDialog` с `ParsedNode.Outbound` как JSON
3. Сохранение → `Source.NodeOverrides[tag] = diff от factory`
4. При генерации `config.json` → `GenerateNodeJSON` мержит override поверх Outbound
5. Overrides персистятся в `state.json` через существующий механизм Save

## Дата реализации: 2026-07-01