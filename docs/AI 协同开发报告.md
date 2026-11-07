# AI 协同开发报告

## 一、关键 Prompt 记录

### 1. 数据库 Schema 设计

**Prompt 1｜先方案、后落地**

> 「首先这个需求以及设计规范，然后说说你的设计方案」

这是整个项目的起点：先让 AI 读需求、做需求拆解和技术选型，再讨论字段。AI 的方案里确定了两条后续一直沿用的设计——合规判定在签退时执行、结果快照写入 `visit`（保证可审计、可统计）；时间统一以 UTC 存库、前端按 `Asia/Shanghai` 展示。方案同时列出了几个容易漏的点：并发重复签到、时间倒挂、坐标非法、时区、医院坐标基准。

**Prompt 2｜字段级设计**

> 「给出每个数据表的数据字段详细设计」

AI 输出 7 张表的字段、类型、约束、索引与状态枚举。其中两个关键处理是：坐标用 `DECIMAL(9,6)`（精度约 0.11 米）而不是 `FLOAT`，避免距离判定出现精度误差；索引按实际查询路径设计，`(mr_id, planned_at)` 支撑代表按时间查拜访，`(product_id, status, checkout_at)` 支撑产品月度看板。

**Prompt 3｜压缩模型复杂度**

> 「将 call_report 表合入 visit 表」

我的指令是合并。AI 给出的依据是一次拜访只有一条记录，当前 1:1 场景合并是合理的；并同步列出影响面：表数 7 → 6、`visit` 新增 5 个记录字段、`materials_distributed` 用 NULL 区分「未填记录 / 未派发 / 已派发」、报告接口少一次关联查询、看板 SQL 不变。

**Prompt 4｜把质量要求前置**

> 「现在基于这个文档，开始后端模块开发。注意每个函数需要有对应的单元测试，每个接口需要进行端到端测试」

这条 Prompt 决定了产出形态，而不只是「写完再补测试」：

- `compliance`、`visitno`、`apperr`、`response`、`model` 五个包覆盖率 100%，`repository` 97.0%、`middleware` 96.6%、`handler` 88.4%、`service` 81.7%；
- 合规边界按 299/300/301 秒、499.99/500/500.01 米逐条覆盖；
- E2E 基于真实 Hertz + MySQL 覆盖每个接口的成功与失败路径；
- 联调阶段 52 项 API 断言全部通过，实测停留 120 秒命中 `SHORT_DURATION`、偏离 700.53 米命中 `LOCATION_DEVIATION`、停留 300 秒且偏离 0 米为正常（`abnormal_reasons=[]`）。

## 二、AI 纠偏亮点案例

### 列表接口签到/签退时间未回填

**问题现象**

后端模块开发完成后进入测试验收。会话记录中对这次缺陷的原话是：**「测试中发现并修复了一个真实缺陷：列表接口的 `checkin_at`/`checkout_at` 因列名与结构体字段映射不一致，导致签到签退时间未回填」**。

**定位过程**（结合代码分析）

1. 先确认根因方向：不是写库失败，而是列表查询读出来的字段没有绑定到结构体——同一份数据，其他路径能读到，列表接口读不到。
2. 读列表查询实现：`ListVisits` 用的是 `Table("visit AS v")` + 手写 `Select` + `Scan`（[gorm.go](file:///Users/tracyswang/Desktop/go_projects/mrvs/backend/internal/repository/gorm.go#L123-L151)）。
3. 对照字段与列名：结构体字段是 `CheckInAt`，GORM 默认命名策略对应 `check_in_at`，而 AI 手写的 SQL 选的是数据库原始列 `checkin_at`。`Scan` 按列名匹配，匹配不上就留空，**不报错、不告警**，所以问题只表现为「字段没值」。

**修复方式**

不改表结构、不改接口，只在 `Select` 中给列显式别名，让映射对齐（[gorm.go](file:///Users/tracyswang/Desktop/go_projects/mrvs/backend/internal/repository/gorm.go#L133-L137)）：

```sql
v.checkin_at AS check_in_at, v.checkout_at AS check_out_at
```

## 全程AI 会话导出 + git history
- 会话记录：[会话记录](./会话记录.md)
- git history：[git history](https://github.com/tracywangsw/mrvs/commits/main)   

## 应用 Demo 录屏
