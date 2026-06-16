# Football Predictor Frontend

Минималистичный React + TypeScript frontend для backend `football-predictor`.

## Страницы

- Матчи: `/api/matches?status=active`
- Результаты: `/api/predictions/me?status=finished`
- Лидерборд: `/api/leaderboard?limit=100`

## Запуск

```bash
cd frontend
npm install
cp .env.example .env
npm run dev
```

Для Telegram Mini App backend ждет raw `initData` в заголовке `Authorization`.
Внутри Telegram он берется из `window.Telegram.WebApp.initData`.

Для локального теста вне Telegram временно укажи в `.env`:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_DEV_INIT_DATA=...
```

Не коммить `.env` с реальными данными.
