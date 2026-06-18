# Vercel (фронт) + Cloudflare Tunnel (бэк с твоего ПК)

Схема:

```
Telegram Mini App
       │
       ▼
https://твой-проект.vercel.app     ← фронт (Vercel)
       │
       ▼  fetch /api/...
https://xxxx.trycloudflare.com     ← туннель → localhost:8080
       │
       ▼
Docker / go run на твоём ПК
```

Карта не нужна. ПК должен быть включён, пока тестируешь.

---

## Часть 1. Фронт — `.env` и Vercel

### Локально (`frontend/.env`)

Скопируй:

```bash
cd frontend
cp .env.example .env
```

Содержимое для **локальной разработки**:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_DEV_INIT_DATA=
```

`VITE_DEV_INIT_DATA` — только если открываешь сайт в Chrome, не в Telegram.

### На Vercel (Dashboard)

**Settings → Environment Variables** → добавь для Production (и Preview, если нужно):

| Name | Value | Пример |
|------|-------|--------|
| `VITE_API_BASE_URL` | URL Cloudflare Tunnel | `https://random-words.trycloudflare.com` |
| `VITE_DEV_INIT_DATA` | оставь пустым | *(пусто)* |

Без `https://`, **без слэша** в конце.

Настройки проекта:

| Поле | Значение |
|------|----------|
| Root Directory | `frontend` |
| Framework | Vite |
| Build Command | `npm run build` |
| Output Directory | `dist` |

После смены `VITE_API_BASE_URL` → **Deployments → Redeploy**.

---

## Часть 2. Бэкенд на ПК

### 2.1. `.env` в корне репозитория

```bash
cp .env.example .env
```

Заполни:

```env
BOT_TOKEN=123456789:ABCdef...
FOOTBALL_DATA_TOKEN=твой_ключ_football_data

FOOTBALL_DATA_COMPETITIONS=CL,WC
FOOTBALL_DATA_DAYS_AHEAD=7

PORT=8080
DB_PATH=/data/football.db
SCHEMA_PATH=internal/repository/schema.sql
STATIC_DIR=
ENABLE_WORKER=true

# Подставишь после деплоя Vercel:
CORS_ORIGIN=https://твой-проект.vercel.app
```

`STATIC_DIR` пустой — Go отдаёт только API, фронт на Vercel.

### 2.2. Запуск через Docker

```bash
docker compose up --build
```

Проверка:

```bash
curl http://localhost:8080/health
```

Ожидается: `{"status":"ok"}`

### 2.2a. Запуск без Docker (альтернатива)

```bash
cd backend
```

Создай `backend/.env`:

```env
BOT_TOKEN=...
FOOTBALL_DATA_TOKEN=...
PORT=8080
DB_PATH=data/football.db
SCHEMA_PATH=internal/repository/schema.sql
STATIC_DIR=
ENABLE_WORKER=true
CORS_ORIGIN=https://твой-проект.vercel.app
```

```bash
go run ./cmd/server
```

---

## Часть 3. Cloudflare Tunnel

### 3.1. Установка cloudflared

Windows (PowerShell от админа):

```powershell
winget install Cloudflare.cloudflared
```

Или: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/

### 3.2. Quick tunnel (для теста)

Пока бэкенд работает на `:8080`:

```bash
cloudflared tunnel --url http://localhost:8080
```

В выводе найди строку вида:

```
https://something-random.trycloudflare.com
```

Проверь:

```bash
curl https://something-random.trycloudflare.com/health
```

### 3.3. Подставь URL туннеля

1. **Vercel** → `VITE_API_BASE_URL` = `https://something-random.trycloudflare.com` → **Redeploy**
2. Убедись, что `CORS_ORIGIN` в `.env` бэка = URL Vercel → перезапусти бэкенд:

```bash
docker compose down && docker compose up -d
# или Ctrl+C и снова go run
```

---

## Часть 4. Порядок действий (чеклист)

1. [ ] Заполни `.env` в корне (`BOT_TOKEN`, `FOOTBALL_DATA_TOKEN`)
2. [ ] `docker compose up --build` (или `go run`)
3. [ ] `curl http://localhost:8080/health` → ok
4. [ ] Задеплой фронт на Vercel (пока с заглушкой API или после туннеля)
5. [ ] Запомни URL Vercel: `https://xxx.vercel.app`
6. [ ] Пропиши `CORS_ORIGIN=https://xxx.vercel.app` в `.env` бэка → перезапуск
7. [ ] `cloudflared tunnel --url http://localhost:8080` → скопируй URL
8. [ ] Vercel: `VITE_API_BASE_URL` = URL туннеля → Redeploy
9. [ ] `curl https://туннель/health` → ok
10. [ ] BotFather: Mini App + Menu Button → **URL Vercel** (не туннель!)
11. [ ] Открой бота в Telegram → «Прогнозы»

---

## Часть 5. Telegram BotFather

URL приложения — **только Vercel**:

```
https://твой-проект.vercel.app
```

Команды:

```
/setmenubutton
```

→ выбери бота → URL = Vercel → текст «Прогнозы»

Туннель в BotFather **не указывай** — пользователи видят Vercel, API идёт в фоне на твой ПК.

---

## Часть 6. Проблемы

### CORS error в консоли браузера

- `CORS_ORIGIN` на бэке = **точный** URL Vercel (`https://...`, без `/` в конце)
- Перезапусти бэкенд после смены `.env`

### «Нет Telegram initData»

Открывай через **бота**, не по ссылке Vercel в Chrome.

### API не отвечает

- ПК включён?
- `docker compose` / `go run` запущены?
- Туннель `cloudflared` запущен в отдельном терминале?
- URL туннеля не устарел?

### После перезапуска туннеля всё сломалось

Quick tunnel **меняет URL** каждый раз. Нужно:

1. Новый URL → Vercel `VITE_API_BASE_URL` → Redeploy
2. Снова запустить туннель

### Матчей нет

Смотри логи бэка: `docker compose logs -f`  
Worker грузит матчи при старте. Проверь `FOOTBALL_DATA_TOKEN`.

---

## Стабильный URL туннеля (опционально)

Quick tunnel меняется при каждом запуске. Для постоянного URL:

1. Бесплатный аккаунт Cloudflare (без карты)
2. Свой домен или бесплатный поддомен
3. Named tunnel: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/get-started/

Тогда не придётся передеплоивать Vercel после каждого рестарта.

---

## Шпаргалка команд

```bash
# Терминал 1 — бэкенд
docker compose up --build

# Терминал 2 — туннель
cloudflared tunnel --url http://localhost:8080

# Проверки
curl http://localhost:8080/health
curl https://ТУННЕЛЬ.trycloudflare.com/health
```

Vercel env:

```
VITE_API_BASE_URL=https://ТУННЕЛЬ.trycloudflare.com
```

Бэкенд `.env`:

```
CORS_ORIGIN=https://ПРОЕКТ.vercel.app
STATIC_DIR=
```

BotFather → `https://ПРОЕКТ.vercel.app`
