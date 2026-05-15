cuckoo（不咕鸟）是以 Go 和 TypeScript 开发，由 LLM 赋能的多人线上故事续写接龙项目，在云端部署后以前端网页与用户交互。

## MVP features

- 多人房间、准备、掷骰排序、按顺序故事接龙。
- 中英文混合字数规则：CJK 字符每字 1 unit；英文连续词和数字连续串各算 1 unit。
- 每回合限时，默认 120 秒，可在 30-600 秒之间配置；超时会自动跳过并记 0 分。
- 对局结束后保存云端归档，用户可在 `/account` 查看最近对局，并进入 `/games/:roomCode` 查看详情。
- Admin 后台 `/admin/users` 支持创建用户、禁用/恢复用户、重置密码；禁用用户不能登录或加入新房间。
- AI 评委当前为 stub 扩展点：后端优先调用 AI service `/judge`，失败时回退本地占位评分。
- WebSocket 草稿预览：当前玩家输入时，其他玩家可看到临时逐字草稿和光标效果；草稿不入库、不计分。

## Project layout

```text
backend/      Go + Gin + GORM + gorilla/websocket
frontend/     React + TypeScript + Vite
ai-service/   Fastify TypeScript AI stub
```

## Local development

Backend defaults to SQLite and seeds an admin from environment variables.

More detail: [Debug Ports, CLI, and Account Management](docs/debug-and-accounts.md).

Red Hat family deployment: [Red Hat Deploy Guide](docs/redhat-deploy.md).

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

Admin users can disable/restore accounts and reset user passwords from `/admin/users`. Disabled accounts cannot log in or join/create new rooms, but historical game archive snapshots keep display names.

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

The script writes debug binaries, frontend dist, AI service dist, env examples, and a tarball under `dist-debug/`. `dist-debug/` and `*.tar.gz` are intentionally ignored by Git; upload the generated tarball manually to GitHub Releases.

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
