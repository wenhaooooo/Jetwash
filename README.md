# Jetwash Platform

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8E?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/yourusername/jetwash-platform?style=social)](https://github.com/yourusername/jetwash-platform/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/yourusername/jetwash-platform?style=social)](https://github.com/yourusername/jetwash-platform/network/members)
[![GitHub Issues](https://img.shields.io/github/issues/yourusername/jetwash-platform)](https://github.com/yourusername/jetwash-platform/issues)

---

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

<div align="center">
  <h3>⭐ Star History</h3>
  <img src="https://api.star-history.com/svg?repos=wenhao/jetwash-platform&type=Date" alt="Star History Chart" />
</div>

<div align="center">
  <sub>Built with ❤️ by Wenhao</sub>
</div>
