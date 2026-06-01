# Deployment Guidelines

## Scenario: Production Compose and Debug Override

### 1. Scope / Trigger

- Trigger: any change to `docker-compose.prod.yml`, `docker-compose.debug.yml`, `frontend/nginx.conf`, `.env.example`, or deployment documentation that changes how the backend is reached.
- This is an infra contract because it controls which ports are exposed, how the frontend reaches the backend, and how users run released images.

### 2. Signatures

- Production command:
  ```bash
  CUL_VERSION=<latest-release-tag> docker compose -f docker-compose.prod.yml up -d
  ```
- Debug command:
  ```bash
  CUL_VERSION=<latest-release-tag> docker compose \
    -f docker-compose.prod.yml \
    -f docker-compose.debug.yml \
    up -d
  ```
- Health check:
  ```bash
  docker compose -f docker-compose.prod.yml exec -T backend wget -qO- http://127.0.0.1:8080/healthz
  ```

### 3. Contracts

- `docker-compose.prod.yml` must define service names `backend` and `frontend`.
- `frontend/nginx.conf` must proxy `/api/` to `http://backend:8080`.
- Production Compose must publish frontend `8088:80`.
- Production Compose must not publish backend `8080` to the host by default.
- `docker-compose.debug.yml` is the opt-in place for backend `8080:8080` host publishing.
- Released-image deployments should pin `CUL_VERSION` to the latest GitHub Release tag instead of hard-coding a stale version in docs.

### 4. Validation & Error Matrix

- Missing `backend` service name -> frontend `/api` proxy cannot resolve the upstream.
- Publishing backend `8080` in production Compose -> larger default network surface than needed.
- Missing debug override -> users have no documented way to direct-check `/healthz` from the host.
- Hard-coded old release tag in docs -> users deploy stale images when copying commands.

### 5. Good/Base/Bad Cases

- Good: `docker-compose.prod.yml` exposes only `8088:80`; `docker-compose.debug.yml` exposes `8080:8080`.
- Base: source-build `docker-compose.yml` may expose backend for local development convenience.
- Bad: production docs tell users to run an old fixed release tag such as `v0.1.0`.

### 6. Tests Required

- Validate production Compose:
  ```bash
  docker compose -f docker-compose.prod.yml config --quiet
  ```
- Validate debug override:
  ```bash
  docker compose -f docker-compose.prod.yml -f docker-compose.debug.yml config --quiet
  ```
- Search docs before finishing:
  ```bash
  rg "v0\\.1\\.0|8080:8080" README.md README.zh-CN.md docs/deployment.md docker-compose*.yml
  ```

### 7. Wrong vs Correct

#### Wrong

```yaml
services:
  backend:
    ports:
      - "8080:8080"
```

Putting this in `docker-compose.prod.yml` exposes the backend API in normal production deployments even though the frontend can reach it over the Compose network.

#### Correct

```yaml
# docker-compose.prod.yml
services:
  backend:
    image: ghcr.io/yyykf/cpa-usage-lens-backend:${CUL_VERSION:-latest}

# docker-compose.debug.yml
services:
  backend:
    ports:
      - "8080:8080"
```

Keep the default deployment small and expose backend direct access only when the operator explicitly opts into debugging.
