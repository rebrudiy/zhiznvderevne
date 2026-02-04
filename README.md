# Жизнь в деревне — блог‑симулятор

Прод‑похожий макет: Go API + генератор постов + PostgreSQL + Redis + Prometheus/Grafana/Alertmanager + CI/CD.

## Быстрый старт (локально)

1. Создай файл `.env` с `OPENAI_API_KEY`.
2. Запусти `docker compose up --build`.
3. Открой:
   - Сайт: `http://localhost:8080`
   - Grafana: `http://localhost:3000` (admin/admin)
   - Prometheus: `http://localhost:9090`

## Генерация постов

Генератор работает в фоне и публикует 1–3 поста в день (случайно в окне 06:00–22:30 по Москве). Настройка в `config/config.yaml`.

## Нагрузка

Пример нагрузочного теста:

```bash
k6 run load/k6.js
```

## CI/CD

GitHub Actions:
- `ci.yml` — тесты Go.
- `deploy.yml` — сборка Docker, пуш в GHCR, деплой на сервер через SSH.

Для деплоя нужны секреты в GitHub:
- `SSH_HOST`
- `SSH_USER`
- `SSH_KEY`

На сервере файл `docker-compose.yml`, `infra/` и `config/` размещаются в `/opt/zhiznvderevne`.

## Домены

Домен в конфиге — `жизньвдеревне.рф` (punycode: `xn--b1aafeabbii5cf7a5l.xn--p1ai`).
