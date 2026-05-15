# Debug Ports, CLI, and Account Management

本文档说明 cuckoo 调试端口、admin 初始账号、CLI 用法、账号管理、对局归档和发布包生成方式。

## 1. Debug 端口

当前 debug 默认端口避开了常见云服务端口：

| Service | Port | URL |
| --- | ---: | --- |
| Frontend Vite dev server | `15173` | `http://localhost:15173` |
| Backend API/WebSocket | `18081` | `http://localhost:18081` |
| AI service stub | `18787` | `http://localhost:18787` |

前端 Vite 会把 `/api` 和 WebSocket 请求代理到 backend：

```text
http://localhost:15173/api/* -> http://localhost:18081/api/*
```

相关默认值在这些文件中：

- `backend/internal/config/config.go`
- `frontend/vite.config.ts`
- `ai-service/src/server.ts`
- `dist-debug/debug.env.example`

## 2. Admin 初始账号密码

后端启动时会读取两个环境变量：

```bash
CUCKOO_ADMIN_USERNAME=admin
CUCKOO_ADMIN_PASSWORD=admin12345
```

默认值：

```text
username: admin
password: admin12345
```

调整方式：

```bash
cd backend
CUCKOO_ADMIN_USERNAME=owner \
CUCKOO_ADMIN_PASSWORD='replace-with-a-strong-password' \
JWT_SECRET='replace-with-a-long-random-secret' \
go run ./cmd/server
```

注意：

- seed admin 只会在数据库里不存在同名用户时创建。
- 如果 SQLite 数据库里已经有 `admin`，只改环境变量不会修改现有用户密码。
- 本地 SQLite 默认文件是 `backend/cuckoo.db`；debug env 示例默认是 `cuckoo-debug.db`。
- 密码不会明文存储，数据库里保存的是 bcrypt hash。

如果你想重新 seed 一个全新的 admin，可以换一个 `DB_DSN`：

```bash
cd backend
DB_DSN=cuckoo-local-new.db \
CUCKOO_ADMIN_USERNAME=owner \
CUCKOO_ADMIN_PASSWORD='replace-with-a-strong-password' \
go run ./cmd/server
```

## 3. 启动服务

Backend：

```bash
cd backend
go run ./cmd/server
```

Frontend：

```bash
cd frontend
npm install
npm run dev
```

AI service：

```bash
cd ai-service
npm install
npm run dev
```

## 4. CLI 创建账号

CLI 入口：

```bash
cd backend
go run ./cmd/cuckoo user add --username alice --role player
```

可选角色：

```text
player
admin
```

如果不传 `--password`，系统会生成初始密码并打印一次：

```bash
go run ./cmd/cuckoo user add --username alice --role player
```

输出示例：

```text
created user alice with initial password: ck-abcd-efgh-ijkl
```

如果显式传入密码：

```bash
go run ./cmd/cuckoo user add --username alice --password 'password123' --role player
```

无论哪种方式，数据库只保存 bcrypt hash。

## 5. Web 后台创建账号

admin 登录后访问：

```text
http://localhost:15173/admin/users
```

后台创建账号时：

- 管理员只输入用户名和角色。
- 系统根据 `JWT_SECRET + username` 生成初始密码。
- 初始密码只在创建成功后显示一次。
- 用户拿到初始密码后，应登录并进入 `/account` 修改密码。
- 管理员可以禁用/恢复账号；禁用账号不能登录，也不能创建或加入新房间。
- 管理员可以重置用户密码；重置后会返回一次性新初始密码，数据库仍只保存 bcrypt hash。
- 管理员不能禁用自己，也不能禁用系统中最后一个可用 admin。

用户自助修改密码页面：

```text
http://localhost:15173/account
```

改密码规则：

- 需要输入当前密码。
- 新密码至少 8 个字符。
- 修改后旧密码立即失效。

## 6. 对局规则、归档和草稿预览

字数 unit 规则：

- CJK 字符（中文、日文假名、韩文）每个字符算 1 unit。
- 英文连续词算 1 unit。
- 数字连续串算 1 unit。
- 标点和空白只作为分隔符，不计 unit。

房间设置：

- `maxUnitsPerTurn`：每回合 unit 上限，范围 5-80。
- `turnTimeLimitSeconds`：每回合限时，默认 120 秒，范围 30-600 秒。
- 后端以服务端记录的 turn start 时间为准校验超时，前端倒计时只负责展示。

超时行为：

- 当前回合超时后，后端自动写入一条 0 分系统 skip contribution。
- 房间广播 `game.turn_timeout` 和 `game.turn_changed`。
- 如果这是最后一个回合，会结束游戏并生成归档。

对局归档：

- 游戏结束后写入 `game_results` 和 `game_archives`。
- 每个房间只保存一份完整故事快照。
- 用户在 `/account` 查看最近对局。
- 对局详情页面为 `/games/:roomCode`。

草稿预览：

- 当前玩家输入时，前端通过 WebSocket 发送 `game.draft_update`。
- 其他玩家收到 `game.draft_updated` 后看到临时草稿和光标效果。
- 草稿只存在前端状态，不入库、不计分、不进入归档。

AI 评委：

- AI service 保留 `/completion`，新增 `/judge` stub。
- 后端评分通过 `ScoringService` 扩展点调用 `/judge`。
- AI 调用失败时回退本地占位分，不影响对局结束。

## 7. Debug 构建与发行

生成 debug 构建产物：

```bash
./scripts/debug-build.sh
```

产物目录：

```text
dist-debug/
  backend/cuckoo-server-debug
  backend/cuckoo-cli-debug
  frontend/dist/
  ai-service/dist/server.js
  debug.env.example
  README-debug.md
  cuckoo-debug-linux-amd64.tar.gz
```

运行 debug backend：

```bash
cd dist-debug/backend
set -a
. ../debug.env.example
set +a
./cuckoo-server-debug
```

`dist-debug/` 和 `*.tar.gz` 不提交到 Git。生成后可手动上传 `dist-debug/cuckoo-debug-linux-amd64.tar.gz` 到 GitHub Release。
