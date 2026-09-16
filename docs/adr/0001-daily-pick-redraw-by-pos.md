# 今日摘抄按摘抄位重抽

重抽要保持其余划线不动，所以指定的是当日可见列表里的摘抄位 `pos`，而不是 `bookmarkId`（换完之后旧 id 已不在集合里）。`POST /api/highlights/random` 已被「换一批」占用，因此单独走 `POST /api/highlights/random/redraw`，避免把 path 里的数字误认成书或划线 id。
