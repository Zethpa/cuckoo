# Red Hat 系统部署说明

适用于 CentOS、Rocky Linux、AlmaLinux、RHEL、Fedora Server 等红帽系系统。

本文不使用 Debian/Ubuntu 常见的 `www-data` 用户，而是创建专用系统用户 `cuckoo`。

## 1. 创建运行用户和目录

```bash
sudo useradd --system --home-dir /var/lib/cuckoo --shell /sbin/nologin cuckoo

sudo mkdir -p /opt/cuckoo /etc/cuckoo /var/lib/cuckoo
sudo chown -R cuckoo:cuckoo /var/lib/cuckoo
sudo chmod 755 /var/lib/cuckoo
```

如果用户已存在，`useradd` 会报错，可以忽略并继续。

## 2. 解压 release 包

假设 release 包名为：

```text
cuckoo-debug-linux-amd64.tar.gz
```

部署到 `/opt/cuckoo`：

```bash
sudo tar -xzf cuckoo-debug-linux-amd64.tar.gz -C /opt/cuckoo --strip-components=1
sudo chmod +x /opt/cuckoo/backend/cuckoo-server-debug
sudo chmod +x /opt/cuckoo/backend/cuckoo-cli-debug
```

确认：

```bash
find /opt/cuckoo -maxdepth 3 -type f | sort
```

## 3. 配置环境变量

```bash
sudo cp /opt/cuckoo/debug.env.example /etc/cuckoo/cuckoo.env
sudo nano /etc/cuckoo/cuckoo.env
```

建议内容：

```bash
CUCKOO_ENV=production
HTTP_ADDR=:18081
FRONTEND_URL=https://cuckoo.xxxx.xx
DB_DRIVER=sqlite
DB_DSN=/var/lib/cuckoo/cuckoo.db
JWT_SECRET=replace-with-a-long-random-secret
CUCKOO_ADMIN_USERNAME=admin
CUCKOO_ADMIN_PASSWORD=replace-with-a-strong-password
AI_SERVICE_URL=http://localhost:18787
```

生成随机 secret：

```bash
openssl rand -base64 48
```

权限：

```bash
sudo chown root:cuckoo /etc/cuckoo/cuckoo.env
sudo chmod 640 /etc/cuckoo/cuckoo.env
```

## 4. 安装 systemd 服务

如果 release 包里包含 `deploy/systemd`：

```bash
sudo cp /opt/cuckoo/deploy/systemd/cuckoo-backend.service /etc/systemd/system/cuckoo-backend.service
sudo cp /opt/cuckoo/deploy/systemd/cuckoo-ai.service /etc/systemd/system/cuckoo-ai.service
```

如果没有，就手动创建：

```bash
sudo nano /etc/systemd/system/cuckoo-backend.service
```

内容：

```ini
[Unit]
Description=Cuckoo Backend
After=network.target

[Service]
Type=simple
User=cuckoo
Group=cuckoo
WorkingDirectory=/opt/cuckoo/backend
EnvironmentFile=/etc/cuckoo/cuckoo.env
ExecStart=/opt/cuckoo/backend/cuckoo-server-debug
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

AI service：

```bash
sudo nano /etc/systemd/system/cuckoo-ai.service
```

内容：

```ini
[Unit]
Description=Cuckoo AI Service Stub
After=network.target

[Service]
Type=simple
User=cuckoo
Group=cuckoo
WorkingDirectory=/opt/cuckoo/ai-service
Environment=PORT=18787
Environment=HOST=127.0.0.1
ExecStart=/usr/bin/node /opt/cuckoo/ai-service/dist/server.js
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

## 5. 安装 AI service 依赖

需要 Node.js：

```bash
node --version
npm --version
```

安装生产依赖：

```bash
cd /opt/cuckoo/ai-service
sudo npm install --omit=dev
sudo chown -R cuckoo:cuckoo /opt/cuckoo/ai-service/node_modules
```

## 6. 启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now cuckoo-backend
sudo systemctl enable --now cuckoo-ai
```

检查：

```bash
sudo systemctl status cuckoo-backend --no-pager -l
sudo systemctl status cuckoo-ai --no-pager -l
```

日志：

```bash
sudo journalctl -u cuckoo-backend -f
sudo journalctl -u cuckoo-ai -f
```

## 7. Caddy 配置

安装 Caddy 后编辑：

```bash
sudo nano /etc/caddy/Caddyfile
```

替换域名：

```caddyfile
cuckoo.xxxx.xx {
  encode zstd gzip

  handle /api/* {
    reverse_proxy 127.0.0.1:18081
  }

  handle {
    root * /opt/cuckoo/frontend/dist
    try_files {path} /index.html
    file_server
  }
}
```

验证并重载：

```bash
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

## 8. 常见排错

查看 backend 失败原因：

```bash
sudo journalctl -xeu cuckoo-backend.service --no-pager -n 100
```

检查 `cuckoo` 用户：

```bash
id cuckoo
```

检查端口：

```bash
ss -ltnp | grep -E ':18081|:18787'
```

手动运行 backend：

```bash
cd /opt/cuckoo/backend
set -a
. /etc/cuckoo/cuckoo.env
set +a
sudo -u cuckoo ./cuckoo-server-debug
```
