# 三层架构文档

本文档描述了 Jetwash 平台的三层文本审查架构。

## 架构概述

Jetwash 平台采用三层漏斗式架构进行文本审查，每一层都有不同的职责和特点：

1. **Layer 1: Speed (快速匹配层)** - 基于 AC 自动机的精确匹配
2. **Layer 2: Semantic (语义检索层)** - 基于 pgvector 的向量检索
3. **Layer 3: Reason (推理层)** - 基于 LLM 的智能推理（支持自动学习）

## 目录结构

```
internal/service/
├── layer1_speed/           # 第一层：快速匹配层
│   ├── ac_automaton.go     # AC 自动机实现
│   ├── text_normalizer.go  # 文本规范化（全半角/繁简转换）
│   ├── regex_matcher.go    # 正则表达式匹配器
│   ├── fuzzy_matcher.go    # 模糊匹配器
│   ├── multilang_matcher.go # 多语言匹配器
│   ├── service.go         # 第一层服务接口
│   └── service_extended.go # 扩展服务接口
├── layer2_semantic/        # 第二层：语义检索层
│   ├── service.go         # 第二层服务接口
│   └── repository.go      # pgvector 仓库实现
├── layer3_reason/         # 第三层：推理层（支持自动学习）
│   └── service.go         # 第三层服务接口（LLM Prompt 组合 + 自动学习）
├── orchestrator/           # 编排层
│   └── orchestrator.go    # 三层漏斗编排 + 自动学习集成
├── queue/                  # 异步队列服务
│   └── queue_service.go   # Redis 队列实现
└── detection_history/      # 检测历史服务
    └── service.go         # 检测历史管理
```

```
internal/cache/
└── redis.go               # Redis 缓存客户端（缓存 + 队列）
```

## Layer 1: Speed (快速匹配层)

### 功能

- 基于 AC 自动机的精确匹配
- 支持挂载 tenant_id payload
- 文本规范化预处理（全半角转换、繁简转换）
- **正则表达式匹配**
- **模糊匹配**
- **多语言匹配**

### 核心组件

#### AC 自动机

```go
type ACAutomaton struct {
    root *ACNode
    mu   sync.RWMutex
}

type Payload struct {
    TenantID  uuid.UUID
    WordText  string
    Category  string
    RiskLevel int
}
```

#### 文本规范化

```go
type TextNormalizer struct {
    traditionalToSimplified map[rune]rune
    halfwidthToFullwidth    map[rune]rune
    fullwidthToHalfwidth    map[rune]rune
}
```

#### 正则表达式匹配器

```go
type RegexMatcher struct {
    patterns map[string]*RegexPattern
    mu       sync.RWMutex
}

type RegexMatchResult struct {
    Pattern   string   `json:"pattern"`
    Payload   *Payload `json:"payload"`
    Position  int      `json:"position"`
    Matched   string   `json:"matched"`
    MatchType string   `json:"match_type"`
}
```

#### 模糊匹配器

```go
type FuzzyMatcher struct {
    words    map[string]*FuzzyWord
    mu       sync.RWMutex
}

type FuzzyMatchResult struct {
    Payload    *Payload `json:"payload"`
    Position   int      `json:"position"`
    Matched    string   `json:"matched"`
    Target     string   `json:"target"`
    Similarity float64  `json:"similarity"`
    MatchType  string   `json:"match_type"`
}
```

#### 多语言匹配器

```go
type MultiLangMatcher struct {
    wordsByLang map[Language]map[string]*MultiLangWord
    mu          sync.RWMutex
}

type Language string

const (
    LanguageZH   Language = "zh"   // 中文
    LanguageEN   Language = "en"   // 英文
    LanguageJA   Language = "ja"   // 日文
    LanguageKO   Language = "ko"   // 韩文
    LanguageAuto Language = "auto" // 自动检测
)
```

### 使用示例

#### 基础 AC 自动机匹配

```go
// 创建服务
layer1Service := layer1_speed.NewLayer1Service()

// 添加敏感词
payload := &layer1_speed.Payload{
    TenantID:  tenantID,
    WordText:  "敏感词",
    Category:  "政治",
    RiskLevel: 5,
}
layer1Service.AddWord("敏感词", payload)

// 检查文本
result, err := layer1Service.CheckText(tenantID, "这是一段包含敏感词的文本")
```

#### 扩展服务（包含所有匹配方式）

```go
// 创建扩展服务
extendedService := layer1_speed.NewExtendedLayer1Service()

// 添加 AC 自动机词
payload := &layer1_speed.Payload{
    TenantID:  tenantID,
    WordText:  "敏感词",
    Category:  "政治",
    RiskLevel: 5,
}
extendedService.AddWord("敏感词", payload)

// 添加正则表达式模式
extendedService.AddRegexPattern(`\d{11}`, []*layer1_speed.Payload{payload})

// 添加模糊词
extendedService.AddFuzzyWord("敏感", []*layer1_speed.Payload{payload})

// 添加多语言词
extendedService.AddMultiLangWord("sensitive", layer1_speed.LanguageEN, []*layer1_speed.Payload{payload})

// 综合匹配（所有匹配方式）
result := extendedService.MatchAll("这是一段文本", tenantID, 0.7, layer1_speed.LanguageAuto)
```

#### 正则表达式匹配

```go
// 创建扩展服务
extendedService := layer1_speed.NewExtendedLayer1Service()

// 添加正则表达式模式（匹配手机号）
payload := &layer1_speed.Payload{
    TenantID:  tenantID,
    WordText:  "手机号",
    Category:  "隐私",
    RiskLevel: 3,
}
extendedService.AddRegexPattern(`1[3-9]\d{9}`, []*layer1_speed.Payload{payload})

// 正则匹配
results := extendedService.MatchRegex("我的手机号是13812345678")
```

#### 模糊匹配

```go
// 创建扩展服务
extendedService := layer1_speed.NewExtendedLayer1Service()

// 添加模糊词
payload := &layer1_speed.Payload{
    TenantID:  tenantID,
    WordText:  "敏感词",
    Category:  "政治",
    RiskLevel: 5,
}
extendedService.AddFuzzyWord("敏感", []*layer1_speed.Payload{payload})

// 模糊匹配（相似度阈值 0.7）
results := extendedService.MatchFuzzy("这段文字包含敏盛内容", 0.7)
```

#### 多语言匹配

```go
// 创建扩展服务
extendedService := layer1_speed.NewExtendedLayer1Service()

// 添加中文词
payload := &layer1_speed.Payload{
    TenantID:  tenantID,
    WordText:  "敏感词",
    Category:  "政治",
    RiskLevel: 5,
}
extendedService.AddMultiLangWord("敏感词", layer1_speed.LanguageZH, []*layer1_speed.Payload{payload})

// 添加英文词
extendedService.AddMultiLangWord("sensitive", layer1_speed.LanguageEN, []*layer1_speed.Payload{payload})

// 多语言匹配（自动检测语言）
results := extendedService.MatchMultiLang("This is sensitive content", layer1_speed.LanguageAuto)
```

## Layer 2: Semantic (语义检索层)

### 功能

- 基于 pgvector 的向量检索
- 支持多种 Filter 条件
- 混合检索（向量 + 关键词）

### 核心组件

#### 语义检索服务

```go
type Layer2Service interface {
    SemanticSearch(tenantID uuid.UUID, text string, threshold float64, limit int, filters map[string]interface{}) (*Layer2Result, error)
    SemanticSearchWithVector(tenantID uuid.UUID, vector pgvector.Vector, threshold float64, limit int, filters map[string]interface{}) (*Layer2Result, error)
    GetTextEmbedding(text string) (pgvector.Vector, error)
}
```

#### 过滤条件构建器

```go
filterBuilder := layer2_semantic.NewFilterBuilder().
    WithCategory("政治").
    WithRiskLevelRange(3, 5).
    WithStatus("active")

filters := filterBuilder.Build()
```

### 使用示例

```go
// 创建服务
semanticRepo := layer2_semantic.NewSemanticRepository(db)
layer2Service := layer2_semantic.NewLayer2Service(semanticRepo)

// 语义检索
result, err := layer2Service.SemanticSearch(
    tenantID,
    "这是一段文本",
    0.3,  // threshold
    10,   // limit
    nil,  // filters
)
```

## Layer 3: Reason (推理层)

### 功能

- 基于 LLM 的智能推理
- Prompt 组合和上下文管理
- 风险评估和建议生成
- **自动学习**：当 LLM 检测到新的违禁词时，自动添加到敏感词库和 Redis 缓存

### 核心组件

#### 推理服务

```go
type Layer3Service interface {
    ReasonText(tenantID uuid.UUID, text string, context *ReasonContext) (*Layer3Result, error)
    ReasonWithMatches(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) (*Layer3Result, error)
    GeneratePrompt(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) string
}
```

#### 推理结果（支持自动学习）

```go
type Layer3Result struct {
    HasRisk       bool     `json:"has_risk"`
    RiskLevel     int      `json:"risk_level"`
    RiskReason    string   `json:"risk_reason"`
    Suggestions   []string `json:"suggestions"`
    IsApproved    bool     `json:"is_approved"`
    Confidence    float64  `json:"confidence"`
    Reasoning     string   `json:"reasoning"`
    DetectedWords []string `json:"detected_words"` // LLM 识别出的违禁词（用于自动学习）
}
```

#### 推理上下文

```go
context := layer3_reason.NewReasonContext().
    WithScenario("社交媒体评论").
    WithUserContext("新用户").
    WithCustomRule("禁止政治敏感内容").
    WithTemperature(0.7)
```

### 自动学习机制

当 Layer 3 检测到新的违禁词且判断文本有风险时，系统会自动执行以下操作：

1. **检测条件**：
   - Layer 3 LLM 检测到风险（`HasRisk = true`）
   - LLM 识别出具体的违禁词（`DetectedWords` 不为空）
   - LLM 不批准该文本（`IsApproved = false`）

2. **自动学习流程**：
   - 将检测到的违禁词添加到数据库（category = "llm_detected", risk_level = 3）
   - 将检测到的违禁词添加到 Redis 缓存（Set 结构，TTL = 7 天）
   - 自动更新 AC 自动机以包含新词

### 使用示例

```go
// 创建 Mock LLM 提供者（实际应用中应使用真实的 LLM）
mockLLMProvider := layer3_reason.NewMockLLMProvider("LLM 响应")
layer3Service := layer3_reason.NewLayer3Service(mockLLMProvider)

// 推理分析（返回包含 DetectedWords 的结果）
result, err := layer3Service.ReasonText(tenantID, "这是一段文本", context)

// 检查是否检测到新的违禁词（用于自动学习）
if len(result.DetectedWords) > 0 && !result.IsApproved {
    // 自动添加到数据库和 Redis
}
```

## Orchestrator (编排层)

### 功能

- 三层漏斗式编排
- 灵活的配置选项
- 结果聚合和总结
- **自动学习集成**：自动将 LLM 检测到的违禁词添加到敏感词库

### 核心组件

#### 编排器

```go
type Orchestrator interface {
    CheckText(tenantID uuid.UUID, text string) (*OrchestratorResult, error)
    CheckTextWithConfig(tenantID uuid.UUID, text string, config *OrchestratorConfig) (*OrchestratorResult, error)
    CheckTextWithContext(tenantID uuid.UUID, text string, context *ReasonContext) (*OrchestratorResult, error)
}
```

#### 编排配置

```go
config := &orchestrator.OrchestratorConfig{
    EnableLayer1:       true,
    EnableLayer2:       true,
    EnableLayer3:       true,
    StopAtLayer1:       false,
    StopAtLayer2:       false,
    Layer2Threshold:    0.3,
    Layer2Limit:        10,
    Layer3EnableReason: true,
    EnableAutoLearning: true, // 是否启用自动学习
}
```

### 自动学习集成

编排器在处理检测结果时，会自动检查 Layer 3 是否检测到新的违禁词：

```go
// 如果 LLM 检测到新的违禁词且判断文本有风险，记录这些违禁词
if len(layer3Result.DetectedWords) > 0 && !layer3Result.IsApproved {
    newlyDetectedWords = layer3Result.DetectedWords
}

// 自动将检测到的违禁词添加到数据库和 Redis
if len(newlyDetectedWords) > 0 {
    o.addDetectedWordsToDatabase(tenantID, newlyDetectedWords)
    o.addDetectedWordsToRedis(tenantID, newlyDetectedWords)
}
```

### 使用示例

```go
// 创建编排器
orchestrator := orchestrator.NewOrchestrator(layer1Service, layer2Service, layer3Service, redisClient)

// 检查文本（自动学习已集成）
result, err := orchestrator.CheckText(tenantID, "这是一段文本")
```

## Redis 缓存与队列

### 功能

- **检测结果缓存**：缓存检测结果，支持分层缓存策略和访问刷新机制
- **敏感词缓存**：缓存敏感词集合，TTL = 7 天
- **异步队列**：基于 Redis List 的消息队列，用于高并发场景
- **内存保护**：配置 LRU 淘汰策略，防止内存溢出

### 核心组件

#### Redis 客户端

```go
type RedisClient struct {
    client *redis.Client
}

// 缓存操作
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
func (r *RedisClient) Get(ctx context.Context, key string) (string, error)

// 带刷新的缓存获取（访问时自动延长过期时间）
func (r *RedisClient) GetWithRefresh(ctx context.Context, key string, expiration time.Duration) (string, error)

// 集合操作（用于敏感词）
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error
func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error)
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error

// 队列操作
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error
func (r *RedisClient) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error)
```

### 缓存键设计

| 键类型 | 键格式 | TTL | 说明 |
|--------|--------|-----|------|
| 检测结果 | `detection:{tenantID}:{hash}` | 分层策略 | 缓存检测结果，根据风险等级动态调整 |
| 敏感词集合 | `sensitive_words:{tenantID}` | 7 天 | 租户敏感词集合 |
| 队列 | `queue:detection` | - | 异步检测队列 |
| 队列结果 | `queue:result:{taskID}` | 24 小时 | 队列任务结果 |

### 分层缓存策略

根据检测结果的风险等级设置不同的缓存过期时间：

| 结果类型 | 风险等级 | TTL | 说明 |
|----------|----------|-----|------|
| 高风险 | >= 4 | 7 天 | 用于审计和合规 |
| 中等风险 | 1-3 | 1 天 | 普通风险结果 |
| 通过 | 0 | 1 小时 | 无风险结果 |

```go
func getCacheTTL(result *OrchestratorResult) time.Duration {
    if !result.Passed && result.RiskLevel >= 4 {
        return 7 * 24 * time.Hour  // 高风险：7天
    } else if !result.Passed {
        return 24 * time.Hour       // 中等风险：1天
    } else {
        return 1 * time.Hour        // 通过：1小时
    }
}
```

### 访问刷新机制

当缓存命中时，自动延长过期时间，确保热门数据持续保留：

```go
func (r *RedisClient) GetWithRefresh(ctx context.Context, key string, expiration time.Duration) (string, error) {
    result, err := r.client.Get(ctx, key).Result()
    if err != nil {
        return "", err
    }
    
    // 异步刷新过期时间（不阻塞主流程）
    go func() {
        r.client.Expire(ctx, key, expiration)
    }()
    
    return result, nil
}
```

### Redis LRU 淘汰策略

为防止内存溢出，配置 Redis 内存限制和 LRU 淘汰策略：

```yaml
redis:
  maxmemory: 512mb
  maxmemory-policy: allkeys-lru
```

**配置说明：**
- `maxmemory`: 内存上限，当达到此限制时触发淘汰
- `maxmemory-policy`: 淘汰策略，`allkeys-lru` 表示淘汰最久未使用的键
- 配合访问刷新机制，热门数据会被持续保留，冷门数据会被自动淘汰

## HTTP 接口

### 基础检查接口

```bash
POST /api/v1/orchestrator/check
Content-Type: application/json
X-API-Key: your-api-key

{
  "text": "这是一段文本"
}
```

### 带配置的检查接口

```bash
POST /api/v1/orchestrator/check/config
Content-Type: application/json
X-API-Key: your-api-key

{
  "text": "这是一段文本",
  "config": {
    "enable_layer1": true,
    "enable_layer2": true,
    "enable_layer3": true,
    "layer2_threshold": 0.3,
    "layer2_limit": 10
  }
}
```

### 带上下文的检查接口

```bash
POST /api/v1/orchestrator/check/context
Content-Type: application/json
X-API-Key: your-api-key

{
  "text": "这是一段文本",
  "context": {
    "scenario": "社交媒体评论",
    "user_context": "新用户",
    "temperature": 0.7
  }
}
```

### 完整检查接口

```bash
POST /api/v1/orchestrator/check/full
Content-Type: application/json
X-API-Key: your-api-key

{
  "text": "这是一段文本",
  "config": {
    "enable_layer1": true,
    "enable_layer2": true,
    "enable_layer3": true
  },
  "context": {
    "scenario": "社交媒体评论",
    "temperature": 0.7
  }
}
```

### 异步检测接口

```bash
POST /api/v1/orchestrator/check/async
Content-Type: application/json
X-API-Key: your-api-key

{
  "text": "这是一段文本"
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "task_id": "uuid-string",
    "status": "pending"
  }
}
```

### 查询异步任务结果

```bash
GET /api/v1/orchestrator/check/async/{task_id}
X-API-Key: your-api-key
```

## 响应格式

### 基础响应

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "passed": false,
    "risk_level": 3,
    "message": "文本包含敏感内容",
    "total_matches": 8,
    "layer1_result": {
      "has_match": true,
      "matched_words": [...],
      "normalized": "规范化后的文本",
      "risk_level": 3,
      "categories": ["政治"]
    },
    "layer2_result": {
      "has_match": true,
      "matched_words": [...],
      "risk_level": 3,
      "categories": ["政治"],
      "threshold": 0.3,
      "total_matches": 5
    },
    "layer3_result": {
      "has_risk": true,
      "risk_level": 3,
      "risk_reason": "文本包含敏感内容",
      "suggestions": ["修改文本内容"],
      "is_approved": false,
      "confidence": 0.9,
      "reasoning": "基于敏感词匹配结果分析",
      "detected_words": ["新检测到的违禁词"] // LLM 识别出的违禁词
    },
    "duration_ms": 150,
    "learned_words": ["新检测到的违禁词"] // 本次检测自动学习到的词
  }
}
```

### 扩展 Layer1 响应（使用扩展服务时）

```json
{
  "has_match": true,
  "matched_words": [...],
  "normalized": "规范化后的文本",
  "risk_level": 5,
  "categories": ["政治", "隐私"],
  "regex_matches": [
    {
      "pattern": "\\d{11}",
      "payload": {...},
      "position": 10,
      "matched": "13812345678",
      "match_type": "regex"
    }
  ],
  "fuzzy_matches": [
    {
      "payload": {...},
      "position": 5,
      "matched": "敏盛",
      "target": "敏感",
      "similarity": 0.85,
      "match_type": "fuzzy"
    }
  ],
  "multilang_matches": [
    {
      "payload": {...},
      "position": 20,
      "matched": "sensitive",
      "language": "en",
      "match_type": "multilang"
    }
  ]
}
```

## 配置说明

### Layer1 配置

- `enable_layer1`: 是否启用第一层
- `stop_at_layer1`: 是否在第一层停止（如果匹配到敏感词）

### Layer2 配置

- `enable_layer2`: 是否启用第二层
- `layer2_threshold`: 语义相似度阈值（0-1）
- `layer2_limit`: 返回结果数量限制
- `stop_at_layer2`: 是否在第二层停止（如果匹配到敏感词）

### Layer3 配置

- `enable_layer3`: 是否启用第三层
- `layer3_enable_reason`: 是否启用推理
- `enable_auto_learning`: 是否启用自动学习

### Redis 配置

```yaml
redis:
  addr: "localhost:16379"
  password: "jetwash-redis"
  db: 0
```

## 性能优化建议

1. **Layer1 优化**
   - 使用 AC 自动机实现 O(n) 时间复杂度的匹配
   - 文本规范化预处理减少重复计算
   - 支持 tenant_id 隔离，避免跨租户干扰
   - 扩展服务支持正则、模糊、多语言匹配

2. **Layer2 优化**
   - 使用 pgvector 索引加速向量检索
   - 合理设置阈值和限制数量
   - 使用 Filter 条件减少检索范围

3. **Layer3 优化**
   - 缓存 LLM 响应
   - 批量处理请求
   - 使用流式响应减少延迟
   - 启用自动学习功能，自动扩展敏感词库

4. **缓存优化**
   - **分层缓存策略**：根据风险等级设置不同 TTL（高风险7天，中风险1天，通过1小时）
   - **访问刷新机制**：缓存命中时自动延长过期时间，热门数据持续保留
   - **LRU 淘汰策略**：配置 Redis 内存上限和 LRU 淘汰，防止内存溢出
   - 设置合理的 TTL，平衡缓存命中率和数据新鲜度

5. **异步处理**
   - 高并发场景使用异步队列
   - 后台消费队列任务，异步返回结果

6. **Benchmark 测试**
   - 使用 `cmd/benchmark/main.go` 进行性能测试
   - 支持 `-mode` 参数选择检测模式（basic/semantic/full）
   - 支持 `-concurrent` 参数设置并发数
   - 支持 `-total` 参数设置总请求数

## 扩展建议

1. **Layer1 扩展** ✅ 已实现
   - ✅ 支持正则表达式匹配
   - ✅ 支持模糊匹配
   - ✅ 支持多语言匹配

2. **Layer2 扩展**
   - 集成更多向量模型
   - 支持自定义相似度算法
   - 支持实时更新向量索引

3. **Layer3 扩展** ✅ 部分实现
   - ✅ 支持自动学习
   - 支持更多 LLM 提供者
   - 支持自定义 Prompt 模板
   - 支持多轮对话

4. **缓存扩展**
   - 支持多级缓存策略
   - 支持缓存预热
   - 支持缓存失效策略

## 注意事项

1. **数据隔离**
   - 所有层都支持 tenant_id 隔离
   - 确保敏感词和向量数据按租户隔离
   - Redis 缓存也按租户隔离

2. **性能考虑**
   - Layer1 最快，Layer2 次之，Layer3 最慢
   - 根据业务需求选择合适的层组合
   - 可以配置在某一层停止以提高性能
   - 高并发场景建议使用异步队列

3. **准确性考虑**
   - Layer1 精确匹配，准确性最高
   - Layer2 语义匹配，可能存在误判
   - Layer3 智能推理，准确性取决于 LLM 能力
   - 自动学习功能需要人工审核确认

4. **成本考虑**
   - Layer1 和 Layer2 成本较低
   - Layer3 需要调用 LLM API，成本较高
   - 建议根据业务需求合理配置
   - 使用缓存减少 LLM 调用次数

5. **Redis 配置**
   - 生产环境建议配置 Redis 密码
   - 配置合适的内存策略
   - 考虑使用 Redis Cluster 保证高可用
