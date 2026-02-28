# План: Переделка деплоя на Docker Compose

## Контекст

VPS ещё не настроен → миграционных затрат нет. Текущий деплой (systemd + manual setup) заменяем на Docker Compose. Один и тот же стек работает локально и на VPS.

## Подход: CI binary + Docker build на VPS (без registry)

- CI собирает Go binary (кросс-компиляция) + frontend dist — как сейчас
- SCP на VPS: binary, dist, docker-compose.yml, Dockerfile, Caddyfile
- VPS: `docker compose up -d --build` (build = FROM alpine + COPY binary, ~2 сек)

**Почему без ghcr.io:** один VPS, registry добавляет настройку auth, а binary уже скомпилирован — build мгновенный.

## Стек

```
Docker Compose
├── postgres   (postgres:17-alpine, volume pgdata)
├── api        (alpine:3.20 + binary, healthcheck, mounts config.yaml)
└── caddy      (caddy:2-alpine, HTTPS, reverse proxy → api, serves frontend)
```

## Структура на VPS: `/opt/bytebrew/`

```
/opt/bytebrew/
├── docker-compose.yml       ← из репо, обновляется CI
├── Caddyfile                ← из репо, обновляется CI
├── .env                     ← секреты, создаётся ВРУЧНУЮ один раз
├── config.yaml              ← конфиг API, создаётся ВРУЧНУЮ один раз
├── api/
│   ├── Dockerfile           ← из репо, обновляется CI
│   └── bytebrew-cloud-api   ← бинарник, обновляется CI
└── frontend/                ← dist файлы, обновляются CI
```

## Этап 1: Создать Docker-файлы

### 1.1 `deploy/api/Dockerfile`
```dockerfile
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY bytebrew-cloud-api /usr/local/bin/
RUN chmod +x /usr/local/bin/bytebrew-cloud-api
EXPOSE 8080
ENTRYPOINT ["bytebrew-cloud-api"]
CMD ["--config", "/etc/bytebrew/config.yaml"]
```
Не multi-stage — binary уже собран в CI. Только runtime.

### 1.2 `deploy/docker-compose.yml`
```yaml
services:
  postgres:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: bytebrew
      POSTGRES_USER: bytebrew
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bytebrew -d bytebrew"]
      interval: 5s
      timeout: 3s
      retries: 5

  api:
    build: ./api
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./config.yaml:/etc/bytebrew/config.yaml:ro
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 10s

  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./frontend:/var/www/html:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      api:
        condition: service_healthy

volumes:
  pgdata:
  caddy_data:
  caddy_config:
```

### 1.3 `deploy/Caddyfile`
Обновить для Docker networking: `api:8080` вместо `127.0.0.1:8080`, домен через env var.
```
{$DOMAIN:localhost} {
    handle /api/* {
        reverse_proxy api:8080
    }
    handle /health {
        reverse_proxy api:8080
    }
    handle {
        root * /var/www/html
        try_files {path} /index.html
        file_server
    }
}
```

### 1.4 `deploy/.env.example`
```
DB_PASSWORD=CHANGE_ME
DOMAIN=localhost
```

### 1.5 `deploy/config.example.yaml`
Копия `deploy/config.production.yaml` с `database.url` указывающим на `postgres:5432` (Docker service name) вместо `localhost:5432`.

## Этап 2: Переписать deploy.yml

**Job `build`** (без изменений по сути):
- Checkout → Go build → frontend build → upload artifacts

**Job `deploy`**:
1. Download artifacts
2. SCP на VPS в `/opt/bytebrew/`:
   - `api/bytebrew-cloud-api` (binary)
   - `api/Dockerfile`
   - `frontend/` (dist files)
   - `docker-compose.yml`
   - `Caddyfile`
3. SSH: backup → docker compose up → health check → rollback if fail

```bash
# Бэкап
cd /opt/bytebrew
cp api/bytebrew-cloud-api api/bytebrew-cloud-api.bak 2>/dev/null || true

# Обновить файлы (уже скопированы SCP)

# Deploy
docker compose up -d --build --remove-orphans

# Health check (ждём start_period)
sleep 10
if ! curl -sf http://localhost:80/health; then
  echo "Health check failed! Rolling back..."
  mv api/bytebrew-cloud-api.bak api/bytebrew-cloud-api
  docker compose up -d --build
  docker compose logs --tail=50 api
  exit 1
fi

rm -f api/bytebrew-cloud-api.bak
```

**Secrets (GitHub):**
- `VPS_HOST` — IP
- `VPS_SSH_KEY` — SSH ключ deploy юзера

**Больше не нужны:** sudo, systemctl, journalctl, sudoers.

## Этап 3: Упростить setup-vps.sh

Новый setup-vps.sh делает только:
1. Установка Docker + Docker Compose plugin
2. Создание deploy юзера + добавление в группу `docker`
3. Создание `/opt/bytebrew/` с правильными permissions
4. SSH ключ для deploy юзера
5. Инструкции: создать `.env` и `config.yaml`

Убрать: PostgreSQL, Caddy, systemd unit, sudoers — всё внутри Docker.

## Файлы

| Действие | Файл | Что |
|----------|------|-----|
| **Создать** | `deploy/api/Dockerfile` | Runtime image для API |
| **Создать** | `deploy/.env.example` | Шаблон .env |
| **Создать** | `deploy/config.example.yaml` | Шаблон конфига для Docker |
| **Переписать** | `deploy/docker-compose.yml` (новый) | Production compose: postgres + api + caddy |
| **Переписать** | `deploy/Caddyfile` | Docker networking (api:8080) |
| **Переписать** | `deploy/setup-vps.sh` | Только Docker + deploy user |
| **Переписать** | `.github/workflows/deploy.yml` | Artifacts + SCP + docker compose |
| **Удалить** | `deploy/bytebrew-cloud-api.service` | Больше не нужен (Docker lifecycle) |

## Верификация

1. **Локально:** собрать binary + frontend, `docker compose up`, curl localhost/health
2. **YAML lint:** проверить docker-compose.yml и deploy.yml синтаксис
3. **Bash syntax:** проверить setup-vps.sh

## Замечания

- `config.production.yaml` сохраняется как справочник, но для Docker используется `config.example.yaml` с `postgres:5432`
- Deploy user НЕ нужен sudoers — только членство в группе `docker`
- Миграции автоматически при старте API (embedded в binary)
- Caddy в Docker автоматически получает HTTPS сертификат через Let's Encrypt
