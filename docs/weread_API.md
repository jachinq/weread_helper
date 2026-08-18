# 微信读书 API 接口文档

**API 版本：1.0.4**

官方项目：

[Tencent/WeChatReading](https://github.com/Tencent/WeChatReading)

---

## 1. 基础信息

### 1.1 API Gateway

```text
POST https://i.weread.qq.com/api/agent/gateway
```

所有接口实际上都通过这个地址调用。

通过 JSON 中的：

```json
{
  "api_name": "/book/info"
}
```

指定实际 API。

官方明确要求每次请求都携带：

```json
"skill_version": "1.0.4"
```

并且业务参数必须直接放在 JSON 顶层，不能放进 `params`、`data`、`body` 等对象中。([GitHub][1])

---

## 2. 鉴权

请求 Header：

```http
Authorization: Bearer wrk-xxxxxxxx
Content-Type: application/json
```

API Key：

```text
wrk-xxxxxxxx
```

官方说明 API Key 与微信读书用户身份绑定，因此需要用户身份的接口不需要自己传 `vid`。([GitHub][1])

API Key 获取入口：

[微信读书 Skills API Key](https://weread.qq.com/r/weread-skills)

---

# 3. Python 基础调用

建议先封装一个统一客户端。

```python
import requests


class WeReadAPI:
    BASE_URL = "https://i.weread.qq.com/api/agent/gateway"
    VERSION = "1.0.4"

    def __init__(self, api_key: str):
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        })

    def call(self, api_name: str, **params):
        payload = {
            "api_name": api_name,
            **params,
            "skill_version": self.VERSION,
        }

        response = self.session.post(
            self.BASE_URL,
            json=payload,
            timeout=30,
        )

        response.raise_for_status()

        data = response.json()

        if data.get("errcode", 0) != 0:
            raise RuntimeError(
                data.get("errmsg", "微信读书 API 调用失败")
            )

        return data
```

使用：

```python
api = WeReadAPI("wrk-xxxxxxxx")

result = api.call(
    "/store/search",
    keyword="三体",
    scope=10,
    count=10,
)

print(result)
```

---

# 4. 接口总览

当前官方 Skill 文档覆盖以下接口：

| 分类   | API                   |
| ---- | --------------------- |
| 搜索   | `/store/search`       |
| 书籍   | `/book/info`          |
| 书籍   | `/book/chapterinfo`   |
| 书籍   | `/book/getprogress`   |
| 书架   | `/shelf/sync`         |
| 阅读统计 | `/readdata/detail`    |
| 笔记   | `/user/notebooks`     |
| 笔记   | `/book/bookmarklist`  |
| 笔记   | `/review/list/mine`   |
| 热门划线 | `/book/underlines`    |
| 热门划线 | `/book/bestbookmarks` |
| 热门划线 | `/book/readreviews`   |
| 想法详情 | `/review/single`      |
| 公开书评 | `/review/list`        |
| 推荐   | `/book/recommend`     |
| 相似书  | `/book/similar`       |

官方还提供：

```json
{
  "api_name": "/_list"
}
```

用于查询当前 Gateway 支持的接口及参数定义。这个接口非常重要，**以后官方 API 更新时建议优先调用它检查接口列表**。([GitHub][1])

---

# 5. 搜索书籍

## `/store/search`

搜索微信读书内容。

### 请求参数

| 参数        | 类型     | 必填 | 说明         |
| --------- | ------ | -: | ---------- |
| `keyword` | string |  ✅ | 搜索关键词      |
| `scope`   | int    |  ❌ | 搜索类型       |
| `maxIdx`  | int    |  ❌ | 翻页位置，默认 0  |
| `count`   | int    |  ❌ | 每页数量，默认 15 |

### scope

| scope | 类型          |
| ----: | ----------- |
|   `0` | 全部          |
|  `10` | 电子书         |
|  `16` | 网文小说        |
|  `14` | 微信听书/有声书/播客 |
|   `6` | 作者          |
|  `12` | 全文搜索        |
|  `13` | 书单          |
|   `2` | 公众号         |
|   `4` | 文章          |

找普通电子书建议：

```json
{
  "api_name": "/store/search",
  "keyword": "三体",
  "scope": 10,
  "count": 10,
  "skill_version": "1.0.4"
}
```

官方特别说明：如果明确是在“找书”，使用 `scope=10`；不要把返回结果中的 `scope=17` 当成请求参数。

### 返回核心字段

```text
results[]
    title
    scope
    scopeCount
    currentCount
    books[]
        searchIdx
        bookInfo
            bookId
            deepLink
            title
            author
            cover
            intro
            publisher
            category
            payType
            price
            soldout
        readingCount
        newRating
        newRatingCount
        newRatingDetail
```

评分：

```text
newRating = 0 ~ 100
```

例如：

```text
96 = 96分
```

---

# 6. 获取书籍详情

## `/book/info`

### 请求

```json
{
  "api_name": "/book/info",
  "bookId": "330006912",
  "skill_version": "1.0.4"
}
```

### 参数

| 参数       | 类型     | 必填 |
| -------- | ------ | -: |
| `bookId` | string |  ✅ |

### 返回

| 字段                | 说明    |
| ----------------- | ----- |
| `bookId`          | 书籍 ID |
| `deepLink`        | 阅读链接  |
| `title`           | 书名    |
| `author`          | 作者    |
| `translator`      | 译者    |
| `cover`           | 封面    |
| `intro`           | 简介    |
| `category`        | 分类    |
| `publisher`       | 出版社   |
| `publishTime`     | 出版时间  |
| `isbn`            | ISBN  |
| `wordCount`       | 总字数   |
| `newRating`       | 百分制评分 |
| `newRatingCount`  | 评分人数  |
| `newRatingDetail` | 评分分布  |

([GitHub][2])

---

# 7. 获取章节目录

## `/book/chapterinfo`

### 请求

```json
{
  "api_name": "/book/chapterinfo",
  "bookId": "330006912",
  "skill_version": "1.0.4"
}
```

### 返回核心字段

```text
bookId
synckey
chapterUpdateTime
chapters[]
```

章节：

| 字段            | 说明      |
| ------------- | ------- |
| `chapterUid`  | 章节 UID  |
| `chapterIdx`  | 章节序号    |
| `title`       | 标题      |
| `wordCount`   | 字数      |
| `level`       | 目录层级    |
| `updateTime`  | 更新时间    |
| `price`       | 章节价格    |
| `paid`        | 是否购买    |
| `isMPChapter` | 是否公众号章节 |
| `anchors`     | 子标题/锚点  |

其中：

```text
chapterUid
```

非常重要。

后续查询划线、章节热门划线等接口会使用它。([GitHub][2])

---

# 8. 获取阅读进度

## `/book/getprogress`

### 请求

```json
{
  "api_name": "/book/getprogress",
  "bookId": "330006912",
  "skill_version": "1.0.4"
}
```

### 返回

```text
book
    chapterUid
    chapterOffset
    progress
```

其中：

```text
progress = 0 ~ 100
```

注意：

```text
1 = 1%
100 = 100%
```

不是 `0.01 ~ 1.0`。([GitHub][2])

---

# 9. 获取书架

## `/shelf/sync`

这个接口非常适合做**同步微信读书书架**。

### 请求

不需要参数：

```json
{
  "api_name": "/shelf/sync",
  "skill_version": "1.0.4"
}
```

用户身份由 API Key 自动确定。

### 返回

主要有：

```text
books[]
albums[]
mp
archive[]
bookCount
```

### books[]

电子书：

```text
bookId
deepLink
title
author
cover
category
readUpdateTime
finishReading
updateTime
isTop
secret
```

### albums[]

有声书/专辑：

```text
albumInfo
    albumId
    name
    authorName
    cover
    trackCount
    finishStatus
    finish
    payType
    intro
    updateTime

albumInfoExtra
    secret
    lecturePaid
    lectureReadUpdateTime
    isTop
```

### 一个很容易踩坑的地方

官方明确规定：

```text
书架总条目 =
books.length
+
albums.length
+
(mp 非空 ? 1 : 0)
```

因为：

* `books` = 电子书
* `albums` = 有声书/专辑
* `mp` = 文章收藏入口

不能只统计：

```python
len(books)
```



---

# 10. 阅读统计

## `/readdata/detail`

### 参数

| 参数         | 类型     | 说明    |
| ---------- | ------ | ----- |
| `mode`     | string | 统计周期  |
| `baseTime` | int    | 基准时间戳 |

### mode

```text
weekly
monthly
annually
overall
```

例如查询本月：

```json
{
  "api_name": "/readdata/detail",
  "mode": "monthly",
  "skill_version": "1.0.4"
}
```

### 主要返回

```text
baseTime
readTimes
dailyReadTimes
readDays
totalReadTime
dayAverageReadTime
compare
readLongest
readStat
preferCategory
preferTime
preferAuthor
preferPublisher
preferCp
readRate
wrReadTime
wrListenTime
rank
medals
preferBooks
yearReport
```

其中：

```text
totalReadTime
```

单位是**秒**。

例如：

```text
3600 = 1小时
5400 = 1小时30分钟
```

不要把它当成分钟。

---

# 11. 获取所有有笔记的书

## `/user/notebooks`

这是做“**我的微信读书笔记同步**”最重要的接口之一。

### 请求

```json
{
  "api_name": "/user/notebooks",
  "count": 100,
  "skill_version": "1.0.4"
}
```

### 参数

| 参数         | 类型  | 说明         |
| ---------- | --- | ---------- |
| `count`    | int | 每页数量，默认 20 |
| `lastSort` | int | 下一页游标      |

### 返回

```text
totalBookCount
totalNoteCount
hasMore

books[]
    bookId
    book
    reviewCount
    noteCount
    bookmarkCount
    readingProgress
    markedStatus
    sort
```

这里有个重要区别：

```text
reviewCount = 想法/点评
noteCount = 划线
bookmarkCount = 书签
```

总笔记数：

```python
reviewCount + noteCount + bookmarkCount
```

而不是直接使用 `noteCount`。

### 分页

第一次：

```json
{
  "api_name": "/user/notebooks",
  "count": 100,
  "skill_version": "1.0.4"
}
```

如果：

```json
"hasMore": 1
```

取最后一项：

```text
books[-1].sort
```

下一次：

```json
{
  "api_name": "/user/notebooks",
  "count": 100,
  "lastSort": 1234567890,
  "skill_version": "1.0.4"
}
```

**不要使用 `offset` / `limit`。**

这是游标分页。

---

# 12. 获取一本书的划线

## `/book/bookmarklist`

虽然名字叫 `bookmarklist`，但官方当前接口已经过滤掉真正的书签。

实际返回的是：

> **划线内容**

### 请求

```json
{
  "api_name": "/book/bookmarklist",
  "bookId": "330006912",
  "skill_version": "1.0.4"
}
```

### 返回

```text
updated[]
    bookmarkId
    bookId
    chapterUid
    markText
    createTime
    type
    range
    colorStyle

chapters[]
    chapterUid
    chapterIdx
    title

book
```

核心字段：

```text
markText
```

就是用户划线的原文。

官方特别说明：真正的书签目前只有数量，没有提供可导出的书签内容。

---

# 13. 获取自己的想法/点评

## `/review/list/mine`

### 请求

```json
{
  "api_name": "/review/list/mine",
  "bookid": "330006912",
  "count": 100,
  "skill_version": "1.0.4"
}
```

注意这里官方参数写的是：

```text
bookid
```

而不是：

```text
bookId
```

程序实现时建议严格按照官方文档。

### 参数

| 参数        | 类型     | 说明    |
| --------- | ------ | ----- |
| `bookid`  | string | 书籍 ID |
| `synckey` | int    | 分页游标  |
| `count`   | int    | 每页数量  |

### 返回

```text
reviews[]
    review
        reviewId
        content
        abstract
        range
        chapterUid
        chapterIdx
        createTime
        star
        chapterName
        isFinish

totalCount
hasMore
synckey
```

其中：

* `content` = 想法/点评
* `abstract` = 对应的划线原文
* `range` = 原文位置
* `chapterUid` = 所属章节
* `chapterName` = 章节名称

因此非常适合把数据转换成：

```text
原文
↓
我的想法
```

的结构。

---

# 14. 获取章节划线热度

## `/book/underlines`

获取某一章节中各种划线的热度。

### 请求

```json
{
  "api_name": "/book/underlines",
  "bookId": "330006912",
  "chapterUid": 123456,
  "skill_version": "1.0.4"
}
```

### 参数

| 参数           | 类型     | 必填 |
| ------------ | ------ | -: |
| `bookId`     | string |  ✅ |
| `chapterUid` | int    |  ✅ |
| `synckey`    | int    |  ❌ |

### 返回

```text
underlines[]
    range
    count
    score
    type
```

注意：

**这个接口不返回划线文本。**

它主要告诉你：

```text
这段文字有多少人划线
```

如果需要原文，应使用 `/book/bestbookmarks`。

---

# 15. 获取全书热门划线

## `/book/bestbookmarks`

这是非常有用的一个接口。

### 请求

```json
{
  "api_name": "/book/bestbookmarks",
  "bookId": "330006912",
  "chapterUid": 0,
  "skill_version": "1.0.4"
}
```

### 参数

| 参数           | 说明             |
| ------------ | -------------- |
| `bookId`     | 书籍 ID          |
| `chapterUid` | 章节 UID，0 表示整本书 |
| `synckey`    | 同步 key         |

官方说明当前固定返回热门划线前 20 条，不支持分页。

### 返回

```text
items[]
    bookId
    userVid
    bookmarkId
    chapterUid
    range
    markText
    totalCount
    simplifiedRange
    traditionalRange

chapters[]
    chapterUid
    chapterIdx
    title
```

其中：

```text
markText
```

就是热门划线原文。

```text
totalCount
```

就是划线人数。

---

# 16. 获取热门划线下的想法

## `/book/readreviews`

可以理解为：

```text
热门划线
   ↓
这条划线下面有哪些读者想法？
```

### 请求

```json
{
  "api_name": "/book/readreviews",
  "bookId": "330006912",
  "chapterUid": 123456,
  "reviews": [
    {
      "range": "393-401",
      "count": 20,
      "maxIdx": 0,
      "synckey": 0
    }
  ],
  "skill_version": "1.0.4"
}
```

### 参数

`reviews` 是一个数组：

```text
reviews[].range
reviews[].maxIdx
reviews[].count
reviews[].synckey
```

### 返回

```text
reviews[]
    range
    totalCount
    hasMore
    maxIdx
    synckey
    pageReviews[]
        reviewId
        review
            abstract
            content
            range
            createTime
            author
```

官方建议的调用链：

```text
/book/bestbookmarks
        ↓
获取热门划线 range
        ↓
/book/readreviews
        ↓
获取该划线下面的想法
```



---

# 17. 获取单条想法详情

## `/review/single`

### 请求

```json
{
  "api_name": "/review/single",
  "reviewId": "123456789",
  "skill_version": "1.0.4"
}
```

可选：

```text
commentsCount
commentsDirection
likesCount
likesDirection
synckey
```

### 返回

```text
reviewId
review
htmlContent
synckey
```

主要用途是进一步获取某条想法的：

* 完整内容
* 评论
* 点赞
* 作者信息



---

# 18. 获取公开书评

## `/review/list`

注意这个和：

```text
/review/list/mine
```

完全不同。

`/review/list`：

> 所有读者对这本书的公开点评

`/review/list/mine`：

> 当前用户自己的想法/点评

### 请求

```json
{
  "api_name": "/review/list",
  "bookId": "330006912",
  "reviewListType": 0,
  "count": 20,
  "maxIdx": 0,
  "synckey": 0,
  "skill_version": "1.0.4"
}
```

### reviewListType

|   值 | 含义 |
| --: | -- |
| `0` | 全部 |
| `1` | 推荐 |
| `2` | 不行 |
| `3` | 最新 |
| `4` | 一般 |

### 返回

```text
synckey
reviewsCnt
recentTotalCnt
reviewsHasMore
reviewsHas5Star
reviewsHas1Star
reviewsHasRecent

reviews[]
    idx
    review
        reviewId
        review
            content
            htmlContent
            star
            isFinish
            createTime
            chapterName
            author
            book
```

这里的评分：

```text
20 = 1星
40 = 2星
60 = 3星
80 = 4星
100 = 5星
```



---

# 19. 个性化推荐

## `/book/recommend`

对应微信读书的：

> 为你推荐

### 请求

```json
{
  "api_name": "/book/recommend",
  "count": 12,
  "maxIdx": 0,
  "skill_version": "1.0.4"
}
```

### 返回

```text
books[]
    bookId
    deepLink
    title
    author
    cover
    intro
    category
    reason
    readingCount
    searchIdx
    newRating
    newRatingCount
    newRatingDetail
    price
    payType
    type
```

分页时：

```text
上一页最后一个 searchIdx
        ↓
下一次 maxIdx
```



---

# 20. 相似书推荐

## `/book/similar`

根据一本书推荐类似书籍。

### 请求

```json
{
  "api_name": "/book/similar",
  "bookId": "330006912",
  "count": 12,
  "maxIdx": 0,
  "skill_version": "1.0.4"
}
```

### 参数

| 参数          | 必填 | 说明     |
| ----------- | -: | ------ |
| `bookId`    |  ✅ | 书籍 ID  |
| `count`     |  ✅ | 每页数量   |
| `maxIdx`    |  ✅ | 翻页位置   |
| `sessionId` |  ❌ | 后续分页使用 |

### 返回

```text
booksimilar
    sessionId
    books[]
        idx
        book
            bookInfo
```

下一页：

```text
maxIdx = 上一页最后一个 books[].idx
```

并带上：

```text
sessionId
```

官方特别强调 `count` 和 `maxIdx` 实际调用时都必须显式传入。

---

# 21. 一个完整的程序调用流程

如果你的目标是做一个**微信读书同步程序**，我建议采用下面的结构：

```text
                    微信读书
                       │
                       │ API Key
                       ▼
             Agent API Gateway
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
     搜索书籍        用户数据        公共数据
        │              │              │
 /store/search    /shelf/sync      /review/list
 /book/info       /user/notebooks  /book/bestbookmarks
 /book/chapterinfo /review/list/mine /book/recommend
 /book/getprogress /bookmarklist   /book/similar
```

如果你是为了**自己的数据库同步**，最值得优先实现的是：

```text
1. /shelf/sync
2. /user/notebooks
3. /book/bookmarklist
4. /review/list/mine
5. /book/info
6. /book/chapterinfo
7. /book/getprogress
8. /readdata/detail
```

这样基本可以构建：

```text
书架
├── 书籍
│   ├── 基本信息
│   ├── 阅读进度
│   ├── 章节
│   ├── 划线
│   └── 想法
│
├── 有声书
│
└── 阅读统计
```

---

## 22. 特别值得注意的几个问题

### ① 不要把 Skill 当成 MCP

你的程序**不需要安装 Skill，也不需要接入大模型**。

直接：

```text
HTTP POST
    ↓
https://i.weread.qq.com/api/agent/gateway
```

即可。

Skill Markdown 本质上可以看作**这套 API 的官方调用说明和工作流文档**。([GitHub][1])

### ② `skill_version` 必须跟着版本变化

当前：

```text
1.0.4
```

以后官方升级后可能变成：

```text
1.0.5
```

官方要求，如果接口返回：

```json
{
  "upgrade_info": {
    "message": "..."
  }
}
```

应该先处理升级提示，而不是继续执行原请求。([GitHub][1])

### ③ `bookId` 和 `bookid` 不要自行统一

例如：

```text
/book/info
bookId
```

而：

```text
/review/list/mine
bookid
```

这是官方当前文档中的实际差异，程序最好严格按照各接口参数定义处理。

### ④ 目前不能把它理解成传统 REST API

虽然调用方式是 HTTP API，但它的设计是：

```text
一个 Gateway
+
api_name
+
API Key
+
skill_version
```

而不是：

```text
GET /api/books/{id}
GET /api/books/{id}/chapters
```

这种传统 REST 设计。

### ⑤ 最重要的是 `/ _list`

建议你的程序启动时可以增加一个**API 能力探测机制**：

```python
api.call("/_list")
```

将当前服务端支持的 API 保存下来。

这样以后腾讯增加接口，你不一定需要立即修改程序核心逻辑。

---

## 23. 推荐的客户端封装

如果你准备正式开发，我建议不要让业务代码直接写：

```python
api.call("/book/info", bookId=book_id)
```

而是进一步封装：

```python
class WeReadClient:

    def search_books(self, keyword, count=10):
        ...

    def get_book(self, book_id):
        ...

    def get_chapters(self, book_id):
        ...

    def get_progress(self, book_id):
        ...

    def get_shelf(self):
        ...

    def get_notebooks(self, ...):
        ...

    def get_highlights(self, book_id):
        ...

    def get_my_reviews(self, book_id):
        ...

    def get_reading_stats(self, mode="monthly"):
        ...

    def get_public_reviews(self, book_id):
        ...
```

这样以后如果腾讯修改 Gateway 或参数，只需要修改底层 Client。



[1]: https://github.com/Tencent/WeChatReading/blob/main/skills/SKILL.md "WeChatReading/skills/SKILL.md at main · Tencent/WeChatReading · GitHub"
[2]: https://github.com/Tencent/WeChatReading/blob/main/skills/book.md?utm_source=chatgpt.com "WeChatReading/skills/book.md at main · Tencent/WeChatReading · GitHub"
