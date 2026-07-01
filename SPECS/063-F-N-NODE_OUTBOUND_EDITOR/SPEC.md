# SPEC 063-F-N: Per-node outbound JSON editor в Source Preview

## Проблема

Пользователь не может редактировать сгенерированные outbound-поля отдельных прокси-узлов
из подписки. Парсер строит `ParsedNode.Outbound` по жёсткой логике из URI-строки,
и нет UI для правки полей типа `streamSettings.tlsSettings.congestion`,
`mux.enabled`, `downlinkOnly`, `uplinkOnly` и любых других sing-box outbound-полей.

## Решение

Добавить в Source Edit → Preview таб кнопку ✏️ для каждого узла, открывающую
raw JSON редактор `ParsedNode.Outbound`. Правки сохраняются как per-node overrides
в state и мержатся поверх сгенерированного Outbound при сборке `config.json`.

## Требования

1. В Preview Nodes списке у каждой строки узла — кнопка ✏️ «Edit node JSON».
2. Диалог: MultiLineEntry с форматированным JSON `ParsedNode.Outbound`.
3. Сохранение: override хранится как `Source.NodeOverrides[tag] = map[string]interface{}`.
4. При `GenerateNodeJSON` override мержится поверх `node.Outbound`.
5. Overrides персистятся в `state.json` и переживают перезапуск.
6. Кнопка «Reset» сбрасывает override до factory (удаляет запись).

## Критерии приёмки

- [ ] В Preview табе Source Edit окна каждый узел имеет кнопку ✏️.
- [ ] Нажатие открывает диалог с raw JSON outbound-а.
- [ ] Правки сохраняются и применяются при сборке `config.json`.
- [ ] Overrides переживают перезапуск приложения.
- [ ] Reset возвращает исходный outbound.