#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT_DIR/dist-debug"
PACKAGE_NAME="cuckoo-debug-linux-amd64.tar.gz"

export GOCACHE="${GOCACHE:-/tmp/cuckoo-go-build}"
export GOMODCACHE="${GOMODCACHE:-/tmp/cuckoo-go-mod}"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR/backend" "$OUT_DIR/frontend" "$OUT_DIR/ai-service"
mkdir -p "$OUT_DIR/deploy" "$OUT_DIR/docs"

(
  cd "$ROOT_DIR/backend"
  go test ./...
  go build -gcflags="all=-N -l" -o "$OUT_DIR/backend/cuckoo-server-debug" ./cmd/server
  go build -gcflags="all=-N -l" -o "$OUT_DIR/backend/cuckoo-cli-debug" ./cmd/cuckoo
)

(
  cd "$ROOT_DIR/frontend"
  npm run build
  cp -R dist "$OUT_DIR/frontend/dist"
)

(
  cd "$ROOT_DIR/ai-service"
  npm run build
  cp -R dist package.json package-lock.json "$OUT_DIR/ai-service/"
)

cp -R "$ROOT_DIR/deploy/." "$OUT_DIR/deploy/"
cp "$ROOT_DIR/readme.md" "$OUT_DIR/"
cp -R "$ROOT_DIR/docs/." "$OUT_DIR/docs/"

cat > "$OUT_DIR/debug.env.example" <<'ENV'
CUCKOO_ENV=development
HTTP_ADDR=:18081
FRONTEND_URL=http://localhost:15173
DB_DRIVER=sqlite
DB_DSN=cuckoo-debug.db
JWT_SECRET=change-this-debug-secret
CUCKOO_ADMIN_USERNAME=admin
CUCKOO_ADMIN_PASSWORD=admin12345
AI_SERVICE_URL=http://localhost:18787
AI_JUDGE_ENABLED=false
OPENAI_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4.1-mini
OPENAI_API_STYLE=responses
AI_JUDGE_PERSONA=
ENV

cat > "$OUT_DIR/README-debug.md" <<'EOF'
# Cuckoo Debug Build

Default debug ports:

- Backend API: http://localhost:18081
- Frontend static site: serve `frontend/dist` with Caddy or another static file server
- AI service: http://localhost:18787

Run backend:

```bash
cd backend
set -a
. ../debug.env.example
set +a
./cuckoo-server-debug
```

Serve frontend:

```bash
# Example with Caddy or any static server:
# root: frontend/dist
# proxy /api/* to 127.0.0.1:18081
```

Run AI service:

```bash
cd ai-service
npm install --omit=dev
PORT=18787 node dist/server.js
```

Enable OpenAI-compatible judge:

```bash
AI_JUDGE_ENABLED=true \
OPENAI_API_KEY=sk-... \
OPENAI_MODEL=gpt-4.1-mini \
OPENAI_API_STYLE=responses \
PORT=18787 node dist/server.js
```

For OpenAI-compatible providers that only support Chat Completions, set:

```bash
OPENAI_API_STYLE=chat_completions
OPENAI_BASE_URL=https://provider.example/v1
```

For Red Hat family deployment, see `deploy/systemd/*.service` and `deploy/caddy/Caddyfile.example`.

Useful checks:

```bash
curl http://localhost:18787/health
curl -i http://localhost:18081/api/me
```
EOF

(
  cd "$ROOT_DIR"
  TAR_TMP="/tmp/$PACKAGE_NAME"
  rm -f "$TAR_TMP"
  tar -czf "$TAR_TMP" -C "$OUT_DIR" .
  mv "$TAR_TMP" "$OUT_DIR/$PACKAGE_NAME"
)

echo "Debug build written to $OUT_DIR"
echo "Release tarball written to $OUT_DIR/$PACKAGE_NAME"
