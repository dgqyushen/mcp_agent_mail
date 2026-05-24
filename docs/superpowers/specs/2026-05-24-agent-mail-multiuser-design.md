# agent-mail 多用户 + Token 管理 设计文档

## 目标

将 agent-mail 从单用户改为多用户系统，支持：
- 管理员通过密码登录面板创建用户、生成/刷新 token
- 用户通过 token 访问 MCP 服务，管理自己的邮箱源（provider）
- Token 刷新后旧 token 立即失效
- 前端页面嵌入 Go 二进制

## 架构

```
/main.go
  ├── /mcp  → authMiddleware(查tokens表) → MCP Server (20工具)
  └── /admin → sessionMiddleware(cookie)  → Admin Handler (Go template)
                    ↓
  store/sqlite/ (users, tokens, mailboxes+user_id)
       ↓
  service/ (UserService 新增)
       ↓
  mcp/ (中间件改造)  +  web/ (模板+handler)
```

所有代码嵌入单一二进制，`docker pull` 即可部署。

## 数据库变更

### 新增 users 表

```sql
CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now'))
);
```

### 新增 tokens 表

```sql
CREATE TABLE tokens (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,       -- SHA256(完整token)
    prefix     TEXT NOT NULL,              -- "atm-***" 用于列表展示
    created_at TEXT DEFAULT (datetime('now')),
    is_active  INTEGER DEFAULT 1
);
```

### mailboxes 表变更

添加 `user_id INTEGER NOT NULL REFERENCES users(id)` 字段，实现用户间邮箱隔离。

## Token 格式

- 格式：`atm-` + 32 位 hex 随机字符
- 示例：`atm-a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6000ff`
- 数据库仅存 SHA256 hash，完整 token 只在创建/刷新时展示一次
- MCP 客户端通过 `X-Agent-Mail-Token` header 传递

## Admin 密码

优先级：环境变量 > 自动生成

- `ADMIN_PASSWORD` 环境变量存在 → 使用该密码
- 不存在 → 首次启动时自动生成随机密码，bcrypt hash 存入 settings 表，明文打印到 stderr
- Admin 登录后设置 session cookie（HTTP-only, SameSite）

## 前端页面

全部使用 Go `embed` + `html/template` + 内联 CSS，无外部框架。

| 路由 | 页面 | 功能 |
|------|------|------|
| `/admin/login` | 登录页 | admin 密码输入 |
| `/admin/users` | 用户列表 | 列出所有用户，创建新用户 |
| `/admin/users/{id}` | 用户详情 | 查看基本信息、刷新 token、管理 provider |
| `/admin/logout` | 登出 | 清除 session |

### 交互流程

1. 管理员打开 `/admin` → 未登录跳转 `/admin/login`
2. 输入密码 → 验证通过，设 session cookie → 跳转用户列表
3. 点击「创建用户」→ 输入用户名 → 展示完整 token（提示复制）
4. 点击用户 → 看详情：
   - 「刷新 token」按钮 → 旧 token 标记 `is_active=0`，生成新 token → 展示
   - 「管理邮箱源」→ 添加/删除该用户的 provider（QQ/Gmail/Outlook/Cloudflare）

## Auth 中间件变更

**改造前**（当前）：
- 读取 `AUTH_HEADER` / `AUTH_TOKEN` 环境变量
- 单一全局 token，不区分用户

**改造后**：
- 从请求中取 `X-Agent-Mail-Token` header
- SHA256 后查 `tokens` 表，找 `is_active=1` 的记录
- 找到 → 将 `user_id` 注入 request context
- 未找到 → 401
- 全局 admin token 保留作为兼容（env 配置的 AUTH_TOKEN 仍可用）

## MCP 工具用户隔离

- 所有 MCP 工具处理器从 context 中获取 `user_id`
- `list_mailboxes` 等工具自动限定在当前用户的邮箱
- 用户 A 的 token 无法访问用户 B 的邮箱

## Provider 工厂扩展

`DefaultProviderFactory` 需要支持新增的 provider 类型：

| ProviderType | 说明 | 优先级 |
|---|---|---|
| `cloudflare` | 已实现 | P0 |
| `gmail` | Gmail API | P1 |
| `outlook` | Microsoft Graph API | P2 |
| `qq` | QQ 邮箱 | P2 |

初期只实现 cloudflare，其余 provider 在后续迭代中添加。

## 文件清单

### 新增
- `store/sqlite/users.go` — 用户 CRUD
- `store/sqlite/tokens.go` — Token CRUD
- `store/sqlite/users_test.go`
- `store/sqlite/tokens_test.go`
- `service/user_svc.go` — 用户管理逻辑
- `service/user_svc_test.go`
- `web/templates/` — Go HTML 模板
- `web/handler.go` — Admin 页面路由
- `web/session.go` — Session 管理

### 修改
- `store/sqlite/db.go` — 迁移新增 users/tokens 表
- `store/sqlite/mailboxes.go` — CRUD 加 user_id 过滤
- `mcp/auth.go` — 中间件改为查 tokens 表
- `mcp/handler.go` — 从 context 取 user_id
- `service/mailbox_svc.go` — 适配用户隔离
- `main.go` — 挂载 admin handler，初始化 admin 密码

### 不变
- `model/types.go` — 类型无需变更
- `provider/` — 接口无需变更
- `provider/cloudflare/` — 无需变更
