# API для мобильного клиента

Базовый адрес задаётся портом `PORT`; пути ниже начинаются от корня сервера. REST использует JSON. Все маршруты, кроме `/health` и `/auth/*`, требуют заголовок `Authorization: Bearer <access_token>`. JWT действует 15 минут; refresh-токена сейчас нет. Ошибки REST имеют вид `{"error":"текст"}`.

## REST

| Метод и путь | Запрос | Успешный ответ | Основные ошибки |
| --- | --- | --- | --- |
| `GET /health` | — | `200 {"status":"ok"}` | `503` при недоступности БД |
| `POST /auth/register` | `{"nickname":"alice","email":"a@example.com","password":"..."}` | `201 {"id":"<uuid>"}` | `400` неверные данные, `409` email занят |
| `POST /auth/login` | `{"email":"a@example.com","password":"..."}` | `200 {"access_token":"<jwt>"}` | `401` неверные данные |
| `GET /me` | Bearer JWT | `200 {"nickname":"alice","email":"a@example.com","rating":1200}` | `401`, `404` |
| `GET /me/games?limit=20&offset=0` | Bearer JWT; `limit` 1–100, по умолчанию 20; `offset` от 0 | `200 {"games":[...],"limit":20,"offset":0}` | `400`, `401` |
| `GET /me/games/:id` | Bearer JWT; `id` — UUID | `200` с данными партии, `fen` и `moves` | `400`, `401`, `404` (включая чужую партию) |
| `POST /matchmaking/search` | Bearer JWT; тело не нужно | `202 {"status":"searching"}` или `201 {"game_id":"<uuid>","white_id":"<uuid>","black_id":"<uuid>"}` | `401`, `409` если уже в очереди |
| `GET /matchmaking/status` | Bearer JWT | `200 {"status":"searching"}` или `200` с `game_id`, `white_id`, `black_id` | `401`, `404` если не в очереди |
| `DELETE /matchmaking/search` | Bearer JWT | `204` без тела | `401`, `404` если не в очереди |

В списке `games` каждая партия содержит `id`, `white_id`, `black_id`, `result`, `ply`, `created_at`, `updated_at`, `white_rating_before`, `black_rating_before`, `white_rating_after`, `black_rating_after`. `result` равен `null` для активной партии, затем `"1-0"`, `"0-1"` или `"1/2-1/2"`. Снимки рейтинга равны `null` до завершения партии. Список отсортирован от новых партий к старым.

`GET /me/games/:id` добавляет текущий `fen` и `moves`: массив объектов `{ "ply": 1, "uci": "e2e4", "fen": "<FEN после хода>", "created_at": "<RFC3339>" }` по порядку ходов. Для проигрывания с начальной позиции используйте стандартную шахматную расстановку, затем последовательно применяйте UCI-ходы или FEN каждого хода.

## Игра по WebSocket

Подключение: `GET /ws/games/:id` с тем же заголовком Bearer JWT и WebSocket upgrade. `id` берётся из ответа матчмейкинга. Подключиться могут только игроки этой партии. Ошибки до upgrade: `400` неверный UUID, `401` нет действующего токена, `403` чужая партия или запрещённый Origin, `404` партия не найдена, `409` партия завершена. Токен в query-параметре не поддерживается.

Клиент отправляет JSON-сообщения:

```json
{"type":"move","payload":{"from":"e2","to":"e4"}}
```

Для превращения добавьте `"promotion":"q"` (также `r`, `b`, `n`). Другие типы: `resign`, `draw_offer`, `draw_accept`, `draw_decline`; им не нужен `payload`.

Сервер присылает:

- `{"type":"game_state","payload":{"fen":"<FEN>","turn":"w","status":"*","last_move":"e2e4","white_ms":599000,"black_ms":600000}}` при подключении и после изменения состояния. `turn` — `w` или `b`; `status` — `*` во время игры или итоговый результат. `last_move` может отсутствовать. Часы задаются в миллисекундах; сейчас контроль 10 минут без добавления.
- `{"type":"draw_offer","payload":{}}` сопернику при предложении ничьей.
- `{"type":"error","payload":{"code":"invalid_move","message":"invalid move"}}` при ошибке команды. Возможные коды: `invalid_move`, `not_your_turn`, `time_expired`, `no_draw_offer`.

Очередь матчмейкинга и часы хранятся в памяти сервера. После перезапуска очередь теряется, а часы незавершённой партии запускаются заново при следующем подключении. `GET /matchmaking/status` удаляет сохранённое уведомление о найденной партии после его получения; сохраните `game_id` на клиенте.
