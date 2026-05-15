cuckoo（不咕鸟）是以 Go 和 TypeScript 开发，由 LLM 赋能的多人线上故事续写接龙项目，在云端部署后以前端网页与用户交互。

## Project layout

```text
backend/      Go + Gin + GORM + gorilla/websocket
frontend/     React + TypeScript + Vite
ai-service/   Fastify TypeScript AI stub
```

## Local development

Backend defaults to SQLite and seeds an admin from environment variables.

More detail: [Debug Ports, CLI, and Account Management](docs/debug-and-accounts.md).

```bash
cd backend
CUCKOO_ADMIN_USERNAME=admin CUCKOO_ADMIN_PASSWORD=admin12345 go run ./cmd/server
```

Create another user:

```bash
cd backend
go run ./cmd/cuckoo user add --username alice --role player
```

If `--password` is omitted, the CLI generates an initial password from the username and server secret, prints it once, and stores only a bcrypt hash. Admin users can also create accounts from `/admin/users`; players can change their own password from `/account`.

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Debug defaults avoid common cloud service ports:

- Backend API: `http://localhost:18081`
- Frontend dev server: `http://localhost:15173`
- AI service: `http://localhost:18787`

The Vite dev server proxies `/api` and WebSocket traffic to the backend.

Build debug artifacts:

```bash
./scripts/debug-build.sh
```

AI service stub:

```bash
cd ai-service
npm install
npm run dev
```

## Useful checks

```bash
cd backend && go test ./...
cd frontend && npm run build
cd ai-service && npm run build
```
