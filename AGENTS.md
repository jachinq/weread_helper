# 微信读书助手

个人自用的微信读书笔记与阅读统计工具。前后端分离：Go BFF 持有 API Key，前端只请求本服务 `/api`，不直连微信读书。

凭证来自 [微信读书 Skills API Key](https://weread.qq.com/r/weread-skills)。可写入 `server/.env` 的 `WEREAD_API_KEY`（库中尚无 Key 时会加密入库作为种子），或在前端「设置」页填写。之后以 SQLite `app_settings` 为准。官方接口说明见仓库根目录 `docs/weread_API.md`（Gateway `skill_version` 当前为 `1.0.4`）。

## 当前已实现

### 本地库与同步

- SQLite（`DATABASE_PATH`，默认 `server/data/weread.db`）存笔记、统计快照、书架、应用设置。
- 同步仅由前端手动触发（`POST /api/sync`）。进程启动、定时器、读本地接口都不会自动开任务。
- 设置里的间隔（默认 6h）只作提醒阈值：`/api/sync/status` 的 `stale` 表示距上次成功同步已超过该时长（或从未同步）。进入站点时前端据此引导用户点「同步」。
- 增量：拉完 `/user/notebooks`，仅对计数/`sort`/`readingProgress` 变化或从未拉过笔记的书刷新章节/划线/想法（该书内全量覆盖）。
- 书架每次同步整份覆盖。阅读统计同步写入当前周/月/年/累计；历史年/月/周在统计页切换时按需拉取，不覆盖其它周期。

### 笔记

- 有笔记的书列表：封面、书名、作者、划线数、想法数、阅读进度；`lastSort` 游标分页，支持「加载更多」。
- 单本书笔记详情：按章节分组展示划线原文（`markText`）与想法（`abstract` + `content`）。
- 前端路由：`/notes`、`/notes/:bookId`。

### 首页摘抄

- 服务端按本地日期缓存当日 5 条划线（进程内存）；当日首次 GET 自动抽取，之后返回同一批。
- 「换一批」走 POST 覆盖当日缓存。进程重启后会重新抽。
- 前端按屏宽从这 5 条里展示 3 / 4 / 5 张藏书票（原文、书名、作者、划线日期）。
- 前端路由：`/`。

### 阅读统计

- 周期切换：周 / 月 / 年 / 累计；年、月、周可步进到历史周期。本地无快照时按需向官方拉取并按周期键入库。
- 展示总阅读时长、阅读天数、日均时长；时长按秒换算（后端附加 `*Formatted` 字段）。
- 有 `dailyReadTimes` 时绘制分钟柱状图；有偏好分类则展示标签。
- 前端路由：`/stats`。

### 书架

- 本地 `is_on_shelf` 的书列表：封面、书名、作者、进度、读完/置顶。
- 前端路由：`/shelf`。

### 设置

- 站点名、Gateway、`skill_version`、提醒阈值（原同步间隔）、API Key 写入 `app_settings`。
- API Key 用 AES-256-GCM 加密；主密钥来自 `SETTINGS_ENCRYPT_KEY` 或数据库同目录的 `settings.key`。
- GET 只返回脱敏 `apiKeyMasked`（如 `wrk-****afaf`）；PUT 时 `apiKey` 留空不修改。
- 保存后对当前进程立即生效，无需重启。无 Key 时服务仍可启动，同步会失败直到填入。
- 前端路由：`/settings`。

### 本服务 REST（给前端）

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/notebooks?count=&lastSort=` | 有笔记的书（本地） |
| GET | `/api/highlights/random` | 当日摘抄（无缓存则抽 5 条写入内存） |
| POST | `/api/highlights/random` | 换一批，覆盖当日内存缓存 |
| GET | `/api/books/:bookId` | 书籍信息 + 阅读进度（本地） |
| GET | `/api/books/:bookId/notes` | 聚合章节、划线、想法（本地） |
| GET | `/api/stats?mode=&year=&month=&week=` | 阅读统计快照（本地；年/月/周可指定周期） |
| POST | `/api/stats/fetch?mode=&year=&month=&week=` | 按需从官方拉取并保存该周期统计 |
| GET | `/api/shelf` | 书架（本地） |
| GET | `/api/sync/status` | 同步状态（含 `stale` 提醒阈值） |
| POST | `/api/sync?force=` | 手动触发同步；`force=1` 刷新全部有笔记的书 |
| GET | `/api/settings` | 应用设置（Key 仅脱敏） |
| PUT | `/api/settings` | 更新设置；空 `apiKey` 不改密钥 |

### 已封装的 Gateway 方法

`internal/weread`：`Notebooks`、`BookInfo`、`Progress`、`Chapters`、`Highlights`、`MyReviews`、`ReadStats`、`Shelf`。由 `internal/syncjob` 调用写入 SQLite。

## 明确未做

- 搜索、公开书评、热门划线、推荐
- 多用户登录（单 API Key 个人助手）
- 真正的书签内容导出（官方目前只有数量）

## 架构约定

```
web (Vite :5173)  --/api-->  server (Gin :8080 + SQLite)  --Bearer wrk-*-->  i.weread.qq.com/api/agent/gateway
```

- 所有官方调用：`POST` Gateway，JSON 顶层带 `api_name`、`skill_version` 和业务参数，不要包进 `params`/`body`。
- **不要统一字段大小写**：`/book/info`、`/book/bookmarklist` 用 `bookId`；`/review/list/mine` 必须用 `bookid`。
- `/user/notebooks` 用 `lastSort` 游标，不用 offset。
- `errcode != 0` 或存在 `upgrade_info` 视为失败，不要当成功数据。
- 划线接口名是 `/book/bookmarklist`，返回的是划线不是书签。
- `reviewCount` = 想法，`noteCount` = 划线，`bookmarkCount` = 书签数量。
- 读接口走本地库；写官方只发生在同步任务，以及统计页按需拉取历史年/月/周快照。

## 目录

- `server/cmd/api`：入口
- `server/internal/weread`：Gateway 客户端
- `server/internal/store`：SQLite
- `server/internal/syncjob`：增量同步
- `server/internal/httpapi`：BFF 路由与笔记聚合
- `server/internal/config`：环境变量
- `server/internal/appcfg`：库内设置加载/保存
- `server/internal/secret`：API Key 加解密与脱敏
- `web/src/views`：`HomeView.vue`、`NotesList.vue`、`NoteDetail.vue`、`StatsView.vue`、`ReportView.vue`、`ShelfView.vue`、`SettingsView.vue`
- `web/src/api.ts`：前端请求封装

## 本地运行

```bash
cd server && go run ./cmd/api
cd web && pnpm install && pnpm run dev
```

开发时 Vite 把 `/api` 代理到 `http://127.0.0.1:8080`。
