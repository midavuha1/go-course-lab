# trip-service

Сервис поездок. Лабораторная работа(2). Умеет создавать поездку, отдавать её по id
и завершать. HTTP на chi, PostgreSQL через pgx, телеметрия через OpenTelemetry.

В лабораторной работе 2 код разложен по слоям, добавлены unit и интеграционные тесты,
линтер, структурные логи, трейсы и метрики. Снаружи сервис не изменился.

## Что понадобится

- Go 1.27 или новее
- Docker и `tripgoctl` для локального окружения (на Windows — через WSL2)
- `make`, `golangci-lint` v2

## Как запустить

```bash
tripgoctl cluster start        # один раз на машине
tripgoctl environment start    # PostgreSQL + коллектор телеметрии, создаёт .env
tripgoctl connect              # адреса базы, Jaeger, Grafana, Prometheus
make migrate                   # накатывает миграции
make run                       # запускает сервис
```

Если порт 8080 занят: `make run HTTP_ADDR=:8081`. При старте сервис проверяет соединение
с базой и не запускается, если она недоступна.

| Команда | Что делает |
|---|---|
| `make generate` | кодоген из OpenAPI и моки (`go generate`) |
| `make lint` | golangci-lint с конфигом курса |
| `make test` | unit-тесты с `-race`, в конце печатает покрытие domain + usecase |
| `make test-integration` | тесты репозитория на базе из `tripgoctl` (тег `integration`), прогоняются дважды |

## Переменные окружения

Примеры — в `.env.example`, рабочие значения — в `.env`, который создаёт `tripgoctl`.
Все переменные, кроме `LOG_LEVEL`, обязательны: без них сервис не запустится и скажет почему.

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
| `OTEL_SERVICE_NAME` | `trip-service` | имя сервиса в телеметрии |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | из `tripgoctl` | адрес коллектора |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | протокол отправки |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | какие трейсы записывать |
| `OTEL_RESOURCE_ATTRIBUTES` | `service.namespace=tripgo,deployment.environment=local` | атрибуты сервиса |

## API

| Метод и путь | Что делает |
|---|---|
| `POST /api/v1/trips` | создаёт поездку, отвечает `201` и заголовком `Location` |
| `GET /api/v1/trips/{tripId}` | отдаёт поездку, `200` |
| `POST /api/v1/trips/{tripId}/finish` | завершает поездку, `200` |
| `GET /health` | `200`, в базу не ходит |
| `GET /ready` | `200`, или `503`, если база недоступна |

Ошибки приходят как `application/problem+json` с полем `code`: `invalid_request` (400),
`trip_not_found` (404), `driver_busy` и `trip_completed` (409), `internal_error` (500,
без деталей внутренней ошибки). Типы и роутер сгенерированы из
`contracts/openapi/trip-service.openapi.yaml`.

## Архитектура

Код разложен по слоям. Каждый слой знает только о тех, что глубже: HTTP и репозиторий знают
про сценарии (usecase) и домен, сценарии — только про домен, домен — ни о ком.

| Слой | Что делает |
|---|---|
| `internal/domain/trip` | поездка и её правила: создание, завершение, статусы, ошибки |
| `internal/model` | структуры, которые переносят данные между слоями |
| `internal/usecase/trip/{create,get,finish}` | сценарии: транзакция, репозиторий, метрика, лог |
| `internal/http/...` | принимает запрос, отдаёт ответ, переводит ошибки в HTTP-коды |
| `internal/repository/trip` | SQL-запросы к PostgreSQL |
| `internal/infra` | подключение к базе, транзакции, логгер и телеметрия |
| `internal/container` | создаёт все объекты и связывает их между собой |
| `cmd/trip-service` | читает конфиг, собирает контейнер, запускает сервер, ждёт сигнала |

Правила слоёв проверяет линтер (depguard): например, импорт `pgx` в домен — ошибка `make lint`.

**Где объявлены интерфейсы и почему там:** 

В файле `contract.go` того пакета, который ими
пользуется. Сценарий сам описывает, что ему нужно от хранилища, и ничего лишнего: create
просит два метода, get — один, finish — три. Репозиторий об этих интерфейсах не знает,
Go сам проверяет, что методы подходят. Поэтому в тестах репозиторий легко заменить моком,
а сценарии не зависят от того, где лежат данные.

**Ошибки.** Домен говорит «водитель занят» или «поездка не найдена», а какой это HTTP-код,
решает одна функция `domainerror.Problem`. Внутренние ошибки клиенту не показываются,
только пишутся в лог.

**Телеметрия.** На каждый запрос — трейс в Jaeger со спанами SQL, строка в логе с тем же
`trace_id` и точка в метриках Grafana. Порядок middleware: Recover → Tracing → Logging →
Metrics → роутер. Если коллектор недоступен, сервис продолжает работать.

**Отличие от схемы в задании:** Точка входа — `cmd/trip-service`: имя осталось с лабы 1.

## Как проверял

Автотесты: `make test`, `make test-integration`, `make lint`.

Руками: в первом терминале `make run`, во втором (порт берите из `HTTP_ADDR`):

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

Телеметрия: трейс запроса виден в Jaeger, `trace_id` из строки лога `request handled`
находит тот же трейс, метрики `http_server_request_duration` и `tripgo_trips_created`
видны в Grafana. С недоступным коллектором (`make run OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1`)
сервис отвечает как обычно, в логе предупреждение `telemetry export failed`.

## Принятые решения

**Уровень изоляции.** `ReadCommitted`. Гонки закрывает сама база, а не уровень изоляции:
частичный уникальный индекс на водителя и блокировка строки при завершении. `Serializable`
потребовал бы повторять транзакции при ошибке `40001`, а дополнительной защиты не дал бы.

**Менеджер транзакций.** `Do(ctx, fn)` открывает транзакцию, кладёт её в контекст и вызывает
`fn`. Репозиторий берёт исполнителя из контекста: есть транзакция — работает через неё, нет —
через пул. Сценарии знают только интерфейс `Do`, про pgx они не знают. Ошибка или паника
дают `ROLLBACK` (на `context.WithoutCancel`, чтобы откат сработал, даже если клиент отключился).
Создание и завершение поездки пишут в `trips` и `trip_status_history` в одном `Do`.

**Запрет двух активных поездок.** Его держит частичный уникальный индекс
`trips_driver_active_unique_idx ON trips (driver_id) WHERE status = 'active'`. Проверка
`SELECT` перед `INSERT` не годится: два параллельных запроса пройдут её оба. Нарушение
индекса (код `23505`) превращается в доменную ошибку и ответ `409 driver_busy`.

**Завершение поездки.** В одной транзакции: `SELECT … FOR UPDATE` блокирует строку,
доменный метод `Trip.Finish` проверяет, что поездка активна, затем `UPDATE`. Правило
перехода живёт в домене и тестируется без базы, а блокировка не даёт двум параллельным
finish завершить поездку дважды: второй дождётся первого и получит `409 trip_completed`.

## Что не сделано

- задание со звёздочкой (мутационное тестирование);
- таймауты HTTP-сервера (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`)
  пока заданы константами в `internal/container/container.go`.
