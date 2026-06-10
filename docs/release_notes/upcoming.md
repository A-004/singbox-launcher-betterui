# Upcoming release — черновик

Сюда складываем пункты, которые войдут в следующий релиз. Перед релизом переносим в `X-Y-Z.md` и очищаем этот файл.

## EN
### Highlights
- **AmneziaWG 2.0 header ranges (`H1-H4 = lo-hi`) no longer break the node.** Real AWG 2.0 exports randomize magic headers as ranges; the parser silently dropped all four, the node imported without them and the handshake never matched. A range now collapses to its start (any in-range value is accepted by the server). Applies to `awg://` links, `vpn://` profiles and pasted `.conf` text. (SPEC 073.2)

### Technical / Internal
- `parseAWGNumeric` in `node_parser_wireguard.go`: plain uint32 as before; `lo-hi` → range start, reversed range tolerated; invalid values keep the skip-with-debug-log policy. Tests: `awg_range_test.go`.

## RU
### Основное
- **Диапазоны заголовков AmneziaWG 2.0 (`H1-H4 = lo-hi`) больше не ломают узел.** Реальные экспорты AWG 2.0 рандомизируют magic-заголовки диапазонами; парсер молча выбрасывал все четыре — узел импортировался без них, и handshake не сходился. Теперь диапазон схлопывается в своё начало (сервер принимает любое значение внутри). Работает для `awg://`-ссылок, `vpn://`-профилей и вставленного `.conf`-текста. (SPEC 073.2)

### Техническое / Внутреннее
- `parseAWGNumeric` в `node_parser_wireguard.go`: одиночный uint32 как раньше; `lo-hi` → начало диапазона, перевёрнутый диапазон терпится; невалидные значения — прежняя политика skip + debug-log. Тесты: `awg_range_test.go`.
