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
- **结构化 JSON 输出**：LLM 返回 JSON 格式，解析失败自动回退到文本解析
- Prompt 组合和上下文管理
- 风险评估和建议生成
- 支持本地（Ollama）和云端 LLM 提供者
- **自动学习**：LLM 检测到违禁词时自动添加到敏感词库和 AC 自动机（增量更新，延迟 < 1ms）
- **LLM 可观测性**：Prometheus 指标（延迟、Token 用量、请求计数）

### ⚡ 性能优化

#### Redis 缓存层
- 检测结果缓存（TTL: 1小时）
- **Embedding 缓存**：相同文本的向量化结果缓存 24 小时，减少 API 调用
- 减少重复文本的计算开销
- 支持租户级别的缓存隔离

#### 异步检测队列
- 基于 Redis List 的消息队列
- 支持高并发场景的削峰填谷
- 阻塞式出队（BRPop）避免空轮询
- 任务结果缓存（TTL: 24小时）

#### 并发优化
- Layer1 和 Layer2 并发执行
- 使用 goroutine 和 sync.WaitGroup 实现并发控制
- 提升整体检测吞吐量

#### 性能测试工具
- 支持并发请求测试
- 支持队列模式测试
- 输出关键性能指标（QPS、平均延迟、P99延迟等）

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

### 🛡️ 工程化特性
- **统一响应格式**：基于 `ecode` 的统一错误码和响应封装
- **优雅关闭**：通过 context 传播取消信号，异步任务在关闭时排空
- **可配置 CORS**：支持域名白名单的跨域配置
- **租户级限流**：基于 Redis 滑动窗口的请求限流（429 + Retry-After）
- **数据库迁移**：基于 golang-migrate 的版本化迁移管理

## 🛠 技术栈

- **语言**: Go 1.25+
- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL + pgvector
- **缓存**: Redis
- **配置管理**: Viper
- **认证**: JWT
- **LLM 支持**: Ollama, 在线（OpenAI 兼容 API）
- **测试**: testify
- **数据库迁移**: golang-migrate
- **监控**: Prometheus + Grafana

## 📁 项目结构

```
├── cmd/
│   ├── server/              # main.go 入口
│   └── benchmark/           # 性能测试工具
├── internal/
│   ├── cache/               # Redis 缓存客户端
│   ├── config/              # Viper 配置定义
│   ├── metrics/             # Prometheus 指标定义
│   ├── middleware/          # Gin 中间件（JWT 鉴权、CORS、限流）
│   ├── models/              # GORM 数据表模型（实体）
│   ├── repository/          # 数据库操作层
│   ├── response/            # 统一 API 响应封装
│   ├── handler/             # Gin 路由控制器
│   ├── router/              # 路由配置
│   └── service/             # 核心业务逻辑层
│       ├── layer1_speed/    # 第一层：快速匹配
│       ├── layer2_semantic/ # 第二层：语义检索
│       ├── layer3_reason/   # 第三层：LLM 推理
│       ├── orchestrator/     # 编排层
│       ├── queue/           # 异步检测队列
│       ├── detection_history/# 检测历史服务
│       └── api_key/         # API 密钥管理
├── migrations/                 # 数据库迁移文件（golang-migrate）
├── pkg/
│   ├── benchmark/           # 性能测试工具
│   └── ecode/               # 统一定义的业务错误码
├── docs/
│   ├── LAYERED_ARCHITECTURE.md  # 三层架构文档
│   └── API_DOCUMENTATION.md       # API 文档
├── config.yaml                # 配置文件
├── docker-compose.yml         # Docker 编排配置
├── go.mod
└── README.md
```

## 🏗️ Architecture

<div align="center">
  <img src="./docs/ARCHITECTURE.png" alt="Jetwash Architecture Diagram" width="800" />
</div>

## 🚀 快速开始

### 方式一：Docker 部署（推荐）

#### 1. 启动所有服务

```bash
# 启动 PostgreSQL、Redis、Ollama 服务
docker-compose up -d

# 查看服务状态
docker-compose ps
```

#### 2. 安装 Ollama 模型

```bash
# 进入 Ollama 容器
docker exec -it jetwash-ollama bash

# 安装推理模型
ollama pull qwen2.5:0.5b

# 安装嵌入模型
ollama pull nomic-embed-text:v1.5

# 退出容器
exit
```

#### 3. 配置应用

编辑 `config.yaml` 文件：

```yaml
server:
  port: 13142
  mode: debug
  read_timeout: 300
  write_timeout: 300

database:
  host: localhost
  port: 15432
  user: postgres
  password: jetwash-postgres
  dbname: jetwash
  sslmode: disable
  max_open_conns: 100
  max_idle_conns: 10

redis:
  addr: "localhost:16379"
  password: "jetwash-redis"
  db: 0

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
  provider: "ollama"  # online, ollama

jwt:
  secret: "jetwash-jwt-secret"
  expire_hour: 24

cors:
  allowed_origins: ["*"]

rate_limit:
  enabled: true
  requests_per_minute: 60
```

#### 4. 数据库迁移

```bash
# 安装 golang-migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移
make migrate-up

# 回滚迁移
make migrate-down

# 创建新迁移
make migrate-create
```

#### 5. 运行应用

```bash
# 下载依赖
go mod tidy

# 运行服务
go run cmd/server/main.go

# 或者编译后运行
go build -o bin/jetwash cmd/server/main.go
./bin/jetwash
```

### 方式二：手动部署

#### 1. 环境准备

确保已安装：

- Go 1.25+
- PostgreSQL 14+（需安装 pgvector 扩展）
- Redis 7+

#### 2. 安装 pgvector 扩展

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

#### 3. 配置 Redis

```bash
# 启动 Redis（带密码）
redis-server --requirepass your-redis-password
```

#### 4. 配置应用

编辑 `config.yaml` 文件，配置数据库和 Redis 连接信息。

#### 5. 运行服务

```bash
# 下载依赖
go mod tidy

# 运行服务
go run cmd/server/main.go
```

## 📚 文档

- [三层架构详细文档](./docs/LAYERED_ARCHITECTURE.md)
- [API 文档](./docs/API_DOCUMENTATION.md)
- [代码优化文档](./docs/optimization/CODE_OPTIMIZATION.md)
- [AC 自动机更新策略](./docs/optimization/AC_AUTOMATON_UPDATE_STRATEGIES.md)
- [技术决策文档](./docs/technical-decisions/THREE_LAYER_ARCHITECTURE.md)

## 📋 TODO / 路线图

### 🎯 高优先级

- [ ] **语音内容检测** - 添加语音转文字功能，检测音频中的敏感内容
- [ ] **视频内容检测** - 提取视频帧和音频，检测敏感视觉内容和语音
- [ ] **图片内容检测** - 集成图像识别，检测敏感图片、Logo和不当内容
- [ ] **实时流检测** - 支持 WebSocket 实时文本/音频流检测

### ⚡ 性能优化

- [ ] **分布式缓存** - 支持 Redis Cluster 实现高可用和水平扩展
- [ ] **负载均衡** - 支持多后端实例和负载均衡
- [x] **租户级限流** - 实现租户级别的请求限流，防止滥用 ✅ 已实现
- [ ] **连接池优化** - 优化数据库和 Redis 连接池配置

### 🔧 功能增强

- [x] **AC 自动机增量更新** - 自动学习采用增量更新，新词 < 1ms 生效 ✅ 已实现
- [ ] **黑名单/白名单** - 支持租户级别的黑名单和白名单规则
- [ ] **实时告警** - 检测到敏感内容时发送 Webhook 通知
- [ ] **审计日志** - 全面的审计日志记录所有敏感操作
- [ ] **API 分析** - 租户级别的 API 使用统计和检测趋势仪表盘
- [ ] **批量检测 API** - 支持单次请求批量检测多条文本
- [ ] **语言检测** - 自动检测文本语言并应用相应规则

### 🔐 安全改进

- [ ] **数据加密** - 静态数据和传输数据加密
- [ ] **访问日志** - 详细记录 API 访问和操作日志
- [ ] **IP 白名单** - 允许租户限制特定 IP 范围的 API 访问
- [ ] **API Key 轮换** - 支持自动 API Key 轮换和管理

### 🧪 测试与质量

- [x] **单元测试** - 核心服务层单元测试（Layer1、Layer3、Orchestrator、Queue、Response） ✅ 已实现
- [ ] **集成测试** - 端到端集成测试
- [ ] **CI/CD 流水线** - 自动化测试和部署流水线
- [ ] **性能基准测试套件** - 定期性能回归测试

### 📈 监控与可观测性

- [x] **Prometheus 指标** - LLM 调用延迟、Token 用量、请求计数，`/metrics` 端点 ✅ 已实现
- [ ] **Grafana 仪表盘** - 实时监控可视化仪表盘
- [ ] **分布式追踪** - 支持 OpenTelemetry 分布式追踪
- [ ] **健康检查 API** - 为所有服务添加健康检查端点

## 🧪 性能测试

### 预热机制

系统包含预热机制，在启动时预加载关键组件以优化首次请求性能：

- **Layer1 AC 自动机**：为指定租户预加载敏感词（在内存中构建 AC 自动机）
- **Layer3 LLM 模型**：发送测试请求将 Ollama 模型加载到内存
- **优势**：将首次请求延迟从秒级降低到毫秒级

### Benchmark 工具

我们提供三个 benchmark 工具来测试不同场景：

#### 1. `benchmark` - 原始性能测试工具

支持随机文本生成、队列模式和不同检测模式。

```bash
# 编译所有 benchmark 工具
go build -o bin/benchmark cmd/benchmark/main.go

# 随机文本测试（混合敏感词）
./bin/benchmark -random -total 100 -concurrent 5 \
  -tenant a8102056-2b5d-4d92-b5e8-083bd96ec523

# 指定模式测试
./bin/benchmark -mode semantic -total 100 -concurrent 5

# 队列模式测试
./bin/benchmark -queue -total 1000

# 使用自定义配置文件
./bin/benchmark -config custom-config.yaml
```

#### 2. `benchmark_normal` - 纯非敏感词测试

测试快速放行路径（Layer1 快速放行），不调用 LLM。

```bash
# 编译并运行
go build -o bin/benchmark_normal cmd/benchmark_normal/main.go
./bin/benchmark_normal -total 1000 -concurrent 100

# 预期性能：QPS 1500+，延迟 < 1ms
```

#### 3. `benchmark_fix` - 混合文本测试（可配置比例）

模拟真实流量，可配置敏感词比例。

```bash
# 编译并运行
go build -o bin/benchmark_fix cmd/benchmark_fix/main.go

# 30% 敏感词 + 70% 非敏感词
./bin/benchmark_fix -total 100 -concurrent 5 -ratio 0.3 \
  -tenant a8102056-2b5d-4d92-b5e8-083bd96ec523

# 10% 敏感词（更接近真实场景）
./bin/benchmark_fix -total 200 -concurrent 5 -ratio 0.1 \
  -tenant a8102056-2b5d-4d92-b5e8-083bd96ec523
```

### Benchmark 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-total` | 总请求数 | 1000 |
| `-concurrent` | 并发请求数 | 100 |
| `-ratio` | 敏感词比例 (0-1) | 0.1 (10%) |
| `-tenant` | 指定租户ID | 自动生成 |
| `-random` | 生成随机文本 | false |
| `-mode` | 检测模式 (full/basic/semantic) | full |
| `-unique` | 添加唯一后缀 | true |
| `-queue` | 队列模式测试 | false |

### 性能对比

| 工具 | 场景 | 预期 QPS | LLM 调用 |
|------|------|----------|----------|
| benchmark_normal | 100% 正常文本 | 1500+ | 0% |
| benchmark_fix (10%) | 10% 敏感词 | 50-100 | 5-10% |
| benchmark_fix (30%) | 30% 敏感词 | 20-50 | 15-20% |
| benchmark (full) | 混合 + LLM | < 5 | 取决于内容 |

### 性能指标说明

- **QPS (Queries Per Second)**: 每秒处理的请求数
- **平均延迟**: 所有请求的平均响应时间
- **P50/P95/P99 延迟**: 50%/95%/99% 的请求响应时间
- **缓存命中率**: 从缓存服务的请求百分比
- **各层触发分布**: 统计哪些检测层被触发
- **成功率**: 请求成功的百分比
- **总耗时**: 所有请求完成的总时间

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
- [Redis](https://redis.io/) - 内存数据库和缓存
- [Ollama](https://ollama.com/) - 本地 LLM 服务
- [Viper](https://github.com/spf13/viper) - 配置管理

---

<div align="center">
  <img src="https://api.star-history.com/svg?repos=wenhaooooo/Jetwash&type=Date" alt="Star 趋势图表" />
</div>

<div align="center">
  <sub>Built with ❤️ by Wenhao</sub>
</div>
