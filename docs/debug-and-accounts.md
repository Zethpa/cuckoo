# Debug Ports, CLI, and Account Management

本文档说明 cuckoo 调试端口、admin 初始账号、CLI 用法和账号创建方式。

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

用户自助修改密码页面：

```text
http://localhost:15173/account
```

改密码规则：

- 需要输入当前密码。
- 新密码至少 8 个字符。
- 修改后旧密码立即失效。

## 6. Debug 构建与发行

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
```

运行 debug backend：

```bash
cd dist-debug/backend
set -a
. ../debug.env.example
set +a
./cuckoo-server-debug
```

