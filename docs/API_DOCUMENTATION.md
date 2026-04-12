# Jetwash Platform API 文档

本文档提供了 Jetwash Platform 的所有 API 接口说明，包括接口路径、请求参数、响应格式等。

---

## 1. 认证方式

### 1.1 API Key 认证（推荐）

所有需要认证的接口都需要在请求头中添加 `X-API-Key` 字段，值为租户的 API 密钥。

```http
X-API-Key: your-api-key-here
```

### 1.2 JWT Token 认证

部分接口支持 JWT Token 认证，在登录后获取 Token，然后在请求头中添加：

```http
Authorization: Bearer your-jwt-token
```

---

## 2. 接口分类

### 2.1 健康检查接口

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/health` | GET | 健康检查 | 否 |

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": "Jetwash Platform is running"
}
```

---

### 2.2 编排器接口（文本检测）

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/orchestrator/check` | POST | 检测文本 | 是 |
| `/api/v1/orchestrator/check/config` | POST | 带配置检测文本 | 是 |
| `/api/v1/orchestrator/check/context` | POST | 带上下文检测文本 | 是 |
| `/api/v1/orchestrator/check/full` | POST | 带配置和上下文检测文本 | 是 |
| `/api/v1/orchestrator/histories` | GET | 获取检测历史列表 | 是 |
| `/api/v1/orchestrator/histories/:id` | GET | 获取检测历史详情 | 是 |

#### 2.2.1 检测文本

**请求示例：**
```json
{
  "text": "测试文本内容"
}
```

**请求参数说明：**
| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| text | string | 是 | 需要检测的文本内容 |

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "passed": false,
    "risk_level": 5,
    "message": "文本包含高风险敏感内容",
    "total_matches": 10,
    "layer1_result": {
      "has_risk": true,
      "matches": ["敏感词1", "敏感词2"]
    },
    "layer2_result": {
      "has_risk": true,
      "matches": [
        {"text": "匹配文本", "confidence": 0.95}
      ]
    },
    "layer3_result": {
      "has_risk": true,
      "is_approved": false,
      "risk_level": 5,
      "risk_reason": "文本包含高风险内容",
      "confidence": 0.98,
      "reasoning": "LLM 推理过程",
      "detected_words": ["新检测到的违禁词"]
    },
    "duration_ms": 150
  }
}
```

**响应字段说明：**
| 字段名 | 类型 | 描述 |
|-------|------|------|
| passed | bool | 是否通过检测 |
| risk_level | int | 风险等级 (1-5) |
| message | string | 检测结果消息 |
| total_matches | int | 匹配的敏感词总数 |
| layer1_result | object | Layer1 AC自动机检测结果 |
| layer2_result | object | Layer2 语义匹配检测结果 |
| layer3_result | object | Layer3 LLM推理检测结果 |
| duration_ms | int | 检测耗时(毫秒) |

#### 2.2.2 带配置检测文本

**请求示例：**
```json
{
  "text": "测试文本内容",
  "config": {
    "enable_layer1": true,
    "enable_layer2": true,
    "enable_layer3": true,
    "layer1_threshold": 0.8,
    "layer2_threshold": 0.7,
    "layer3_threshold": 0.6
  }
}
```

**请求参数说明：**
| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| text | string | 是 | 需要检测的文本内容 |
| config | object | 否 | 检测配置 |
| config.enable_layer1 | bool | 否 | 是否启用Layer1检测 |
| config.enable_layer2 | bool | 否 | 是否启用Layer2检测 |
| config.enable_layer3 | bool | 否 | 是否启用Layer3检测 |
| config.layer1_threshold | float | 否 | Layer1阈值 |
| config.layer2_threshold | float | 否 | Layer2阈值 |
| config.layer3_threshold | float | 否 | Layer3阈值 |

#### 2.2.3 获取检测历史列表

**请求参数（Query）：**
| 参数名 | 类型 | 默认值 | 描述 |
|-------|------|-------|------|
| page | int | 1 | 页码 |
| page_size | int | 10 | 每页数量 |

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "list": [
      {
        "id": 1,
        "text": "测试文本",
        "mode": "full",
        "is_offensive": true,
        "duration": 150,
        "created_at": "2024-01-01T12:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

---

### 2.3 普通用户接口

#### 2.3.1 认证相关

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/normal/login` | POST | 用户登录 | 否 |
| `/api/v1/normal/register` | POST | 用户注册 | 否 |

**登录请求示例：**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**登录响应示例：**
```json
{
  "code": 0,
  "message": "Login success",
  "data": {
    "id": "uuid-string",
    "api_key": "your-api-key",
    "name": "Tenant Name",
    "email": "user@example.com",
    "status": 1,
    "token": "jwt-token"
  }
}
```

**注册请求示例：**
```json
{
  "name": "Tenant Name",
  "email": "tenant@example.com",
  "password": "password123"
}
```

#### 2.3.2 敏感词管理

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/normal/words` | GET | 获取所有敏感词 | 是 |
| `/api/v1/normal/words/:id/status/:status` | PUT | 更新敏感词状态 | 是 |
| `/api/v1/normal/words/batch-import` | POST | 异步批量导入敏感词 | 是 |
| `/api/v1/normal/words/import` | POST | 导入单个敏感词 | 是 |

**获取敏感词列表请求参数（Query）：**
| 参数名 | 类型 | 默认值 | 描述 |
|-------|------|-------|------|
| page | int | 1 | 页码 |
| page_size | int | 10 | 每页数量 |
| category | string | - | 分类筛选 |
| status | int | - | 状态筛选 |

**获取敏感词列表响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "list": [
      {
        "id": 1,
        "word_text": "敏感词",
        "category": "profanity",
        "risk_level": 3,
        "status": 1,
        "created_at": "2024-01-01T12:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

**导入单个敏感词请求示例：**
```json
{
  "word_text": "敏感词",
  "category": "profanity",
  "risk_level": 3,
  "description": "不良内容"
}
```

**批量导入敏感词请求：**
- 表单提交，文件字段名：`file`
- 支持 CSV 格式文件

**CSV 文件格式：**
```csv
word_text,category,risk_level,description
敏感词1,profanity,3,不良内容
敏感词2,violence,4,暴力内容
```

#### 2.3.3 导入任务管理

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/normal/import-tasks` | GET | 获取导入任务列表 | 是 |
| `/api/v1/normal/import-tasks/:id` | GET | 获取导入任务状态 | 是 |

**获取导入任务状态响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": "uuid-string",
    "file_name": "words.csv",
    "status": "completed",
    "total": 100,
    "imported": 95,
    "failed": 5,
    "error_msg": "5条记录导入失败",
    "started_at": "2024-01-01T12:00:00Z",
    "completed_at": "2024-01-01T12:01:00Z"
  }
}
```

#### 2.3.4 检测历史管理

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/normal/detection-history` | GET | 获取检测历史列表 | 是 |
| `/api/v1/normal/detection-history/:id` | GET | 获取单个检测历史详情 | 是 |
| `/api/v1/normal/detection-history/:id` | DELETE | 删除检测历史 | 是 |
| `/api/v1/normal/detection-history` | DELETE | 清空检测历史 | 是 |

**获取检测历史列表请求参数（Query）：**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|-------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 10 | 每页数量，最大100 |
| mode | string | 否 | - | 检测模式过滤 |
| start_time | string | 否 | - | 开始时间（ISO 8601格式） |
| end_time | string | 否 | - | 结束时间（ISO 8601格式） |

**获取检测历史列表响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "histories": [
      {
        "id": 1,
        "tenant_id": "917d94dc-3b64-4bb0-9e0b-4e4b748f9349",
        "text": "待检测文本...",
        "mode": "basic",
        "is_offensive": true,
        "result_json": "{\"is_offensive\": true, \"matches\": [...]}",
        "duration": 150,
        "created_at": "2026-03-24T10:00:00Z",
        "updated_at": "2026-03-24T10:00:00Z",
        "matches": [
          {
            "id": 1,
            "history_id": 1,
            "type": "政治",
            "text": "敏感词",
            "confidence": 0.95,
            "created_at": "2026-03-24T10:00:00Z"
          }
        ]
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10,
    "total_pages": 10
  }
}
```

**获取单个检测历史详情响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": 1,
    "tenant_id": "917d94dc-3b64-4bb0-9e0b-4e4b748f9349",
    "text": "待检测文本...",
    "mode": "basic",
    "is_offensive": true,
    "result_json": "{\"is_offensive\": true, \"matches\": [...]}",
    "duration": 150,
    "created_at": "2026-03-24T10:00:00Z",
    "updated_at": "2026-03-24T10:00:00Z",
    "matches": [
      {
        "id": 1,
        "history_id": 1,
        "type": "政治",
        "text": "敏感词",
        "confidence": 0.95,
        "created_at": "2026-03-24T10:00:00Z"
      }
    ]
  }
}
```

**删除检测历史响应示例：**
```json
{
  "code": 0,
  "message": "Deleted successfully"
}
```

**清空检测历史响应示例：**
```json
{
  "code": 0,
  "message": "Cleared successfully"
}
```

**注意事项：**
- 时间参数使用 ISO 8601 格式，例如 `2026-03-24T10:00:00Z`
- `page_size` 最大值为 100
- 删除操作使用软删除，数据不会立即从数据库中删除
- 删除检测历史时会级联删除关联的匹配记录

---

### 2.4 租户管理接口

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/tenants` | POST | 创建租户 | 否 |
| `/api/v1/tenants` | GET | 列出租户 | 否 |
| `/api/v1/tenants/:id` | GET | 获取租户详情 | 否 |
| `/api/v1/tenants/:id` | PUT | 更新租户 | 否 |
| `/api/v1/tenants/:id` | DELETE | 删除租户 | 否 |

**创建租户请求示例：**
```json
{
  "name": "Tenant Name",
  "email": "tenant@example.com",
  "password": "password123"
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": "uuid-string",
    "parent_id": null,
    "api_key": "generated-api-key",
    "name": "Tenant Name",
    "email": "tenant@example.com",
    "status": 1,
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

**更新租户请求示例：**
```json
{
  "name": "New Tenant Name",
  "email": "new-email@example.com",
  "password": "new-password"
}
```

**注意**：创建租户时，如果当前用户已登录，系统会自动将新租户的 `parent_id` 设置为当前登录租户的ID，实现租户层级结构。租户层级最多支持5层。

---

### 2.5 API密钥管理接口

| 接口路径 | HTTP 方法 | 描述 | 是否需要鉴权 |
|---------|----------|------|------------|
| `/api/v1/api-keys` | POST | 创建API密钥 | 是 |
| `/api/v1/api-keys` | GET | 列出API密钥 | 是 |
| `/api/v1/api-keys/:id` | GET | 获取API密钥详情 | 是 |
| `/api/v1/api-keys/:id` | PUT | 更新API密钥 | 是 |
| `/api/v1/api-keys/:id` | DELETE | 删除API密钥 | 是 |
| `/api/v1/api-keys/:id/activate` | POST | 激活API密钥 | 是 |
| `/api/v1/api-keys/:id/deactivate` | POST | 停用API密钥 | 是 |

**创建API密钥请求示例：**
```json
{
  "name": "API Key Name",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**创建API密钥响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": "uuid-string",
    "api_key": "generated-api-key",
    "name": "API Key Name",
    "status": 1,
    "expires_at": "2024-12-31T23:59:59Z",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

---

## 3. 响应格式

所有 API 接口的响应格式统一为：

```json
{
  "code": 0,              // 状态码，0表示成功，非0表示错误
  "message": "Success",   // 响应消息
  "data": {}              // 响应数据，根据接口不同而不同
}
```

---

## 4. 错误码

| 错误码 | HTTP状态码 | 描述 |
|-------|-----------|------|
| 0 | 200 | 成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未授权，API密钥无效 |
| 403 | 403 | 禁止访问，权限不足 |
| 404 | 404 | 资源不存在 |
| 409 | 409 | 资源冲突（如租户名称已存在） |
| 500 | 500 | 服务器内部错误 |

---

## 5. 分页参数

对于支持分页的接口，使用以下参数：

| 参数名 | 类型 | 默认值 | 描述 |
|-------|------|-------|------|
| page | int | 1 | 页码 |
| page_size | int | 10 | 每页数量 |

**示例：**
```
GET /api/v1/tenants?page=1&page_size=20
```

---

## 6. 状态码说明

### 租户状态
| 值 | 描述 |
|---|------|
| 1 | 活跃 (active) |
| 2 | 不活跃 (inactive) |
| 3 | 暂停 (suspended) |

### API密钥状态
| 值 | 描述 |
|---|------|
| 1 | 激活 (active) |
| 2 | 停用 (inactive) |

### 敏感词状态
| 值 | 描述 |
|---|------|
| 1 | 启用 (active) |
| 2 | 停用 (inactive) |
| 3 | 归档 (archived) |

### 导入任务状态
| 值 | 描述 |
|---|------|
| pending | 待处理 |
| processing | 处理中 |
| completed | 完成 |
| failed | 失败 |

---

## 7. 敏感词分类

| 分类 | 描述 |
|------|------|
| profanity | 脏话/辱骂 |
| violence | 暴力 |
| pornography | 色情 |
| politics | 政治敏感 |
| advertising | 广告 |
| spam | 垃圾信息 |
| llm_detected | LLM检测到的违禁词 |
| other | 其他 |

---

## 8. 风险等级

| 等级 | 描述 |
|------|------|
| 1 | 低风险 |
| 2 | 中低风险 |
| 3 | 中等风险 |
| 4 | 较高风险 |
| 5 | 高风险 |

---

## 9. 缓存策略

### 9.1 分层缓存策略

系统根据检测结果的风险等级设置不同的缓存过期时间：

| 结果类型 | 风险等级 | TTL | 说明 |
|----------|----------|-----|------|
| 高风险 | >= 4 | 7 天 | 用于审计和合规 |
| 中等风险 | 1-3 | 1 天 | 普通风险结果 |
| 通过 | 0 | 1 小时 | 无风险结果 |

### 9.2 访问刷新机制

当缓存命中时，系统自动延长过期时间（异步执行，不阻塞主流程），确保热门数据持续保留。

### 9.3 Redis LRU 淘汰策略

系统配置了 Redis 内存保护机制：

| 参数 | 值 | 说明 |
|------|-----|------|
| maxmemory | 512mb | 内存上限 |
| maxmemory-policy | allkeys-lru | 淘汰最久未使用的键 |

---

## 10. 性能测试工具

### 10.1 使用方式

```bash
# 基本用法
go run cmd/benchmark/main.go -total 1000 -concurrent 100

# 指定检测模式
go run cmd/benchmark/main.go -total 1000 -concurrent 100 -mode basic

# 指定测试文本
go run cmd/benchmark/main.go -total 1000 -concurrent 100 -text "测试文本"

# 使用队列模式
go run cmd/benchmark/main.go -total 1000 -concurrent 100 -queue
```

### 10.2 参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|-------|------|
| -total | int | 1000 | 总请求数 |
| -concurrent | int | 100 | 并发请求数 |
| -mode | string | full | 检测模式（basic/semantic/full） |
| -text | string | 测试文本 | 待检测文本 |
| -queue | bool | false | 是否使用队列模式 |
| -config | string | config.yaml | 配置文件路径 |

### 10.3 检测模式说明

| 模式 | 说明 | 性能 |
|------|------|------|
| basic | 仅 Layer1（AC自动机） | 最快（QPS ~3000） |
| semantic | Layer1 + Layer2（语义检索） | 中等 |
| full | 完整三层检测 | 最慢（QPS ~2） |

---

## 11. 批量导入文件格式

批量导入敏感词支持 CSV 格式，文件格式如下：

| 列名 | 类型 | 必填 | 描述 |
|------|------|------|------|
| word_text | string | 是 | 敏感词文本 |
| category | string | 是 | 分类 |
| risk_level | int | 是 | 风险等级 (1-5) |
| description | string | 否 | 描述 |

**示例 CSV 文件：**
```csv
word_text,category,risk_level,description
敏感词1,profanity,3,不良内容
敏感词2,violence,4,暴力内容
敏感词3,pornography,5,色情内容
```
