Сервис поездок для лабораторной работы 1. Он умеет создавать поездку, отдавать её по
id и завершать. Работает на chi, данные хранит в PostgreSQL через pgx.

## Что понадобится

- Go 1.27 или новее
- Docker и `tripgoctl` для локального окружения (на Windows я запускал через WSL2)
- `make`

## Как запустить

```bash
tripgoctl cluster start        # один раз на машине
tripgoctl environment start    # поднимает PostgreSQL и создаёт .env
tripgoctl connect              # покажет адреса компонентов
make migrate                   # накатывает миграции
make run                       # запускает сервис
```

`tripgoctl` кладёт в `.env` адрес PostgreSQL. Если там не хватает каких-то переменных из
списка ниже, допишите их из `.env.example`, не меняя `DATABASE_URL`. Если порт 8080
занят, запустите так: `make run HTTP_ADDR=:8081`.

При старте сервис проверяет соединение с базой и не запускается, если она недоступна.
Кроме `run` и `migrate` в Makefile есть `make migrate-down` (откат одной миграции) и
`make generate` (кодоген из OpenAPI). Собрать проект можно командой `go build ./...`.
Автотестов в этой работе нет, они появятся в лабораторной работе 2.

## Переменные окружения

Примеры значений лежат в `.env.example`, а рабочий `DATABASE_URL` берите из `.env`. Все
переменные, кроме `LOG_LEVEL`, обязательны. Сервис проверяет их при старте: если чего-то
нет, таймаут нулевой или размер пула некорректный, он не запустится и скажет почему.

| Переменная | Пример | Зачем |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес HTTP-сервера |
| `LOG_LEVEL` | `info` | уровень логов (по умолчанию `info`) |
| `SHUTDOWN_TIMEOUT` | `10s` | сколько ждать активные запросы при остановке |
| `DATABASE_URL` | из `tripgoctl` | подключение к PostgreSQL |
| `DATABASE_MAX_CONNS` | `10` | максимальный размер пула |
| `DATABASE_MIN_CONNS` | `2` | минимальный размер пула |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | сколько живёт соединение |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | таймаут подключения и проверки при старте |
| `DATABASE_QUERY_TIMEOUT` | `3s` | таймаут каждого запроса к базе |

## API

| Метод и путь | Что делает |
|---|---|
| `POST /api/v1/trips` | создаёт поездку, отвечает `201` и заголовком `Location` |
| `GET /api/v1/trips/{tripId}` | отдаёт поездку, `200` |
| `POST /api/v1/trips/{tripId}/finish` | завершает поездку, `200` |
| `GET /health` | `200`, в базу не ходит |
| `GET /ready` | `200`, или `503`, если база недоступна |

Ошибки по поездкам приходят как `application/problem+json` с полем `code`:
`invalid_request` (400), `trip_not_found` (404), `driver_busy` и `trip_completed` (409),
`internal_error` (500). `/health` и `/ready` отдают обычный JSON со статусом `ok` или
`unavailable`. Типы и роутер сгенерированы из `contracts/openapi/trip-service.openapi.yaml`.

## Как проверял

В первом терминале `make migrate && make run`, во втором (порт берите из `HTTP_ADDR`):

```bash
cat > trip.json <<'EOF'
{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
 "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
 "start_point":{"latitude":59.9398,"longitude":30.3146},
 "end_point":{"latitude":59.9290,"longitude":30.3626},
 "price":1450}
EOF

curl -i -X POST localhost:8080/api/v1/trips -H 'content-type: application/json' -d @trip.json  # 201
curl -i -X POST localhost:8080/api/v1/trips -H 'content-type: application/json' -d @trip.json  # 409 driver_busy
curl -i localhost:8080/api/v1/trips/$TRIP_ID                # 200 (TRIP_ID из ответа на создание)
curl -i -X POST localhost:8080/api/v1/trips/$TRIP_ID/finish # 200
curl -i -X POST localhost:8080/api/v1/trips/$TRIP_ID/finish # 409 trip_completed
curl -i localhost:8080/api/v1/trips/00000000-0000-0000-0000-000000000001  # 404 trip_not_found
curl -i localhost:8080/api/v1/trips/abc                                   # 400 invalid_request
```

Гонку проверяет тот же запрос с другим `driver_id` в файле `race.json`. Двадцать
параллельных вызовов должны дать один `201` и девятнадцать `409`:

```bash
seq 20 | xargs -P20 -I{} curl -s -o /dev/null -w '%{http_code}\n' \
  -X POST localhost:8080/api/v1/trips -H 'content-type: application/json' \
  -d @race.json | sort | uniq -c
```

Ещё можно проверить:

- `make migrate-down` дважды откатывает обе миграции, `make migrate` накатывает их снова;
- после `tripgoctl environment stop` `/ready` отвечает `503`, а `/health` остаётся `200`;
  после `tripgoctl environment start` `/ready` снова `200`;
- по `Ctrl+C` в терминале сервиса в логе появляются `graceful shutdown initiated` и
  `server stopped gracefully`;
- атомарность: если вторая вставка (в `trip_status_history`) падает, в `trips` ничего не
  остаётся. Я проверял это подменой статуса в записи журнала на недопустимый: повторные
  запросы возвращали `500`, а не `409 driver_busy`.

## Принятые решения

**Уровень изоляции.** `ReadCommitted`. Гонки закрывает сама база, а не уровень
изоляции: частичный уникальный индекс на водителя и условие `status = 'active'` в
`UPDATE` при завершении. `Serializable` потребовал бы повторять транзакции при ошибке
`40001`, а дополнительной защиты не дал бы.

**Менеджер транзакций.** `Do(ctx, fn)` открывает транзакцию, кладёт её в контекст и
вызывает `fn`. Репозиторий берёт исполнителя из контекста: есть транзакция, работает
через неё, нет, через пул. Бизнес-код знает только интерфейс `Do`, про pgx он не знает.
Успех даёт `COMMIT`, ошибка или паника `ROLLBACK` (откат идёт на `context.WithoutCancel`,
чтобы сработать, даже если клиент уже отключился). Вложенный `Do` переиспользует
уже открытую транзакцию. Создание и завершение поездки пишут в `trips` и
`trip_status_history` в одном `Do`.

**Запрет двух активных поездок.** Его держит частичный уникальный индекс
`trips_driver_active_unique_idx ON trips (driver_id) WHERE status = 'active'`. Проверка
`SELECT` перед `INSERT` не годится: два параллельных запроса пройдут её оба. Нарушение
индекса (код `23505`) превращается в доменную ошибку и ответ `409 driver_busy`.

**Завершение поездки.** Один `UPDATE ... WHERE id = $1 AND status = 'active' RETURNING`.
Ноль затронутых строк значит, что поездки нет или она уже завершена; какой из случаев,
выясняет дополнительное чтение (`404` или `409 trip_completed`).

## Что не сделано

- задания со звёздочкой (идемпотентность, Dockerfile);
- таймауты HTTP-сервера (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`,
  `IdleTimeout`) заданы константами в `cmd/trip-service/main.go`; в конфиг их вынесу при
  рефакторинге в лабораторной работе 2.
