# 三层架构文档

本文档描述了 Jetwash 平台的三层文本审查架构。

## 架构概述

Jetwash 平台采用三层漏斗式架构进行文本审查，每一层都有不同的职责和特点：

1. **Layer 1: Speed (快速匹配层)** - 基于 AC 自动机的精确匹配
2. **Layer 2: Semantic (语义检索层)** - 基于 pgvector 的向量检索
3. **Layer 3: Reason (推理层)** - 基于 LLM 的智能推理

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
├── layer3_reason/         # 第三层：推理层
│   └── service.go         # 第三层服务接口（LLM Prompt 组合）
└── orchestrator/           # 编排层
    └── orchestrator.go    # 三层漏斗编排
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

### 核心组件

#### 推理服务

```go
type Layer3Service interface {
    ReasonText(tenantID uuid.UUID, text string, context *ReasonContext) (*Layer3Result, error)
    ReasonWithMatches(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) (*Layer3Result, error)
    GeneratePrompt(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) string
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

### 使用示例

```go
// 创建 Mock LLM 提供者（实际应用中应使用真实的 LLM）
mockLLMProvider := layer3_reason.NewMockLLMProvider("LLM 响应")
layer3Service := layer3_reason.NewLayer3Service(mockLLMProvider)

// 推理分析
result, err := layer3Service.ReasonText(tenantID, "这是一段文本", context)
```

## Orchestrator (编排层)

### 功能

- 三层漏斗式编排
- 灵活的配置选项
- 结果聚合和总结

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
}
```

### 使用示例

```go
// 创建编排器
orchestrator := orchestrator.NewOrchestrator(layer1Service, layer2Service, layer3Service)

// 检查文本
result, err := orchestrator.CheckText(tenantID, "这是一段文本")
```

## HTTP 接口

### 基础检查接口

```bash
POST /api/v1/orchestrator/check
Content-Type: application/json

{
  "tenant_id": "uuid",
  "text": "这是一段文本"
}
```

### 带配置的检查接口

```bash
POST /api/v1/orchestrator/check/config
Content-Type: application/json

{
  "tenant_id": "uuid",
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

{
  "tenant_id": "uuid",
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

{
  "tenant_id": "uuid",
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
      "reasoning": "基于敏感词匹配结果分析"
    },
    "total_matches": 8,
    "execution_time": 150
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

## 性能优化建议

1. **Layer1 优化**
   - 使用 AC 自动机实现 O(n) 时间复杂度的匹配
   - 文本规范化预处理减少重复计算
   - 支持 tenant_id 隔离，避免跨租户干扰

2. **Layer2 优化**
   - 使用 pgvector 索引加速向量检索
   - 合理设置阈值和限制数量
   - 使用 Filter 条件减少检索范围

3. **Layer3 优化**
   - 缓存 LLM 响应
   - 批量处理请求
   - 使用流式响应减少延迟

## 扩展建议

1. **Layer1 扩展** ✅ 已实现
   - ✅ 支持正则表达式匹配
   - ✅ 支持模糊匹配
   - ✅ 支持多语言匹配

2. **Layer2 扩展**
   - 集成更多向量模型
   - 支持自定义相似度算法
   - 支持实时更新向量索引

3. **Layer3 扩展**
   - 支持更多 LLM 提供者
   - 支持自定义 Prompt 模板
   - 支持多轮对话

## 注意事项

1. **数据隔离**
   - 所有层都支持 tenant_id 隔离
   - 确保敏感词和向量数据按租户隔离

2. **性能考虑**
   - Layer1 最快，Layer2 次之，Layer3 最慢
   - 根据业务需求选择合适的层组合
   - 可以配置在某一层停止以提高性能

3. **准确性考虑**
   - Layer1 精确匹配，准确性最高
   - Layer2 语义匹配，可能存在误判
   - Layer3 智能推理，准确性取决于 LLM 能力

4. **成本考虑**
   - Layer1 和 Layer2 成本较低
   - Layer3 需要调用 LLM API，成本较高
   - 建议根据业务需求合理配置
