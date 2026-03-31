# Jetwash Platform

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8E?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/yourusername/jetwash-platform?style=social)](https://github.com/yourusername/jetwash-platform/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/yourusername/jetwash-platform?style=social)](https://github.com/yourusername/jetwash-platform/network/members)
[![GitHub Issues](https://img.shields.io/github/issues/yourusername/jetwash-platform)](https://github.com/yourusername/jetwash-platform/issues)

<div align="center">
  <h3>⭐ Star History</h3>
  <img src="https://api.star-history.com/svg?repos=yourusername/jetwash-platform&type=Date" alt="Star History Chart" />
</div>

---

## 📖 [English](#english) | [中文](#chinese)

---

<a name="english"></a>
# Jetwash Platform

A multi-tenant sensitive word filtering and text moderation SaaS platform built with Go, featuring a three-layer funnel architecture for comprehensive text review:

1. **Layer 1: Speed (Fast Matching Layer)** - Exact matching based on AC automaton
2. **Layer 2: Semantic (Semantic Retrieval Layer)** - Vector search based on pgvector
3. **Layer 3: Reason (Reasoning Layer)** - Intelligent reasoning based on LLM

## ✨ Features

### 🏢 Multi-Tenant Data Isolation
- Each tenant has independent API keys
- All sensitive word data is isolated through TenantID
- Tenant status management (active/inactive/suspended)
- Hierarchical tenant structure (up to 5 levels)

### 🔍 Three-Layer Funnel Text Review

#### Layer 1: Fast Matching Layer
- Exact matching based on AC automaton
- Support for regular expression matching
- Support for fuzzy matching
- Multi-language matching support
- Text normalization preprocessing (full/half-width conversion, traditional/simplified conversion)

#### Layer 2: Semantic Retrieval Layer
- Vector similarity calculation using pgvector
- Support for Euclidean distance (`<->`) and cosine distance (`<=>`)
- Configurable similarity threshold and result count
- Support for multiple filter conditions

#### Layer 3: Reasoning Layer
- Intelligent reasoning based on LLM
- Prompt composition and context management
- Risk assessment and suggestion generation
- Support for both local (Ollama) and cloud LLM providers

### 📝 Sensitive Word Management
- CRUD operations for sensitive words
- Enable/disable sensitive words
- Batch import sensitive words (CSV format)
- Paginated query of sensitive word list
- Duplicate detection and automatic skipping

### 📊 Detection History
- Record every text detection result
- Support paginated queries and filtering
- Support time range filtering
- Support deletion and clearing history

### 🔐 Authentication & Authorization
- JWT token authentication
- API key authentication
- Multi-tenant data isolation
- Role-based access control

## 🛠 Tech Stack

- **Language**: Go 1.25+
- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL + pgvector
- **Configuration**: Viper
- **Authentication**: JWT
- **LLM Support**: Ollama, Online (OpenAI-compatible APIs)

## 📁 Project Structure

```
├── cmd/
│   └── server/              # main.go entry point
├── internal/
│   ├── config/              # Viper configuration definitions
│   ├── middleware/          # Gin middleware (JWT auth, rate limiting)
│   ├── models/              # GORM data models (entities)
│   ├── repository/          # Database operation layer
│   ├── handler/             # Gin route controllers
│   ├── router/              # Route configuration
│   └── service/             # Core business logic layer
│       ├── layer1_speed/    # Layer 1: Fast matching
│       ├── layer2_semantic/ # Layer 2: Semantic retrieval
│       ├── layer3_reason/   # Layer 3: LLM reasoning
│       ├── orchestrator/     # Orchestrator layer
│       ├── detection_history/# Detection history service
│       └── api_key/         # API key management
├── pkg/
│   ├── ecode/               # Unified business error codes
│   └── logger/              # Logging wrapper
├── docs/
│   ├── LAYERED_ARCHITECTURE.md  # Three-layer architecture documentation
│   └── API_DOCUMENTATION.md       # API documentation
├── migrations/                 # Database migration files
├── config.yaml                # Configuration file
├── go.mod
└── README.md
```

## 🚀 Quick Start

### 1. Prerequisites

Ensure you have installed:

- Go 1.25+
- PostgreSQL 14+ (with pgvector extension)

### 2. Install pgvector Extension

```sql
-- Connect to PostgreSQL
psql -U postgres

-- Create database
CREATE DATABASE jetwash;

-- Connect to database
\c jetwash

-- Install pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
```

### 3. Configure Database

Edit `config.yaml` file:

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

### 4. Run Service

```bash
# Download dependencies
go mod tidy

# Run service
go run cmd/server/main.go

# Or build and run
go build -o bin/server cmd/server/main.go
./bin/server
```

## 📚 Documentation

- [Three-Layer Architecture Documentation](./docs/LAYERED_ARCHITECTURE.md)
- [API Documentation](./docs/API_DOCUMENTATION.md)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Gin](https://gin-gonic.com/) - HTTP web framework
- [GORM](https://gorm.io/) - ORM library
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search for PostgreSQL
- [Ollama](https://ollama.com/) - Local LLM service
- [Viper](https://github.com/spf13/viper) - Configuration management

---

<a name="chinese"></a>
# Jetwash 平台

[![Go 版本](https://img.shields.io/badge/Go-1.25+-00ADD8E?style=flat&logo=go)](https://golang.org/)
[![许可证](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/yourusername/jetwash-platform?style=social)](https://github.com/yourusername/jetwash-platform/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/yourusername/jetwash-platform?style=social)](https://github.com/yourusername/jetwash-platform/network/members)
[![GitHub Issues](https://img.shields.io/github/issues/yourusername/jetwash-platform)](https://github.com/yourusername/jetwash-platform/issues)

<div align="center">
  <h3>⭐ Star 趋势</h3>
  <img src="https://api.star-history.com/svg?repos=yourusername/jetwash-platform&type=Date" alt="Star 趋势图表" />
</div>

---

## 📖 [English](#english) | [中文](#chinese)

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
  <sub>Built with ❤️ by Jetwash Team</sub>
</div>
