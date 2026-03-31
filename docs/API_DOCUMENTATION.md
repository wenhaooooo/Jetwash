# Jetwash Platform API 文档

本文档提供了Jetwash Platform的所有API接口说明，包括接口路径、请求参数、响应格式等。

## 1. 认证方式

所有需要认证的接口都需要在请求头中添加 `X-API-Key` 字段，值为租户的API密钥。

```http
X-API-Key: your-api-key-here
```

## 2. 接口分类

### 2.1 健康检查接口

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
| `/health` | GET | 健康检查 | 否 |

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": "Jetwash Platform is running"
}
```

### 2.2 编排器接口（文本检测）

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
| `/api/v1/orchestrator/check` | POST | 检测文本 | 是 |
| `/api/v1/orchestrator/check/config` | POST | 带配置检测文本 | 是 |
| `/api/v1/orchestrator/check/context` | POST | 带上下文检测文本 | 是 |
| `/api/v1/orchestrator/check/full` | POST | 带配置和上下文检测文本 | 是 |
| `/api/v1/orchestrator/histories` | GET | 获取检测历史 | 是 |
| `/api/v1/orchestrator/histories/:id` | GET | 获取检测历史详情 | 是 |

**检测文本请求示例：**
```json
{
  "text": "测试文本内容"
}
```

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
      "matches": [...]  // 匹配的敏感词
    },
    "layer2_result": {
      "has_risk": true,
      "matches": [...]  // 语义匹配结果
    },
    "layer3_result": {
      "has_risk": true,
      "is_approved": false,
      "risk_level": 5,
      "risk_reason": "文本包含高风险内容"
    }
  }
}
```

### 2.3 普通用户接口

#### 2.3.1 认证相关

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
| `/api/v1/normal/login` | POST | 用户登录 | 否 |
| `/api/v1/normal/register` | POST | 用户注册 | 否 |

**登录请求示例：**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "token": "jwt-token",
    "tenant": {
      "id": "uuid",
      "name": "Tenant Name",
      "email": "user@example.com"
    }
  }
}
```

#### 2.3.2 敏感词管理

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
| `/api/v1/normal/words` | GET | 获取所有敏感词 | 是 |
| `/api/v1/normal/words/:id/status/:status` | PUT | 更新敏感词状态 | 是 |
| `/api/v1/normal/words/batch-import` | POST | 异步批量导入敏感词 | 是 |
| `/api/v1/normal/words/import` | POST | 导入敏感词 | 是 |

**导入敏感词请求示例：**
```json
{
  "word_text": "敏感词",
  "category": "profanity",
  "risk_level": 3
}
```

**批量导入敏感词请求：**
- 表单提交，文件字段名：`file`
- 支持CSV格式文件

### 2.3.3 导入任务管理

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
| `/api/v1/normal/import-tasks` | GET | 获取导入任务列表 | 是 |
| `/api/v1/normal/import-tasks/:id` | GET | 获取导入任务状态 | 是 |

### 2.3.4 检测历史管理

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
| `/api/v1/normal/detection-history` | GET | 获取检测历史列表 | 是 |
| `/api/v1/normal/detection-history/:id` | GET | 获取单个检测历史详情 | 是 |
| `/api/v1/normal/detection-history/:id` | DELETE | 删除检测历史 | 是 |
| `/api/v1/normal/detection-history` | DELETE | 清空检测历史 | 是 |

### 2.4 租户管理接口

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
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

**注意**：创建租户时，如果当前用户已登录，系统会自动将新租户的 `parent_id` 设置为当前登录租户的ID，实现租户层级结构。

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": "uuid",
    "parent_id": "parent-uuid",  // 父租户ID，如果为顶级租户则为null
    "api_key": "generated-api-key",
    "name": "Tenant Name",
    "email": "tenant@example.com",
    "status": 1
  }
}
```

### 2.5 API密钥管理接口

| 接口路径 | 方法 | 描述 | 鉴权 |
|---------|------|------|------|
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

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "id": "uuid",
    "api_key": "generated-api-key",
    "name": "API Key Name",
    "status": 1,
    "expires_at": "2024-12-31T23:59:59Z"
  }
}
```

## 3. 响应格式

所有API接口的响应格式统一为：

```json
{
  "code": 0,              // 状态码，0表示成功，非0表示错误
  "message": "Success",  // 响应消息
  "data": {}             // 响应数据，根据接口不同而不同
}
```

## 4. 错误码

| 错误码 | 描述 |
|-------|------|
| 400 | 请求参数错误 |
| 401 | 未授权，API密钥无效 |
| 403 | 禁止访问，权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突，如租户名称已存在 |
| 500 | 服务器内部错误 |

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

## 6. 状态码说明

- **租户状态**：1-活跃，2-不活跃，3-暂停
- **API密钥状态**：1-激活，2-停用
- **导入任务状态**：1-待处理，2-处理中，3-完成，4-失败

## 7. 导入文件格式

批量导入敏感词支持CSV格式，文件格式如下：

| 列名 | 描述 | 示例 |
|------|------|------|
| word_text | 敏感词文本 | 敏感词 |
| category | 分类 | profanity |
| risk_level | 风险等级 | 3 |
| description | 描述（可选） | 不良内容 |

**示例CSV文件：**
```csv
word_text,category,risk_level,description
敏感词1,profanity,3,不良内容
敏感词2,violence,4,暴力内容
```
