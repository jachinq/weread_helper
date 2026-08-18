# 微信读书助手

个人读书笔记展示与阅读统计。Go 后端代理官方 Agent Gateway，Vite + Vue 3 前端只请求本服务 `/api`。

## 准备

1. 在 [微信读书 Skills API Key](https://weread.qq.com/r/weread-skills) 获取密钥  
2. 复制环境变量：

```bash
copy .env.example server\.env
```

将 `WEREAD_API_KEY` 换成真实值（也可启动后在「设置」页填写；首次会把 env 里的 Key 加密写入本地库）。

## 本地开发

后端（默认 `:8080`）：

```bash
cd server
go run ./cmd/api
```

前端（默认 `:5173`，`/api` 代理到后端）：

```bash
cd web
pnpm install
pnpm run dev
```

浏览器打开 `http://127.0.0.1:5173`。

## Windows 桌面（Tauri 2 便携）

代码在仓库根目录 [`desktop/`](desktop/)。壳程序启动时拉起 Go sidecar（只监听 `127.0.0.1` 随机端口），退出壳或托盘「退出」时结束 sidecar。SQLite、`settings.key`、WebView 缓存写在 **exe 同级 `data/`**，不使用用户文档/默认 AppData（WebView2 Runtime 仍由系统安装）。

依赖：Rust、Go、pnpm、本机 WebView2。

```bash
cd desktop
pnpm install
pnpm run sidecar
pnpm run dev
```

打包 NSIS 安装器（默认每用户安装，目录可写；也可把构建产物整夹拷走当便携包）：

```bash
cd desktop
pnpm run build
```

产物一般在 `desktop/src-tauri/target/release/bundle/nsis/`。便携使用时请保持 exe、sidecar（`weread-helper.exe`）与 `resources/web` 相对位置，数据在同目录 `data/`。

托盘菜单：显示主窗口、开机自启、退出。关闭窗口会隐藏到托盘。开机自启写入当前用户 Run 项（指向当时的 exe 绝对路径，移动文件夹后需重新勾选）。

## Docker 部署

镜像为多阶段构建：Node 打前端、Go 静态编译后端（`CGO_ENABLED=0` + 去符号），最终仅保留 Alpine 运行层（CA 证书、时区、`su-exec`）。容器内由同一进程提供 `/api` 与前端静态文件。SQLite 与加密密钥文件写在数据卷 `/data`。

### 使用 Compose（推荐）

在仓库根目录创建 `.env`（不要提交）：

```env
WEREAD_API_KEY=wrk-xxxxxxxx
# 可选：固定 32 字节密钥的 64 位 hex。不填则首次启动写入 /data/settings.key
# SETTINGS_ENCRYPT_KEY=
```

构建默认走国内镜像：Alpine apk（阿里云）、npm/pnpm（npmmirror）、Go modules（goproxy.cn）。直接 `docker build` 同样使用这些默认值。

启动：

```bash
docker compose up -d --build
```

切回官方源（海外或镜像异常时）：

```bash
docker compose build \
  --build-arg ALPINE_MIRROR=https://dl-cdn.alpinelinux.org/alpine \
  --build-arg NPM_REGISTRY=https://registry.npmjs.org \
  --build-arg GOPROXY=https://proxy.golang.org,direct \
  --build-arg GOSUMDB=sum.golang.org
```

或在根目录 `.env` 里覆盖 `ALPINE_MIRROR` / `NPM_REGISTRY` / `GOPROXY` / `GOSUMDB`。

浏览器打开 `http://127.0.0.1:8080`。API Key 也可在站点「设置」页填写。

常用命令：

```bash
docker compose logs -f
docker compose ps
docker compose down
```

数据保存在 Docker 卷 `weread-data`。卸载容器但保留数据：`docker compose down`。连同数据一起删除：`docker compose down -v`。

### 仅构建 / 运行镜像

```bash
docker build -t weread-helper:latest .
docker run --name weread -d -p 8080:8080 \
  -e WEREAD_API_KEY=wrk-xxxxxxxx \
  -e TZ=Asia/Shanghai \
  -v weread-data:/data \
  weread-helper:latest
```

### 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `WEREAD_API_KEY` | 空 | Skills API Key；库中尚无 Key 时作为种子写入 |
| `SETTINGS_ENCRYPT_KEY` | 空 | AES-256 主密钥（64 位 hex）。空则使用 `/data/settings.key` |
| `DATABASE_PATH` | `/data/weread.db` | SQLite 路径 |
| `WEB_DIR` | `/app/web` | 前端静态目录；为空则不托管页面 |
| `LISTEN_ADDR` | `:8080` | 监听地址 |
| `TZ` | `Asia/Shanghai` | 时区（影响当日摘抄按本地日期缓存） |
| `SKILL_VERSION` | `1.0.4` | Gateway skill 版本 |
| `GATEWAY_URL` | 官方 Gateway | Agent Gateway 地址 |
| `SYNC_INTERVAL` | `6h` | 同步过期提醒阈值 |

换机或重建容器时，若要继续解密已有库中的 API Key，请一并迁移数据卷，或固定 `SETTINGS_ENCRYPT_KEY`。

构建参数（仅 `docker build` / `compose build`，默认国内源）：

| 参数 | 默认 | 说明 |
|------|------|------|
| `ALPINE_MIRROR` | `https://mirrors.aliyun.com/alpine` | Alpine apk |
| `NPM_REGISTRY` | `https://registry.npmmirror.com` | pnpm / corepack |
| `GOPROXY` | `https://goproxy.cn,direct` | Go 模块代理 |
| `GOSUMDB` | `sum.golang.google.cn` | Go 校验和数据库 |

## 接口

- `GET /api/health`
- `GET /api/notebooks?count=&lastSort=`
- `GET /api/books/:bookId`
- `GET /api/books/:bookId/notes`
- `GET /api/stats?mode=weekly|monthly|annually|overall`
- `GET /api/shelf`
- `GET /api/sync/status`
- `POST /api/sync?force=`
- `GET /api/settings`
- `PUT /api/settings`
