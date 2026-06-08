# Upcoming release — черновик

Сюда складываем пункты, которые войдут в следующий релиз. Перед релизом переносим в `X-Y-Z.md` и очищаем этот файл.

**Не добавлять** сюда мелкие правки **только UI** (порядок виджетов, выравнивание, стиль кнопок без смены действия и т.п.). Писать **новое поведение**: данные, форматы, сохранение, заметные для пользователя возможности.

## EN
### Highlights
- **Debug API: "Regenerate token" button.** Settings → Debug API now has a Regenerate button next to Copy token. It rotates the bearer token (confirm dialog — the old token stops working immediately) and, if the API is running, restarts the listener with the new token.

### Technical / Internal
-

## RU
### Основное
- **Debug API: кнопка «Перегенерировать токен».** В Settings → Debug API рядом с «Копировать токен» появилась кнопка перегенерации. Она ротирует bearer-токен (с подтверждением — старый сразу перестаёт работать) и, если API запущен, перезапускает listener с новым токеном.

### Техническое / Внутреннее
-
