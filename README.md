# 微信读书助手

个人读书笔记展示与阅读统计。Go 后端代理官方 Agent Gateway，Vite + Vue 3 前端只请求本服务 `/api`。

## 准备

1. 在 [微信读书 Skills API Key](https://weread.qq.com/r/weread-skills) 获取密钥  
2. 复制环境变量：

```bash
copy .env.example server\.env
```

将 `WEREAD_API_KEY` 换成真实值。

## 运行

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

## 接口

- `GET /api/health`
- `GET /api/notebooks?count=&lastSort=`
- `GET /api/books/:bookId`
- `GET /api/books/:bookId/notes`
- `GET /api/stats?mode=weekly|monthly|annually|overall`
