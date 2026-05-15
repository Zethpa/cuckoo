#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT_DIR/dist-debug"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR/backend" "$OUT_DIR/frontend" "$OUT_DIR/ai-service"

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
ENV

cat > "$OUT_DIR/README-debug.md" <<'EOF'
# Cuckoo Debug Build

Default debug ports:

- Backend API: http://localhost:18081
- Frontend dev server: http://localhost:15173
- AI service: http://localhost:18787

Run backend:

```bash
cd backend
set -a
. ../debug.env.example
set +a
./cuckoo-server-debug
```

Run frontend from source during debug:

```bash
cd ../frontend-source
npm run dev
```

Run AI service:

```bash
cd ai-service
npm install --omit=dev
PORT=18787 node dist/server.js
```
EOF

echo "Debug build written to $OUT_DIR"
