# PLAN 063-F-N: Per-node outbound JSON editor

## Архитектура

### Слой данных (L2 core-domain)
- `ProxySource` + `Source` получают поле `NodeOverrides map[string]map[string]interface{}` — ключ = tag узла, значение = override поля для `node.Outbound`.
- `GenerateNodeJSON` применяет override через `mergeOverrideOverOutbound(node, override)` перед сериализацией.
- Overrides персистятся в `state.json` как часть source-объекта.

### Слой UI (L6 ui-views)
- `source_edit_window.go`: Preview таб — заменить `widget.NewList` на кастомный список с кнопкой ✏️ у каждого узла.
- Диалог: `ShowNodeEditDialog(parent, node, onSave)` — MultiLineEntry + Save/Cancel/Reset.
- Узлы кэшируются в preview-горутине; overrides хранятся в `model.Sources[i].NodeOverrides`.

### Поток данных
```
Source Edit Preview → parsePreviewNodesFromBody → nodes[]
  → render list with ✏️ per node
  → click ✏️ → ShowNodeEditDialog(node.Outbound)
  → Save → Source.NodeOverrides[tag] = edited map
  → serializeParserAfterSourceEdit → state.json
  → GenerateNodeJSON → mergeOverride → config.json
```

## Файлы

### Изменяемые
| Файл | Изменения |
|------|-----------|
| `core/config/configtypes/types.go` | `ProxySource.NodeOverrides` + `ParsedNode.Override` |
| `core/config/outbound_generator.go` | `GenerateNodeJSON` — merge override |
| `ui/configurator/models/source.go` | `Source.NodeOverrides` |
| `ui/configurator/tabs/source_edit_window.go` | Preview list with ✏️ + dialog |
| `internal/locale/en.json` | Строки для диалога |
| `core/state/` | Сериализация NodeOverrides (авто через JSON) |

### Новые
| Файл | Назначение |
|------|------------|
| `ui/configurator/tabs/node_edit_dialog.go` | Диалог raw JSON редактора |

## Риски
- Override должен быть shallow merge — не заменять весь Outbound, а только переопределять указанные ключи.
- При изменении source (новый fetch) узлы могут поменяться — stale overrides для несуществующих тегов должны игнорироваться.