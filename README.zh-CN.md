# Jetwash

[![Go 版本](https://img.shields.io/badge/Go-1.25+-00ADD8E?style=flat&logo=go)](https://golang.org/)
[![许可证](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

多租户敏感词与文本风控 SaaS 平台

## 项目简介

Jetwash 是一个基于 Go 语言开发的多租户敏感词与文本风控 SaaS 平台，采用三层漏斗式架构进行文本审查：

1. **Layer 1: Speed (快速匹配层)** - 基于 AC 自动机的精确匹配
2. **Layer 2: Semantic (语义检索层)** - 基于 pgvector 的向量检索
3. **Layer 3: Reason (推理层)** - 基于 LLM 的智能推理

## ✨ 核心功能

### 🏢 多租户数据隔离
- 每个租户拥有独立的 API Key
- 所有敏感词数据通过 TenantID 进行隔离
- 租户状态管理（active/inactive/suspended）
- 支持租户层级结构（最多5层）

### 🔍 三层漏斗式文本审查

#### Layer 1: 快速匹配层
- 基于 AC 自动机的精确匹配
- 支持正则表达式匹配
- 支持模糊匹配
- 支持多语言匹配
- 文本规范化预处理（全半角转换、繁简转换）

#### Layer 2: 语义检索层
- 使用 pgvector 进行向量相似度计算
- 支持欧几里得距离（`<->`）和余弦距离（`<=>`）
- 可配置相似度阈值和返回结果数量
- 支持多种 Filter 条件

#### Layer 3: 推理层
- 基于 LLM 的智能推理
- Prompt 组合和上下文管理
- 风险评估和建议生成
- 支持本地（Ollama）和云端 LLM 提供者

### 📝 敏感词管理
- 敏感词 CRUD 操作
- 敏感词启用/禁用
- 批量导入敏感词（CSV 格式）
- 分页查询敏感词列表
- 重复检测和自动跳过

### 📊 检测历史
- 记录每次文本检测结果
- 支持分页查询和过滤
- 支持时间范围筛选
- 支持删除和清空历史

### 🔐 认证与授权
- JWT Token 认证
- API Key 认证
- 多租户数据隔离
- 基于角色的访问控制

## 🛠 技术栈

- **语言**: Go 1.25+
- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL + pgvector
- **配置管理**: Viper
- **认证**: JWT
- **LLM 支持**: Ollama, 在线（OpenAI 兼容 API）

## 📁 项目结构

```
├── cmd/
│   └── server/              # main.go 入口
├── internal/
│   ├── config/              # Viper 配置定义
│   ├── middleware/          # Gin 中间件（JWT 鉴权、限流）
│   ├── models/              # GORM 数据表模型（实体）
│   ├── repository/          # 数据库操作层
│   ├── handler/             # Gin 路由控制器
│   ├── router/              # 路由配置
│   └── service/             # 核心业务逻辑层
│       ├── layer1_speed/    # 第一层：快速匹配
│       ├── layer2_semantic/ # 第二层：语义检索
│       ├── layer3_reason/   # 第三层：LLM 推理
│       ├── orchestrator/     # 编排层
│       ├── detection_history/# 检测历史服务
│       └── api_key/         # API 密钥管理
├── pkg/
│   ├── ecode/               # 统一定义的业务错误码
│   └── logger/              # 日志封装
├── docs/
│   ├── LAYERED_ARCHITECTURE.md  # 三层架构文档
│   └── API_DOCUMENTATION.md       # API 文档
├── migrations/                 # 数据库迁移文件
├── config.yaml                # 配置文件
├── go.mod
└── README.md
```

## 🚀 快速开始

### 1. 环境准备

确保已安装：

- Go 1.25+
- PostgreSQL 14+（需安装 pgvector 扩展）

### 2. 安装 pgvector 扩展

```sql
-- 连接到 PostgreSQL
psql -U postgres

-- 创建数据库
CREATE DATABASE jetwash;

-- 连接到数据库
\c jetwash

-- 安装 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;
```

### 3. 配置数据库

编辑 `config.yaml` 文件：

```yaml
server:
  port: 13142
  mode: debug  # debug, release
  read_timeout: 300
  write_timeout: 300

database:
  host: localhost
  port: 15432
  user: postgres
  password: 123456
  dbname: jetwash
  sslmode: disable
  max_open_conns: 100
  max_idle_conns: 10

llm:
  ollama:
    host: "http://localhost"
    port: 11434
    embedding_model: "nomic-embed-text:v1.5"
    reasoning_model: "qwen2.5:0.5b"
  online:
    api_key: "your-api-key"
    model: "glm-4.7"
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    embedding_model: "embedding-3"
  provider: "online"  # online, ollama

jwt:
  secret: "jetwash-jwt-secret"
  expire_hour: 24
```

### 4. 运行服务

```bash
# 下载依赖
go mod tidy

# 运行服务
go run cmd/server/main.go

# 或者编译后运行
go build -o bin/server cmd/server/main.go
./bin/server
```

## 📚 文档

- [三层架构详细文档](./docs/LAYERED_ARCHITECTURE.md)
- [API 文档](./docs/API_DOCUMENTATION.md)

## 🤝 贡献

欢迎贡献！请随时提交 Pull Request。

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [Gin](https://gin-gonic.com/) - HTTP Web 框架
- [GORM](https://gorm.io/) - ORM 库
- [pgvector](https://github.com/pgvector/pgvector) - PostgreSQL 向量相似度搜索
- [Ollama](https://ollama.com/) - 本地 LLM 服务
- [Viper](https://github.com/spf13/viper) - 配置管理

---

<div align="center">
  <h3>⭐ Star 趋势</h3>
  <img src="https://api.star-history.com/svg?repos=wenhao/jetwash&type=Date" alt="Star 趋势图表" />
</div>

<div align="center">
  <sub>Built with ❤️ by Wenhao</sub>
</div>
