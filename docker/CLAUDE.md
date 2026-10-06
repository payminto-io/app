# Payminto Docker Configuration — CLAUDE.md

## What This Is

The `docker/` directory contains configuration files that are mounted into the Docker containers at runtime. It is not a standalone service — it provides supporting config for the containers defined in the root `docker-compose.yml` (production) and `docker-compose.dev.yml` (development). There are two subdirectories: `nginx/` for the reverse proxy config and `postgres/` for database initialisation. This directory does NOT contain Dockerfiles — those live alongside each service (`backend/Dockerfile`, `frontend/Dockerfile`, etc.).

## Directory Layout

```
docker/
├── nginx/
│   └── nginx.conf        # Reverse proxy: HTTP→HTTPS, route splitting by path prefix
└── postgres/
    └── init.sql          # DB init script: creates database, user, and grants
```

## File Descriptions

### `postgres/init.sql`

Runs once when the PostgreSQL container is first created (via the Docker entrypoint `docker-entrypoint-initdb.d/` mechanism). It creates the `payminto` database and a `payminto` user with the appropriate grants. The password is injected via the `POSTGRES_PASSWORD` environment variable in `docker-compose.yml`.

Typical content:
```sql
CREATE DATABASE payminto;
CREATE USER payminto WITH ENCRYPTED PASSWORD 'POSTGRES_PASSWORD_PLACEHOLDER';
GRANT ALL PRIVILEGES ON DATABASE payminto TO payminto;
\c payminto
GRANT ALL ON SCHEMA public TO payminto;
```

> Note: In production, `setup.sh` generates a random `POSTGRES_PASSWORD` and writes it to `.env`. The compose file passes it to the postgres container as `POSTGRES_PASSWORD`. The `init.sql` is templated or the password is set via `POSTGRES_USER` / `POSTGRES_PASSWORD` env vars handled by the official postgres Docker image — the `init.sql` may just grant privileges and rely on `POSTGRES_USER` + `POSTGRES_DB` env vars to create the DB automatically.

### `nginx/nginx.conf`

The Nginx reverse proxy configuration. It handles:

1. **HTTP → HTTPS redirect**: All traffic on port 80 is redirected to port 443 with a 301.
2. **SSL/TLS termination**: TLS is terminated at Nginx using Let's Encrypt certificates mounted from the host at `/etc/letsencrypt/`.
3. **Path-based routing**:
   - `location /api/` → `proxy_pass http://backend:8080` (Go API)
   - `location /mcp` → `proxy_pass http://mcp-server:3333` (MCP server)
   - `location /widget/` → serves static file from compiled widget dist
   - `location /` → `proxy_pass http://frontend:3000` (Next.js dashboard)
4. **Security headers**: HSTS, X-Frame-Options, X-Content-Type-Options

Simplified structure:
```nginx
server {
    listen 80;
    server_name $DOMAIN;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name $DOMAIN;

    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /mcp {
        proxy_pass http://mcp-server:3333;
    }

    location / {
        proxy_pass http://frontend:3000;
    }
}
```

Internal service names (`backend`, `frontend`, `mcp-server`) are resolved by Docker's internal DNS using the service names defined in `docker-compose.yml`.

## How It Fits into docker-compose

### Production (`../docker-compose.yml`)

The production compose file:
- Starts all services: `postgres`, `redis`, `backend`, `frontend`, `mcp-server`, `nginx`
- Mounts `docker/nginx/nginx.conf` into the nginx container at `/etc/nginx/conf.d/default.conf`
- Mounts `docker/postgres/init.sql` into the postgres container at `/docker-entrypoint-initdb.d/init.sql`
- Mounts Let's Encrypt certs from the host at `/etc/letsencrypt/` into nginx
- Reads all secrets from a root `.env` file

### Development (`../docker-compose.dev.yml`)

The dev compose file starts only `postgres` and `redis` — the backend, frontend, and nginx are run locally for hot-reloading. The `postgres/init.sql` is still mounted. Nginx is NOT used in dev (each service is accessed directly on its port).

## Common Operations

```bash
# From payminto/ directory:

# Start full production stack
docker compose up -d --build

# Start only postgres + redis (dev mode)
docker compose -f docker-compose.dev.yml up -d

# View nginx logs
docker compose logs -f nginx

# View postgres logs
docker compose logs -f postgres

# Reload nginx config (after editing nginx.conf) without downtime
docker compose exec nginx nginx -s reload

# Validate nginx config before reloading
docker compose exec nginx nginx -t

# Connect to postgres CLI
docker compose exec postgres psql -U payminto -d payminto

# Restart a single service
docker compose restart backend

# Rebuild a single service image
docker compose up -d --build backend
```

## Let's Encrypt Setup

SSL certificates are NOT auto-renewed inside the Docker setup by default. The recommended approach:
1. Run `certbot` on the host machine (outside Docker) to obtain and auto-renew certs.
2. Nginx reads the certs from the host via the volume mount `/etc/letsencrypt/`.
3. On renewal, run `docker compose exec nginx nginx -s reload` to pick up new certs.

The `setup.sh` script handles initial cert issuance via certbot when you provide `DOMAIN` and `EMAIL`.

## Integration Points

| System | Direction | Details |
|--------|-----------|---------|
| Nginx → backend | internal Docker network | `http://backend:8080` |
| Nginx → frontend | internal Docker network | `http://frontend:3000` |
| Nginx → mcp-server | internal Docker network | `http://mcp-server:3333` |
| Postgres init | one-time at container creation | `init.sql` runs via docker-entrypoint-initdb.d |
| Host Let's Encrypt | volume mount into nginx | `/etc/letsencrypt/` read-only |

## Key Files to Read First

1. `nginx/nginx.conf` — full routing and SSL config
2. `postgres/init.sql` — database and user creation
3. `../docker-compose.yml` — see how these files are mounted (volumes section)
4. `../docker-compose.dev.yml` — dev-only compose (postgres/redis only)
