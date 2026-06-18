# Полная инструкция по деплою Football Predictor

Один сервер: Go отдаёт и API (`/api/*`), и фронтенд (React). База — SQLite на persistent volume.

Рекомендуемый бесплатный вариант: **Fly.io** (HTTPS из коробки + диск для БД).

---

## Что понадобится заранее

| Что | Где взять |
|-----|-----------|
| Аккаунт GitHub | github.com (если деплоишь через git) |
| Telegram-бот | [@BotFather](https://t.me/BotFather) → `/newbot` |
| Ключ football-data.org | [football-data.org/client/register](https://www.football-data.org/client/register) (бесплатно) |
| Аккаунт Fly.io | [fly.io](https://fly.io) (нужна карта, на free tier списывают мало или 0) |

Сохрани два секрета — они понадобятся везде:

- `BOT_TOKEN` — строка вида `123456789:ABCdef...`
- `FOOTBALL_DATA_TOKEN` — ключ с football-data.org (придёт на почту)

---

## Шаг 0. Проверка локально (перед облаком)

Убедись, что проект собирается и работает у тебя на машине.

### 0.1. Создай `.env` в корне репозитория

```bash
cp .env.example .env
```

Открой `.env` и заполни:

```env
BOT_TOKEN=твой_токен_от_BotFather
FOOTBALL_DATA_TOKEN=твой_ключ_football_data
FOOTBALL_DATA_COMPETITIONS=CL,WC
FOOTBALL_DATA_DAYS_AHEAD=7
PORT=8080
DB_PATH=/data/football.db
SCHEMA_PATH=internal/repository/schema.sql
STATIC_DIR=static
ENABLE_WORKER=true
CORS_ORIGIN=
```

### 0.2. Запусти Docker

```bash
docker compose up --build
```

Дождись строк в логах:

```
HTTP server listening on :8080
Serving static files from static
Worker: Расписание матчей обновляется раз в сутки
```

### 0.3. Проверь health

В другом терминале:

```bash
curl http://localhost:8080/health
```

Ожидаемый ответ: `{"status":"ok"}`

Открой в браузере `http://localhost:8080` — увидишь интерфейс, но **без Telegram** авторизация не пройдёт (это нормально).

Останови: `Ctrl+C` или `docker compose down`.

---

## Шаг 1. Деплой на Fly.io

### 1.1. Установи flyctl

Windows (PowerShell):

```powershell
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"
```

macOS / Linux:

```bash
curl -L https://fly.io/install.sh | sh
```

Проверка:

```bash
fly version
```

### 1.2. Войди в аккаунт

```bash
fly auth login
```

Откроется браузер — залогинься.

### 1.3. Зарегистрируй приложение

Из **корня** репозитория (`football-predictor/`):

```bash
fly launch --no-deploy
```

На вопросы отвечай примерно так:

| Вопрос | Ответ |
|--------|-------|
| App name | уникальное имя, напр. `my-football-predictor` (или оставь предложенное) |
| Region | `ams` (Амстердам) — должен совпадать с `primary_region` в `fly.toml` |
| PostgreSQL | **No** (у нас SQLite) |
| Redis | **No** |
| Deploy now | **No** (мы ещё настроим volume и секреты) |

Если имя `football-predictor` занято — Fly предложит другое. Запомни его: URL будет `https://ИМЯ.fly.dev`.

При необходимости поправь `app = "..."` в файле `fly.toml`.

### 1.4. Создай диск для SQLite

**Важно:** регион volume должен совпадать с регионом приложения (`ams`).

```bash
fly volumes create football_data --region ams --size 1
```

Имя `football_data` должно совпадать с `source` в `fly.toml`:

```toml
[[mounts]]
  source = "football_data"
  destination = "/data"
```

### 1.5. Задай секреты

```bash
fly secrets set BOT_TOKEN="твой_BOT_TOKEN" FOOTBALL_DATA_TOKEN="твой_FOOTBALL_DATA_TOKEN"
```

Проверить (значения не покажут, только имена):

```bash
fly secrets list
```

### 1.6. Деплой

```bash
fly deploy
```

Первый деплой займёт 3–10 минут (сборка фронта + Go).

### 1.7. Проверь, что сервер жив

```bash
fly status
fly logs
```

В логах должны быть:

- `HTTP server listening on :8080`
- `Serving static files from static`
- `Worker: Расписание матчей обновляется раз в сутки`

Проверь health:

```bash
curl https://ИМЯ-ПРИЛОЖЕНИЯ.fly.dev/health
```

Открой в браузере `https://ИМЯ-ПРИЛОЖЕНИЯ.fly.dev` — должен открыться интерфейс.

### 1.8. Если приложение «спит» (free tier)

В `fly.toml` стоит `min_machines_running = 0` — машина останавливается без трафика. Первый запрос после простоя может идти 10–30 секунд. Это нормально для бесплатного тарифа.

Чтобы всегда было онлайн (платно), поменяй на `min_machines_running = 1`.

---

## Шаг 2. Настройка Telegram Mini App

Без этого шага бот есть, но **открыть приложение из Telegram нельзя**.

### 2.1. Создай Mini App

В [@BotFather](https://t.me/BotFather):

1. `/mybots` → выбери своего бота
2. **Bot Settings** → **Menu Button** → **Configure menu button**
3. Укажи:
   - **Text:** `Прогнозы` (или любой)
   - **URL:** `https://ИМЯ-ПРИЛОЖЕНИЯ.fly.dev` (твой Fly URL, **с https://**)

Либо через команды:

```
/newapp
```

Выбери бота → введи название приложения → введи описание → загрузи иконку (или пропусти) → **URL:** `https://ИМЯ-ПРИЛОЖЕНИЯ.fly.dev`

### 2.2. Кнопка меню (если ещё не настроена)

```
/setmenubutton
```

Выбери бота → URL тот же: `https://ИМЯ-ПРИЛОЖЕНИЯ.fly.dev`

### 2.3. Проверка в Telegram

1. Открой своего бота в Telegram (на телефоне или десктопе)
2. Нажми кнопку меню **«Прогнозы»** (или как назвал)
3. Должно открыться Mini App с матчами
4. Профиль вверху справа должен показать твоё имя и очки (не «Telegram»)

Если видишь ошибку «Нет Telegram initData» — ты открыл сайт в обычном браузере, а не через бота.

---

## Шаг 3. Проверка, что всё работает end-to-end

Чеклист:

- [ ] `curl https://ИМЯ.fly.dev/health` → `{"status":"ok"}`
- [ ] Mini App открывается из бота
- [ ] Вкладка **Матчи** — появляются игры (worker грузит их при старте; если пусто — см. раздел «Проблемы»)
- [ ] Можно выбрать П1 / X / П2 — прогноз сохраняется
- [ ] **Лидерборд** — ты в списке после первого прогноза
- [ ] **Статистика прогнозов** на карточке — показывает голоса
- [ ] После завершения матча (до 30 мин) — результат и очки обновятся

---

## Шаг 4. Обновление после изменений в коде

```bash
fly deploy
```

Секреты и volume сохраняются. База на диске `/data` не сбрасывается.

---

## Альтернатива: свой VPS (Oracle Cloud Always Free)

Если Fly.io не подходит:

1. Создай Ubuntu VM (Oracle Cloud Always Free, Ampere A1)
2. Открой порты 22, 80, 443
3. На сервере:

```bash
sudo apt update
sudo apt install -y docker.io docker-compose-plugin git

git clone https://github.com/ТВОЙ_ЮЗЕР/football-predictor.git
cd football-predictor
cp .env.example .env
nano .env   # BOT_TOKEN, FOOTBALL_DATA_TOKEN

sudo docker compose up -d --build
```

4. Поставь Caddy для HTTPS (пример):

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy
```

`/etc/caddy/Caddyfile`:

```
predict.твой-домен.ru {
    reverse_proxy localhost:8080
}
```

```bash
sudo systemctl reload caddy
```

5. В BotFather укажи `https://predict.твой-домен.ru`

---

## Переменные окружения (справочник)

| Переменная | Обязательно | Описание |
|------------|-------------|----------|
| `BOT_TOKEN` | да | Токен бота от BotFather |
| `FOOTBALL_DATA_TOKEN` | да | API-ключ football-data.org |
| `FOOTBALL_DATA_COMPETITIONS` | нет | Лиги: `CL,WC` (Лига чемпионов, ЧМ) |
| `FOOTBALL_DATA_DAYS_AHEAD` | нет | Сколько дней вперёд грузить матчи (7) |
| `PORT` | нет | Порт HTTP (8080) |
| `DB_PATH` | нет | Путь к SQLite (`/data/football.db` в Docker) |
| `STATIC_DIR` | нет | Папка фронта (`static` в Docker) |
| `ENABLE_WORKER` | нет | Фоновая синхронизация (`true`) |
| `CORS_ORIGIN` | нет | Не нужен при одном сервере |

На Fly.io секреты задаются через `fly secrets set`, остальное — в `fly.toml` `[env]`.

---

## Проблемы и решения

### «Нет Telegram initData» / 401 Unauthorized

Открывай приложение **только через бота** (кнопка меню), не по прямой ссылке в Chrome.

### Матчей нет на вкладке «Матчи»

1. Проверь логи: `fly logs`
2. Ищи `Worker Error (Fetch)` — часто неверный `FOOTBALL_DATA_TOKEN`
3. Убедись, что сейчас есть матчи CL/WC в ближайшие 7 дней (межсезонье = пусто)
4. Перезапусти: `fly apps restart ИМЯ-ПРИЛОЖЕНИЯ`

### База обнулилась после деплоя

Volume не примонтирован. Проверь:

```bash
fly volumes list
```

В `fly.toml` должен быть блок `[[mounts]]`. Пересоздай volume и задеплой снова.

### Mini App не открывается / белый экран

1. URL в BotFather должен быть **https://** (не http)
2. Проверь `curl https://ИМЯ.fly.dev/health`
3. Смотри логи: `fly logs`

### football-data.org: 429 Too Many Requests

Бесплатный тариф — 10 запросов в минуту. Worker не должен спамить; если видишь 429 — подожди минуту и перезапусти.

### Docker локально не стартует

- Установлен Docker Desktop?
- Порты 8080 свободен?
- В `.env` заполнены оба токена?

---

## Схема работы после деплоя

```
Пользователь в Telegram
        │
        ▼
  Кнопка «Прогнозы» в боте
        │
        ▼
  https://ИМЯ.fly.dev  (фронт + API)
        │
        ├── /api/*  → Go + SQLite (/data/football.db)
        └── /       → React (статика)
        
  Worker в фоне:
  ├── при старте — загрузка матчей
  ├── раз в сутки — обновление расписания
  └── каждые 30 мин — проверка результатов
```

---

## Краткая шпаргалка (все команды подряд)

```bash
# Локально
cp .env.example .env
# заполни токены в .env
docker compose up --build

# Fly.io
fly auth login
fly launch --no-deploy
fly volumes create football_data --region ams --size 1
fly secrets set BOT_TOKEN="..." FOOTBALL_DATA_TOKEN="..."
fly deploy
curl https://ИМЯ.fly.dev/health

# BotFather: Menu Button / newapp → https://ИМЯ.fly.dev
```

---

## Деплой на Vercel (только фронт) + API на Fly.io

**На одном Vercel всё не получится** — этот проект не serverless:

| Компонент | Vercel | Почему |
|-----------|--------|--------|
| React-фронт | да | статика, бесплатно |
| Go API (Gin) | нет | нужен постоянно работающий сервер |
| SQLite | нет | нет постоянного диска на serverless |
| Worker (cron) | нет | нет фоновых процессов |

Рабочая схема: **фронт на Vercel**, **бэкенд на Fly.io** (бесплатно).

```
Telegram Mini App
       │
       ▼
https://твой-проект.vercel.app   ← фронт (Vercel)
       │
       ▼  API-запросы
https://твой-api.fly.dev/api/... ← бэкенд (Fly.io) + SQLite
```

### Шаг 1. Задеплой API на Fly.io (без фронта)

Создай **отдельное** приложение для API, например `football-predictor-api`:

```bash
fly auth login
```

Скопируй `fly.toml` или создай новый:

```bash
fly launch --no-deploy --name football-predictor-api
```

В `fly.toml` для API-приложения **убери раздачу статики** и задай CORS (подставишь URL Vercel после деплоя фронта):

```toml
app = "football-predictor-api"
primary_region = "ams"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"
  DB_PATH = "/data/football.db"
  SCHEMA_PATH = "internal/repository/schema.sql"
  STATIC_DIR = ""
  ENABLE_WORKER = "true"
  FOOTBALL_DATA_COMPETITIONS = "CL,WC"
  FOOTBALL_DATA_DAYS_AHEAD = "7"
  CORS_ORIGIN = "https://твой-проект.vercel.app"

[http_service]
  internal_port = 8080
  force_https = true

[[mounts]]
  source = "football_data"
  destination = "/data"
```

```bash
fly volumes create football_data --region ams --size 1 -a football-predictor-api
fly secrets set BOT_TOKEN="..." FOOTBALL_DATA_TOKEN="..." -a football-predictor-api
fly deploy -a football-predictor-api
```

Проверка:

```bash
curl https://football-predictor-api.fly.dev/health
```

Запомни URL API: `https://football-predictor-api.fly.dev`

### Шаг 2. Задеплой фронт на Vercel

1. Залей репозиторий на **GitHub**
2. Зайди на [vercel.com](https://vercel.com) → **Add New Project** → импортируй репозиторий
3. Настройки сборки:

| Поле | Значение |
|------|----------|
| Root Directory | `frontend` |
| Framework Preset | Vite |
| Build Command | `npm run build` |
| Output Directory | `dist` |

4. **Environment Variables** (важно — именно на этапе сборки):

| Имя | Значение |
|-----|----------|
| `VITE_API_BASE_URL` | `https://football-predictor-api.fly.dev` |

Без `https://`, без слэша в конце.

5. **Deploy**

После деплоя получишь URL вида `https://football-predictor-xxx.vercel.app`

### Шаг 3. Обнови CORS на бэкенде

Подставь реальный URL Vercel:

```bash
fly secrets set CORS_ORIGIN="https://football-predictor-xxx.vercel.app" -a football-predictor-api
```

Или поправь `CORS_ORIGIN` в `fly.toml` и сделай `fly deploy`.

### Шаг 4. Telegram BotFather

Mini App URL и Menu Button → **URL Vercel**, не Fly:

```
https://football-predictor-xxx.vercel.app
```

### Шаг 5. Проверка

1. `curl https://football-predictor-api.fly.dev/health` → ok
2. Открой `https://твой-проект.vercel.app` в браузере — будет ошибка auth (нормально)
3. Открой бота в Telegram → кнопка «Прогнозы» → должно работать

### Обновление

| Что менял | Команда |
|-----------|---------|
| Только фронт | push в GitHub → Vercel пересоберёт сам |
| Только бэкенд | `fly deploy -a football-predictor-api` |
| Сменил `VITE_API_BASE_URL` | Vercel → Settings → Environment Variables → **Redeploy** |

### Проблемы Vercel + Fly

| Проблема | Решение |
|----------|---------|
| CORS error в консоли | `CORS_ORIGIN` на Fly = точный URL Vercel (с `https://`, без `/` в конце) |
| 401 в Telegram | Открывай через бота, не напрямую Vercel |
| Старый API после смены URL | Redeploy на Vercel после изменения `VITE_API_BASE_URL` |
| Fly API спит | Первый запрос 10–30 сек — cold start |

### Почему не «всё на Vercel»

Чтобы всё было на Vercel, пришлось бы:

- переписать Go API на Serverless Functions
- заменить SQLite на внешнюю БД (Supabase / Neon)
- вынести worker в Vercel Cron

Это другой проект. Текущий стек рассчитан на один Docker-контейнер или связку **Vercel (фронт) + Fly (API)**.

Готово.
