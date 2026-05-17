# Jetwash 项目 — 大厂后端实习生面试亮点整理

> 多租户敏感词过滤与文本风控 SaaS 平台
> 聚焦技术方案选型与架构决策

***

## 一、项目概述

Jetwash 是一个基于 Go 语言开发的多租户敏感词过滤与文本风控 SaaS 平台。核心功能是对文本内容进行多层审查，检测其中的敏感词、违规内容等。采用**三层漏斗式架构**：AC 自动机精确匹配 → pgvector 语义检索 → LLM 智能推理，兼顾性能与准确率。

**技术栈：** Go / Gin / GORM / PostgreSQL + pgvector / Redis / Ollama（本地 LLM） / Docker Compose / Prometheus / testify / golang-migrate

***

## 二、亮点一：三层漏斗式文本审查架构

### 1. 方案背景与问题定义

文本内容审核面临三个核心矛盾：**性能、准确率、成本**。如果每次都调用 LLM，成本极高且延迟大；如果只用规则匹配，无法识别语义变体。因此需要一个多层递进的架构来平衡这三者。

### 2. 方案对比与选型

| 方案            | QPS           | LLM 调用占比    | 成本/千次           | 准确率          |
| ------------- | ------------- | ----------- | --------------- | ------------ |
| 单层 LLM 直接审核   | 2\~5          | 100%        | $1\~$10         | 高但慢          |
| 双层（规则 + LLM）  | 50\~200       | 20%\~40%    | $0.2\~$4        | 中等           |
| **三层漏斗（本项目）** | **500\~3000** | **5%\~10%** | **$0.05\~$0.5** | **95%\~98%** |

**最终选择三层漏斗架构，核心理由：**

- **性能：** 绝大多数请求在 Layer1 就被拦截（QPS 3200，P99 8ms），仅 5%\~10% 的请求需要调用 LLM
- **成本：** LLM 调用从 100% 降至 5%\~10%，成本降低 90%\~95%
- **准确率：** 前两层处理确定性内容，LLM 专注处理复杂语义场景，整体准确率达 95%\~98%

### 3. 架构设计细节

#### Layer1 和 Layer2 并发执行

使用 `goroutine` + `sync.WaitGroup` 实现 Layer1（AC 自动机）和 Layer2（向量检索）的并行执行。这样整体延迟从"Layer1 + Layer2"压缩为"max(Layer1, Layer2)"。实测语义模式下 QPS 从串行的 \~300 提升至 580。

#### 短路优化机制

当 Layer1 检测到高风险匹配（riskLevel >= 4）时直接拒绝，不再调用 LLM，避免不必要的 API 开销。可通过配置 `StopAtLayer1` / `StopAtLayer2` 灵活控制短路策略。

#### 歧义穿透机制

当 Layer1 的所有匹配都是歧义匹配时（如"白骨精"可能只是讨论西游记），不会在 Layer1 直接拦截，而是继续传递给 Layer2/3 做更精确的语义判断，有效降低了误报率。

> 💡 **面试话术**
> "我设计了一个三层漏斗架构，核心思路是用尽可能低成本的方式处理尽可能多的请求。通过让 Layer1 和 Layer2 并发执行，我们将语义模式的 QPS 从 300 提升到 580，同时 LLM 调用比例从 100% 降到 5%\~10%，成本降低了 90% 以上。"

***

## 三、亮点二：AC 自动机精确匹配与增量更新策略

### 1. 匹配算法选型

在 Layer1 的精确匹配层，需要一个能同时匹配数万个敏感词的算法。对比了以下方案：

| 算法              | 时间复杂度    | 多模式匹配          | 适合场景       |
| --------------- | -------- | -------------- | ---------- |
| Trie 单模式        | O(n)     | ✗ 需逐个查询        | 单词查找       |
| KMP             | O(n+m)   | ✗ 单模式          | 单模式匹配      |
| 正则表达式           | O(n\*m)  | ✔              | 复杂模式但慢     |
| Boyer-Moore     | O(n/m)   | ✗ 单模式          | 长模式优化      |
| **AC 自动机（本项目）** | **O(n)** | **✔ 一次扫描全部匹配** | **大规模多模式** |

**最终选择 AC 自动机**，因为它是唯一同时满足"O(n) 时间复杂度"和"多模式同时匹配"的算法。匹配时间与敏感词数量无关，非常适合敏感词库不断增长的场景。

### 2. AC 自动机更新策略

当自动学习机制发现新的违禁词时，需要将新词加入 AC 自动机。对比了五种更新策略：

| 更新策略          | 新词生效时间    | 性能影响            | 并发安全  |
| ------------- | --------- | --------------- | ----- |
| 定时全量重建        | 0\~10 分钟  | 无影响             | 优     |
| 立即全量重建        | < 1ms     | 650\~1300ms 卡顿  | 差     |
| 延迟批量重建        | 30 秒      | 低               | 中     |
| **增量更新（本项目）** | **< 1ms** | **5.6\~11.5ms** | **优** |
| 智能缓存失效        | 1s\~10min | 低\~中            | 中     |

**选择增量更新的理由：** 新词立即生效（< 1ms），性能影响仅 5.6\~11.5ms（比全量重建的 650\~1300ms 低两个数量级），通过读写锁保证并发安全。具体实现是先调用 `AddWord` 插入新节点，再调用 `BuildFail` 重建失败指针。

### 3. 多租户隔离与并发安全

每个租户维护独立的 AC 自动机实例（`map[string]*ACAutomaton`），使用**双重检查锁定（Double-Checked Locking）** 实现懒初始化：先 `RLock` 读取，未命中后升级为 `Lock`，加锁后再次检查。在读多写少场景下性能优异。自动机实例通过 `sync.Map` 缓存，10 分钟过期自动重载。

> 💡 **面试话术**
> "我选择 AC 自动机而非正则或 KMP，因为它是唯一同时满足 O(n) 时间复杂度和多模式同时匹配的算法。对于更新策略，我对比了五种方案，最终选择增量更新，新词生效时间 < 1ms，性能开销比全量重建低两个数量级。"

***

## 四、亮点三：语义检索层的向量数据库选型

### 1. 方案对比

| 方案                | 优势                          | 劣势                | 适用场景       |
| ----------------- | --------------------------- | ----------------- | ---------- |
| Milvus            | 专用向量数据库，性能强                 | 运维复杂，技术栈多一套       | 千万级向量      |
| Weaviate / Qdrant | 功能丰富，生态完善                   | 引入新依赖，学习成本高       | 中大型项目      |
| **pgvector（本项目）** | **与 PostgreSQL 技术栈统一，运维简单** | **超大规模性能不如专用数据库** | **中小规模场景** |

**选择 pgvector 的理由：** 项目已经使用 PostgreSQL 作为主数据库，引入 pgvector 扩展可以避免引入额外的向量数据库组件，降低运维复杂度。对于单租户数万条敏感词的规模，pgvector 的性能完全够用（查询延迟 10\~50ms）。同时可以利用 PostgreSQL 的事务、权限等能力，不需要在两套数据库之间做数据同步。

### 2. FilterBuilder 模式

封装了链式 API 构建过滤条件（`WithCategory`、`WithRiskLevelRange`、`WithStatus` 等），在向量检索前先缩小搜索范围，显著减少计算量。这种"先过滤再检索"的思路类似数据库中的索引下推。

> 💡 **面试话术**
> "在向量数据库选型上，我没有盲目追求最先进的方案，而是根据实际规模和技术栈做出最优选择。我们的敏感词规模在万级，pgvector 的性能完全够用，而且避免了引入额外组件的运维成本。如果未来规模增长到千万级，可以通过 Repository 接口无缝切换到 Milvus。"

***

## 五、亮点四：LLM 自动学习闭环

### 1. 设计思路

当 Layer3（LLM）检测到新的违禁词时，编排器在 `defer` 中自动执行以下操作，形成"LLM 发现 → 规则沉淀 → Layer1 拦截"的闭环：

- **步骤 1：** 将新词写入数据库（标记为 `llm_detected` 类别，默认风险等级 3）
- **步骤 2：** 将新词添加到 Redis Set（TTL 7 天）
- **步骤 3：** 增量更新 AC 自动机（`AddWord` + `BuildFail`），新词延迟 < 1ms 生效

### 2. 设计考量

这个闭环的核心价值在于：随着系统运行，Layer1 的拦截能力会持续增强，LLM 的调用比例会越来越低，形成"越用越快、越用越便宜"的正向循环。同时，新词入库后需要人工审核确认（标记为 `llm_detected` 便于筛选），避免 LLM 的幻觉导致误拦截。

> 💡 **面试话术**
> "我设计了一个自动学习闭环：LLM 发现新的违禁词后，自动沉淀到 Layer1 的 AC 自动机中。这样随着系统运行，Layer1 的拦截能力会越来越强，LLM 调用比例从初始的 10% 逐渐下降，形成正向循环。"

***

## 六、亮点五：分层 TTL 缓存策略

### 1. 设计思路

不同风险等级的检测结果具有不同的业务特性，因此设计了分层 TTL 缓存策略：

| 检测结果                | TTL   | 设计理由            |
| ------------------- | ----- | --------------- |
| 高风险（riskLevel >= 4） | 7 天   | 审计合规需要，结果需要长期保留 |
| 中等风险                | 24 小时 | 平衡缓存命中率和安全性     |
| 通过（无风险）             | 1 小时  | 合规文本可能被微调后重新检测  |

### 2. 缓存键设计

使用 SHA256 哈希文本内容生成固定 64 字符的缓存键（`detection:{tenantID}:{sha256hash}`），避免了长文本作为 Redis 键的性能问题。缓存键中包含 `tenantID`，确保多租户数据隔离。

### 3. GetWithRefresh 模式

读取缓存时异步刷新 TTL，使用独立 goroutine 和 `context.Background()` 执行 `EXPIRE` 命令，避免父 context 取消影响刷新操作。刷新失败仅记录警告日志，不影响主流程。

> 💡 **面试话术**
> "我设计了分层 TTL 缓存策略，根据风险等级设置不同的过期时间。高风险结果缓存 7 天用于审计，通过结果只缓存 1 小时以便快速响应内容变化。另外缓存键使用 SHA256 哈希而非原始文本，解决了长文本作为 Redis 键的性能问题。"

***

## 七、亮点六：LLM 提供者的策略模式与双模部署

### 1. 设计思路

通过 `LLMProvider` 接口抽象 LLM 调用，实现了四种提供者：`MockLLMProvider`（测试）、`OllamaLLMProvider`（本地部署）、`OnlineLLMProvider`（云端 API，对接智谱 AI）、`OpenAILLMProvider`（预留）。运行时通过配置文件灵活切换。

### 2. 本地优先 + 云端兜底

默认使用 Ollama 本地部署（qwen2.5:0.5b），保护数据隐私且无 API 调用成本；同时支持切换到在线 API（智谱 AI GLM-4.7），获得更强的推理能力。启动时异步检查 LLM 服务可达性，仅日志警告不阻塞启动，确保 Layer1 和 Layer2 仍可工作（降级容错）。

> 💡 **面试话术**
> "我通过接口抽象实现了 LLM 提供者的策略模式，支持本地 Ollama 和在线 API 灵活切换。本地部署保护数据隐私，云端兜底保证可用性。启动时采用软检查策略，LLM 不可用不阻塞启动，确保前两层仍可正常工作。"

***

## 八、亮点七：并发设计与性能优化

### 1. 并发模式汇总

项目中合理运用了多种 Go 并发原语：

| 并发原语                  | 应用场景                 | 选择理由          |
| --------------------- | -------------------- | ------------- |
| goroutine + WaitGroup | Layer1 和 Layer2 并发执行 | 等待多个任务完成后聚合结果 |
| sync.RWMutex          | AC 自动机读写保护           | 读多写少场景，读锁不互斥  |
| sync.Map              | 租户 AC 自动机缓存          | 高并发读取，内建分片锁   |
| 双重检查锁定                | 租户自动机懒初始化            | 避免重复初始化，减少锁竞争 |
| 独立 goroutine 异步刷新     | 缓存 TTL 刷新            | 不阻塞主流程请求      |

### 2. 异步队列

基于 Redis List 实现异步检测队列，使用 `BRPop` 阻塞式出队避免忙轮询。队列处理在独立 goroutine 中运行，与 HTTP 服务器并行。选择 Redis List 而非专用消息队列（RabbitMQ/Kafka），是因为项目已经引入 Redis，且异步检测场景的复杂度不高，不需要专用 MQ 的高级特性。

### 3. 降级容错设计

- **Redis 不可用：** 跳过缓存，直接执行检测（性能降低但功能不受影响）
- **AC 自动机插入失败：** 仅记录日志，不影响主流程检测结果
- **LLM 不可用：** 启动时软检查，不阻塞启动，Layer1 和 Layer2 仍可工作
- **检测历史保存失败：** 异步保存不阻塞主流程，失败仅记录警告日志

> 💡 **面试话术**
> "在并发设计上，我根据不同场景选择了合适的并发原语：读多写少用 RWMutex，高并发缓存用 sync.Map，懒初始化用双重检查锁定。整体设计遵循降级容错原则，任何一层的失败都不会导致整个系统不可用。"

***

## 九、亮点八：接口驱动设计与多租户架构

### 1. 接口驱动设计

项目大量使用 Go 接口定义服务契约，包括 `Layer1Service`、`Layer2Service`、`Layer3Service`、`LLMProvider`、`EmbeddingProvider`、`Orchestrator`、`QueueService`、`SemanticRepository` 等。这种设计的价值在于：

- **可测试性：** 测试时可以轻松 mock 任意层的依赖（如 `MockLLMProvider`）
- **可替换性：** 通过接口可以无缝切换实现（如未来从 pgvector 切换到 Milvus）
- **依赖倒置：** 业务层依赖接口而非具体实现，符合 SOLID 原则

### 2. 多租户架构

基于 `TenantID` 实现数据隔离，支持租户层级结构（`parent_id`，最多 5 层）。每个租户拥有独立的 AC 自动机、API Key、敏感词库和缓存空间。认证中间件支持 JWT Token 和 API Key 双模式，适用于"用户登录 + 服务间调用"的混合场景。

> 💡 **面试话术**
> "项目采用接口驱动设计，所有服务都通过接口定义而非具体实现。这样在测试时可以 mock 任意层的依赖，未来如果需要替换向量数据库或 LLM 提供者，只需要实现新的接口即可，不需要改动业务逻辑。"

***

## 十、亮点九：非敏感词快速放行（Fast Pass）优化

### 1. 问题发现

内容审核系统有一个典型的业务特征：**95%+ 的文本都是非敏感词**。但在原始架构中，Layer1 和 Layer2 是无条件并发执行的——即使 Layer1 在 \~5ms 内就判断出文本完全正常，Layer2 仍然会白白执行一次 Embedding API 调用（50\~300ms）。在高并发下，大量无效的 Embedding 请求会成为整个系统的吞吐瓶颈。

### 2. 方案对比

| 方案                         | 非敏感词延迟         | 敏感词延迟          | 实现复杂度 | 缺点                  |
| -------------------------- | -------------- | -------------- | ----- | ------------------- |
| 保持并发（原方案）                  | 100\~300ms     | 100\~300ms     | 低     | 非敏感词浪费 Embedding 调用 |
| **Layer1 优先串行（Fast Pass）** | **\~5ms**      | **100\~300ms** | **低** | 敏感词路径失去并发优势（但占比极小）  |
| Embedding 结果缓存             | 100\~300ms（首次） | 100\~300ms     | 中     | 仅对重复文本有效，缓存管理复杂     |
| 动态并发策略                     | \~5ms          | 100\~300ms     | 高     | 增加调度复杂度，收益与串行方案相同   |

**最终选择 Layer1 优先串行（Fast Pass）**，理由：

- 敏感词仅占 5% 以下，失去并发优势的影响极小
- 实现简单，仅修改编排器调度逻辑，不影响各层内部实现
- 非敏感词延迟从 100\~300ms 降到 \~5ms，QPS 提升 20\~60 倍
- 通过配置开关 `EnableFastPass` 控制，可随时回退到原始并发模式

### 3. 实现细节

在编排器中新增 `EnableFastPass` 配置项（默认开启），核心逻辑：

```
EnableFastPass 开启时：
  1. 先单独执行 Layer1（AC 自动机，~5ms）
  2. Layer1 无匹配 → 直接返回通过，跳过 Layer2/3（快速放行）
  3. Layer1 有匹配 → 仅启动 Layer2 继续检测（Layer1 已完成无需重复）
  4. 最终由 Layer3 做最终判断

EnableFastPass 关闭时：
  → 回退到原始的 Layer1 + Layer2 并发执行模式
```

**关键设计决策：**

- **可配置回退：** 通过 `EnableFastPass` 开关控制，不破坏原有并发逻辑，关闭后完全回退
- **Fast Pass 路径下 Layer1 有匹配时不并发：** 因为 Layer1 已执行完毕，只需串行启动 Layer2，无需再起 goroutine 做并发
- **defer 逻辑不受影响：** 检测历史保存、缓存写入等善后逻辑统一由 defer 处理，无论走哪条路径都会执行

### 4. 优化效果

| 场景                | 优化前延迟      | 优化后延迟      | QPS 提升      |
| ----------------- | ---------- | ---------- | ----------- |
| 非敏感词（95%+）        | 100\~300ms | **\~5ms**  | **20\~60x** |
| Layer1 直接命中的敏感词   | \~5ms      | \~5ms      | 不变          |
| 需要语义/LLM 判断的（<5%） | 100\~300ms | 100\~300ms | 不变（占比极小）    |

> 💡 **面试话术**
> "我发现实际场景中 95% 以上的文本都是非敏感词，但原始架构下每次请求都会无条件调用 Embedding API（50\~300ms），这是巨大的浪费。所以我设计了 Fast Pass 快速放行机制：先执行 Layer1 AC 自动机（\~5ms），如果无匹配就直接返回，跳过 Layer2 和 Layer3。这样非敏感词的延迟从 100ms 降到 5ms，QPS 提升了 20\~60 倍。同时通过配置开关保留了回退到原始并发模式的能力。"

---

## 十一、亮点十：异步写入优化——将非关键路径 DB 操作从请求链路剥离

### 1. 问题发现

在压测中发现，即使 94% 的请求走了 Fast Pass（Layer1 \~5ms 就完成检测），QPS 仍然只有 **8.95**，平均响应时间 **111ms**。通过火焰图分析定位到瓶颈：`defer` 中的 `SaveDetectionHistory` 是**同步阻塞的数据库事务写入**，每次请求包含 2+ 次 SQL INSERT，耗时 50\~100ms，直接阻塞了检测结果的返回。

```
用户感知延迟 = Layer1检测(~5ms) + SaveDetectionHistory(~50-100ms) + Redis写入(~1ms)
                                    ↑ 这里是瓶颈，但完全不需要阻塞返回
```

### 2. 方案对比

| 方案 | 请求延迟 | 实现复杂度 | 数据可靠性 | 缺点 |
|------|---------|-----------|-----------|------|
| 同步写入（原方案） | +50\~100ms | 低 | 强一致 | 阻塞返回 |
| goroutine 异步（每次起协程） | +0.1ms | 低 | 弱 | 高并发时 goroutine 爆炸，无法控制并发度 |
| **Channel + Worker（本项目）** | **+0.001ms** | **中** | **最终一致** | channel 满时丢弃任务 |
| 消息队列（Kafka/RabbitMQ） | +0.001ms | 高 | 最终一致 | 引入新组件，运维成本高 |

**最终选择 Channel + Worker**，理由：
- 带缓冲 channel（容量 1000）天然提供背压控制，避免 goroutine 爆炸
- 单 worker 串行消费，DB 连接池压力可控
- 不引入新组件，与现有架构一致
- 检测历史属于审计日志，允许最终一致性，短暂延迟可接受

### 3. 实现细节

```go
// orchestrator 结构体新增带缓冲 channel
historyChan chan *asyncTask  // 容量 1000

// NewOrchestrator 中启动后台 worker
go o.asyncWorker()

// asyncWorker 后台消费：检测历史写入 + 新词入库 + 缓存写入
func (o *orchestrator) asyncWorker() {
    for task := range o.historyChan {
        o.detectionHistoryService.SaveDetectionHistory(...)
        o.addDetectedWordsToDatabase(...)
        o.redisClient.Set(...)
    }
}

// defer 中改为非阻塞发送
select {
case o.historyChan <- task:  // 成功入队
default:                     // channel 满时丢弃并告警，不阻塞请求
    logger.Warn("channel full, dropping task")
}
```

**关键设计决策：**

- **select + default 非阻塞发送：** channel 满时不阻塞请求，丢弃任务并记录告警。检测历史是审计日志，丢失少量记录可接受，但不能影响核心检测链路
- **单 worker 串行消费：** 避免 DB 连接池被并发写入打满，与检测请求共享连接池
- **channel 容量 1000：** 足够吸收突发流量，正常情况下 worker 消费速度远大于生产速度

### 4. 优化效果

| 场景 | 优化前 | 优化后 |
|------|--------|--------|
| 非敏感词请求延迟 | \~111ms | **\~7ms**（5ms检测 + 1ms缓存查询 + <1μs channel发送） |
| QPS（5并发） | 8.95 | **\~700+** |
| DB 写入可靠性 | 强一致 | 最终一致（延迟 < 1s） |

> 💡 **面试话术**
> "压测发现 QPS 只有 9，但 94% 的请求在 5ms 内就完成了检测。我用火焰图定位到瓶颈在 defer 里的同步 DB 写入——每次请求都要等事务提交才能返回。我把检测历史、缓存写入这些非关键操作改成了 channel + worker 异步模式，请求链路上只增加了一次 channel 发送（< 1μs），QPS 从 9 提升到了 700+。channel 满的时候用 select + default 非阻塞丢弃并告警，保证核心检测链路不被阻塞。"

---

## 十二、亮点十一：LLM 超时控制与降级策略

### 1. 问题发现

异步写入优化后，进一步压测发现 QPS 仍然只有 7。通过数据反推发现：**2% 的 LLM 请求（每个耗时 6~14 秒）占据了总耗时的 87%**。即使 90% 的请求在 5ms 内完成，2 个 LLM 慢请求就足以拖垮整体吞吐量。这就是典型的**长尾请求问题**。

### 2. 方案对比

| 方案 | 效果 | 缺点 |
|------|------|------|
| 优化 Prompt 减少 token | 对 0.5B 小模型无效甚至负优化 | 模型太小，指令太短反而输出不稳定 |
| LLM 请求完全异步化 | 响应立即返回 | 需要回调机制，架构变复杂 |
| **超时 + 规则降级（本项目）** | **兜底保护，可控延迟** | **超时后判断精度略降** |
| 多 LLM 实例并行 | 提高并发 | 需要更多 GPU/内存资源 |

**最终选择超时 + 规则降级**，理由：
- 不改变架构，仅增加超时控制逻辑
- 超时后基于已有的 Layer1/Layer2 匹配结果做规则判断，精度损失可控
- 可配置超时时间，适应不同硬件环境

### 3. 实现细节

```go
// OrchestratorConfig 新增配置
Layer3TimeoutMs: 3000  // LLM 推理超时 3 秒

// 编排器中用 context.WithTimeout + select 实现
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

go func() {
    result, err := layer3Service.ReasonWithMatches(...)
    // 通过 channel 返回结果
}()

select {
case r := <-done:    // LLM 正常返回
case <-ctx.Done():   // 超时！降级为规则判断
    if maxRiskLevel >= 3 → 拒绝（基于已有匹配）
    else              → 通过（低风险匹配未达阈值）
}
```

**降级策略设计：**
- 风险等级 >= 3 的已有匹配 → 降级时直接拒绝（宁可误杀不可漏放）
- 风险等级 < 3 的已有匹配 → 降级时保守放行
- 超时不视为错误，返回降级结果而非 error

### 4. 优化效果

| 场景 | 优化前 | 优化后 |
|------|--------|--------|
| LLM 单次最大耗时 | 6~14 秒（无上限） | **3 秒（硬限制）** |
| LLM 超时请求的处理 | 阻塞等待 | **立即降级返回** |
| 整体 QPS（10% 敏感词） | 7~9 | **~15+**（受 LLM 超时上限控制） |

> 💡 **面试话术**
> "压测发现 2% 的 LLM 请求每个耗时 6~14 秒，占据了总耗时的 87%，直接决定了整体 QPS。我给 LLM 调用加了 3 秒超时控制，超时后降级为基于已有匹配结果的规则判断——有中等风险匹配就拒绝，否则放行。这样把 LLM 的最大耗时从无上限降到了 3 秒硬限制，保证了整体吞吐量的下限。"

---

## 十三、亮点十二：LLM 完全异步化——基于 Redis Stream 的异步审核架构

### 1. 问题发现

即使有了 Fast Pass（90% 请求 ~5ms）和 LLM 超时控制（3 秒兜底），QPS 仍然被 LLM 调用拖累。根本原因是：**LLM 推理根本不需要在请求的关键路径上**。对于"先发布后审核"场景，可以先返回基于 Layer1/2 的初步结果，LLM 推理放到后台异步执行。

### 2. 方案对比

| 方案 | 响应延迟 | 结果一致性 | 架构复杂度 |
|------|---------|-----------|-----------|
| 同步等待 LLM | 6~14 秒 | 强一致 | 低 |
| LLM 超时降级 | 最多 3 秒 | 降级时精度略低 | 低 |
| **Redis Stream 异步 LLM（本项目）** | **~150ms** | **最终一致** | **中** |
| Kafka 异步 LLM | ~150ms | 最终一致 | 高（引入新组件） |

**最终选择 Redis Stream**，理由：
- 项目已使用 Redis，无需引入 Kafka/RabbitMQ 等新组件
- Redis Stream 支持消费组（Consumer Group）、消息 ACK、消息回溯，比 Redis List 更适合 MQ 场景
- 设置 MaxLen=10000 防止 Stream 无限增长

### 3. 架构设计

```
同步路径（请求链路）：
  请求 → Redis缓存查询(~1ms) → Layer1 AC自动机(~5ms) → Layer2 Embedding(~150ms)
       → 高风险? → 直接拒绝
       → 无匹配? → 直接通过
       → 需要LLM? → 发送到Redis Stream → 立即返回初步结果（~150ms）

异步路径（后台 Worker）：
  Redis Stream → LLM Review Worker 消费 → 调用 LLM 推理
       → 发现新敏感词 → 增量更新 AC 自动机 + 写入 DB + 更新 Redis Set
       → 未发现 → 无需操作
```

### 4. API 响应变化

```json
// 异步审核模式下的响应
{
  "passed": true,
  "risk_level": 0,
  "message": "文本审查通过（初步结果，LLM异步审核中）",
  "review_id": "550e8400-e29b-41d4-a716-446655440000",
  "review_status": "pending_llm_review"
}
```

### 5. 关键设计决策

- **降级容错：** Redis Stream 发送失败时，自动降级为同步超时模式，不丢失 LLM 审核能力
- **初步结果保守策略：** `maxRiskLevel >= 3` 时初步结果为拒绝（宁可误杀），否则放行
- **Worker 单线程消费：** 避免并发调用 LLM 导致 Ollama 过载（本地 LLM 通常是单线程推理）
- **消费组模式：** 支持多实例部署，每条消息只会被一个 Worker 消费

### 6. 优化效果

| 场景 | 优化前 | 优化后 |
|------|--------|--------|
| 需要 LLM 审核的请求延迟 | 3~14 秒 | **~150ms**（立即返回初步结果） |
| 整体 QPS（10% 敏感词） | 7~15 | **~600+**（仅受 Layer2 Embedding 限制） |
| LLM 调用 | 阻塞请求链路 | 完全异步，不影响响应 |

> 💡 **面试话术**
> "我发现 LLM 推理根本不需要在请求的关键路径上。对于'先发布后审核'场景，我先返回基于 Layer1/2 的初步结果（~150ms），把 LLM 推理任务发到 Redis Stream 由后台 Worker 异步执行。如果 LLM 发现了新的敏感词，就增量更新到 AC 自动机里，后续请求就能在 Layer1 直接拦截。这样 LLM 的延迟从请求链路上完全剥离，QPS 从 15 提升到了 600+。用 Redis Stream 而不是 Kafka，是因为项目已经用了 Redis，不想引入额外组件。"

---

## 十四、亮点十三：结构化 LLM 输出——从脆弱文本解析到 JSON Schema

### 1. 问题发现

原始的 `parseLLMResponse` 使用 `strings.Split` + `fmt.Sscanf` 解析 LLM 的自由文本输出。这种解析方式极其脆弱——LLM 可能返回不同格式（多一个空格、少一个换行、中英文混用），导致解析失败或字段错位。

### 2. 方案对比

| 方案 | 可靠性 | 实现复杂度 | 兼容性 |
|------|--------|-----------|--------|
| 文本解析（原方案） | 低，格式稍有变化就失败 | 低 | 仅支持固定格式 |
| Function Calling | 高，结构化输出 | 中 | 仅部分 API 支持 |
| **JSON Schema + 回退（本项目）** | **高，失败自动回退** | **低** | **兼容所有 LLM** |

**最终选择 JSON Schema + 文本回退**，理由：
- Prompt 中明确要求 LLM 返回 JSON 格式，并给出示例
- 解析时先提取 `{...}` 块（兼容 markdown 包裹），再 `json.Unmarshal`
- JSON 解析失败时自动回退到旧的文本解析器，保证向后兼容
- 对 `risk_level`（0-5）和 `confidence`（0.0-1.0）做范围校验，防止 LLM 幻觉输出越界值

### 3. 实现细节

```go
func (s *layer3Service) parseLLMResponseJSON(response string) *Layer3Result {
    // 1. 提取 JSON 块（兼容 markdown ```json 包裹）
    jsonStr := response
    if idx := strings.Index(response, "{"); idx >= 0 {
        if endIdx := strings.LastIndex(response, "}"); endIdx > idx {
            jsonStr = response[idx : endIdx+1]
        }
    }

    // 2. 反序列化到结构体
    var parsed struct {
        RiskLevel     int      `json:"risk_level"`
        HasRisk       bool     `json:"has_risk"`
        RiskReason    string   `json:"risk_reason"`
        DetectedWords []string `json:"detected_words"`
        Suggestions   []string `json:"suggestions"`
        IsApproved    bool     `json:"is_approved"`
        Confidence    float64  `json:"confidence"`
        Reasoning     string   `json:"reasoning"`
    }

    if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
        log.Printf("WARN: JSON parsing failed, falling back to text: %v", err)
        return s.parseLLMResponse(response)  // 回退到文本解析
    }

    // 3. 范围校验（防御 LLM 幻觉）
    result.RiskLevel = clamp(parsed.RiskLevel, 0, 5)
    result.Confidence = clamp(parsed.Confidence, 0.0, 1.0)
    return result
}
```

> 💡 **面试话术**
> "LLM 的输出格式不稳定是常见问题。我设计了 JSON Schema + 文本回退的双层解析策略：先要求 LLM 返回 JSON，解析失败时自动回退到旧的文本解析器。同时对 risk_level 和 confidence 做范围校验，防止 LLM 输出越界值。这样既提高了可靠性，又保证了向后兼容。"

---

## 十五、亮点十四：Embedding 缓存——减少重复 API 调用

### 1. 问题发现

Layer2 语义检索每次请求都调用 Embedding API（50~300ms），但实际场景中大量文本是重复的（如模板化的评论、常见违规内容）。相同文本的 Embedding 结果是确定性的，完全可以通过缓存避免重复调用。

### 2. 实现方案

```go
// 缓存键：SHA256(text)，前缀 "embed:"
key := "embed:" + hex.EncodeToString(sha256.Sum256([]byte(text))[:])

// 序列化：float32 → []byte（binary.LittleEndian）
data := make([]byte, len(embedding)*4)
for i, f := range embedding {
    binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
}
redis.Set(ctx, key, data, 24*time.Hour)
```

**设计决策：**
- **TTL 24 小时**：Embedding 模型是确定性的，相同文本永远返回相同向量，24 小时缓存足够
- **二进制序列化**：使用 `encoding/binary` 而非 JSON，存储空间减少 60%+，序列化/反序列化性能更高
- **SHA256 哈希键**：固定 64 字符，避免长文本作为 Redis 键

> 💡 **面试话术**
> "我为 Embedding API 添加了 Redis 缓存，相同文本 24 小时内直接返回缓存结果。使用 SHA256 哈希作为缓存键，二进制序列化减少存储空间。这在重复文本较多的场景下可以减少 80%+ 的 API 调用。"

---

## 十六、亮点十五：LLM 可观测性——Prometheus 指标体系

### 1. 设计思路

LLM 调用是系统中最不稳定、最昂贵的环节，需要完善的可观测性。通过 Prometheus 指标可以实时监控 LLM 的健康状态、成本消耗和性能瓶颈。

### 2. 指标设计

| 指标 | 类型 | 标签 | 用途 |
|------|------|------|------|
| `llm_request_duration_seconds` | Histogram | model, status | 监控 LLM 延迟分布 |
| `llm_tokens_total` | Counter | model, type(prompt/completion) | 统计 Token 消耗成本 |
| `llm_requests_total` | Counter | model, status(success/error) | 监控成功率和错误率 |

### 3. 实现细节

```go
// 在 OnlineLLMProvider.GenerateTextWithMessages 中埋点
start := time.Now()
resp, err := o.httpClient.Do(req)
if err != nil {
    metrics.LLMRequests.WithLabelValues(o.Model, "error").Inc()
    metrics.LLMLatency.WithLabelValues(o.Model, "error").Observe(time.Since(start).Seconds())
    return "", err
}
metrics.LLMRequests.WithLabelValues(o.Model, "success").Inc()
metrics.LLMLatency.WithLabelValues(o.Model, "success").Observe(time.Since(start).Seconds())
metrics.LLMTokens.WithLabelValues(o.Model, "prompt").Add(float64(resp.Usage.PromptTokens))
metrics.LLMTokens.WithLabelValues(o.Model, "completion").Add(float64(resp.Usage.CompletionTokens))
```

**`/metrics` 端点**：通过 `promhttp.Handler()` 暴露，Grafana 可直接对接。

> 💡 **面试话术**
> "LLM 调用是最不稳定和最昂贵的环节，所以我添加了完整的 Prometheus 指标：延迟直方图监控 P99、Token 计数器追踪成本、请求计数器监控成功率。通过 Grafana 可以实时看到 LLM 的健康状态，及时发现异常。"

---

## 十七、亮点十六：优雅关闭——Context 传播与 Channel 排空

### 1. 问题发现

原始的 `asyncWorker` 和 `Queue.Process` 是无限循环，没有信号处理。服务器重启时，in-flight 的异步任务（检测历史保存、新词入库）直接丢失。

### 2. 实现方案

```go
// main.go：创建可取消的 context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 监听 OS 信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// 先取消 context，通知所有 worker 停止
cancel()

// HTTP 服务器优雅关闭（10 秒超时）
srv.Shutdown(ctx)

// asyncWorker：select 监听 ctx.Done()，关闭时排空 channel
func (o *orchestrator) asyncWorker(ctx context.Context) {
    for {
        select {
        case task := <-o.historyChan:
            o.processAsyncTask(task)
        case <-ctx.Done():
            // 排空剩余任务
            for {
                select {
                case task := <-o.historyChan:
                    o.processAsyncTask(task)
                default:
                    return  // channel 已空，安全退出
                }
            }
        }
    }
}
```

**关键设计决策：**
- **Context 传播**：所有 worker 共享同一个 cancellable context，cancel() 时一起停止
- **Channel 排空**：收到停止信号后，先排空 channel 中的剩余任务再退出，确保不丢数据
- **BRPop 超时**：Redis `BRPop` 从永久阻塞改为 2 秒超时，避免无法响应 context 取消

> 💡 **面试话术**
> "我用 context.WithCancel 统一管理所有后台 worker 的生命周期。收到 SIGTERM 后，先取消 context 通知所有 worker，然后排空 channel 中的剩余任务，最后关闭 HTTP 服务器。这样保证了重启时不会丢失 in-flight 的异步任务。"

---

## 十八、亮点十七：统一响应格式与错误码体系

### 1. 设计思路

原始代码中 handler 层直接使用 `gin.H{"code": 500, "message": "error"}` 返回错误，HTTP 状态码和业务错误码混在一起，格式不统一。

### 2. 实现方案

```go
// 统一响应结构
type Response struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

// 统一错误码
type Ecode interface {
    Code() int
    Message() string
}

// HTTP 状态码映射
func mapCodeToHTTPStatus(e Ecode) int {
    switch e.Code() {
    case 1001: return 401  // Unauthorized
    case 1002: return 403  // Forbidden
    case 1003: return 404  // NotFound
    case 1004: return 400  // InvalidParams
    case 1005: return 500  // Internal
    case 1006: return 429  // RateLimitExceeded
    case 1007: return 409  // Conflict
    }
}

// Handler 层使用
response.Error(c, ecode.ErrUnauthorized)
response.OK(c, data)
response.Created(c, data)
```

> 💡 **面试话术**
> "我设计了统一的响应格式和错误码体系。业务错误码通过接口抽象，HTTP 状态码映射集中管理。Handler 层只需要调用 response.OK 或 response.Error，不需要关心具体的 HTTP 状态码和 JSON 格式。"

---

## 十九、亮点十八：工程化测试实践

### 1. 测试策略

| 测试类型 | 覆盖范围 | 工具 |
|---------|---------|------|
| 单元测试 | 各层 Service 独立测试 | testify + 手动 mock |
| 集成测试 | Orchestrator 三层联动 | 各层 mock 组合 |
| 并发安全测试 | AC 自动机多 goroutine 读写 | `go test -race` |

### 2. 测试亮点

**Table-Driven Tests**（Go 惯例）：
```go
tests := []struct {
    name  string
    input string
    want  string
}{
    {"lowercase", "Hello", "hello"},
    {"trim spaces", "  hello  ", "hello"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := svc.NormalizeText(tt.input)
        assert.Equal(t, tt.want, got)
    })
}
```

**MockLLMProvider**（接口驱动的可测试性）：
```go
type MockLLMProvider struct {
    response string
    err      error
}
func (m *MockLLMProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
    return m.response, m.err
}
```

**并发安全测试**：
```go
func TestCheckText_ConcurrentAccess(t *testing.T) {
    svc := NewLayer1Service()
    svc.Initialize(tenantID, words)
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            result, err := svc.CheckText(tenantID, "text")
            assert.NoError(t, err)
            assert.True(t, result.HasMatch)
        }()
    }
    wg.Wait()
}
```

> 💡 **面试话术**
> "项目有完整的单元测试覆盖，使用 testify 做断言，table-driven 模式组织测试用例。Mock 对象通过接口实现，可以轻松隔离依赖。对于 AC 自动机这种并发组件，用 `go test -race` 做竞态检测，确保读写锁的正确性。"

---

## 二十、亮点十九：租户级限流与安全防护

### 1. 设计方案

基于 Redis 的固定窗口限流，每个租户独立计数：

```go
func RateLimit(redisClient *cache.RedisClient, requestsPerMinute int) gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID, _ := GetTenantID(c)
        key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix()/60)

        count, err := redisClient.Incr(c.Request.Context(), key)
        if err != nil {
            c.Next()  // Redis 故障时放行（fail-open）
            return
        }
        if count == 1 {
            redisClient.Expire(c.Request.Context(), key, 2*time.Minute)
        }
        if count > int64(requestsPerMinute) {
            c.Header("Retry-After", "60")
            c.JSON(429, gin.H{"code": 1006, "message": "Rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**关键设计决策：**
- **Fail-Open**：Redis 故障时放行请求，不因限流组件故障导致整个系统不可用
- **固定窗口**：使用 `time.Now().Unix()/60` 作为窗口标识，简单高效
- **Retry-After 头**：告知客户端何时可以重试，符合 HTTP 规范
- **Auth 后置**：限流中间件在鉴权之后执行，确保 `tenant_id` 已解析

> 💡 **面试话术**
> "我用 Redis 实现了租户级限流，每个租户每分钟 60 次请求。采用 fail-open 策略，Redis 故障时放行而不是阻塞所有请求。限流中间件放在鉴权之后，确保 tenant_id 已经解析。超限时返回 429 状态码和 Retry-After 头，符合 HTTP 规范。"

---

## 二十一、亮点二十：数据库版本化迁移

### 1. 设计思路

原始方案只有 `docs/SCHEMA.sql` 文件，没有迁移工具。每次 schema 变更需要手动执行 SQL，容易出错且无法追溯历史。

### 2. 实现方案

集成 `golang-migrate/migrate`，将 schema 拆分为版本化迁移文件：

```
migrations/
├── 000001_init.up.sql    # 创建表、索引、初始数据
└── 000001_init.down.sql  # 回滚：按依赖反序 DROP
```

Makefile 集成：
```makefile
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_NAME ?= jetwash
DB_SSLMODE ?= disable

MIGRATE_CMD=migrate -path migrations -database \
  "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"

migrate-up:
	$(MIGRATE_CMD) up

migrate-down:
	$(MIGRATE_CMD) down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name
```

> 💡 **面试话术**
> "我用 golang-migrate 实现了数据库版本化迁移。每个迁移文件有 up 和 down 两个版本，支持回滚。通过 Makefile 封装常用命令，团队成员不需要记复杂的 CLI 参数。迁移文件纳入 Git 版本管理，schema 变更可追溯。"

---

## 二十二、总结：面试亮点速查表

| 序号 | 亮点            | 核心关键词            | 可引导的讨论方向   |
| -- | ------------- | ---------------- | ---------- |
| 1  | 三层漏斗架构        | 性能/成本/准确率平衡      | 系统设计、方案对比  |
| 2  | AC 自动机增量更新    | 算法选型、并发安全        | 数据结构与算法    |
| 3  | pgvector 向量检索 | 技术栈统一、运维成本       | 数据库选型      |
| 4  | LLM 自动学习闭环    | 知识沉淀、正向循环        | 系统设计、AI 工程 |
| 5  | 分层 TTL 缓存     | 缓存策略、哈希键设计       | 缓存设计       |
| 6  | LLM 提供者策略模式   | 接口抽象、降级容错        | 设计模式       |
| 7  | 并发设计与性能优化     | goroutine、读写锁、降级 | Go 并发编程    |
| 8  | 接口驱动与多租户      | SOLID、依赖倒置、数据隔离  | 架构设计       |
| 9  | 非敏感词快速放行      | 业务特征驱动优化、可配置回退   | 性能优化、系统设计  |
| 10 | 异步写入优化        | 关键路径剥离、channel+worker | 性能调优、Go 并发 |
| 11 | LLM 超时控制与降级   | 长尾请求、超时兜底、规则降级  | 性能调优、系统设计 |
| 12 | LLM 完全异步化      | Redis Stream、最终一致性、异步审核 | 消息队列、系统设计 |
| 13 | 结构化 LLM 输出    | JSON Schema、回退策略、范围校验 | LLM 工程化    |
| 14 | Embedding 缓存   | 二进制序列化、SHA256 键、TTL | 缓存优化       |
| 15 | LLM 可观测性       | Prometheus、Token 追踪、延迟直方图 | 可观测性      |
| 16 | 优雅关闭          | Context 传播、Channel 排空、信号处理 | Go 工程化     |
| 17 | 统一响应格式        | Ecode 接口、HTTP 映射、错误码体系 | API 设计      |
| 18 | 工程化测试实践       | testify、table-driven、竞态检测 | 测试与质量     |
| 19 | 租户级限流         | Redis 限流、fail-open、429 | 安全防护       |
| 20 | 数据库版本化迁移      | golang-migrate、up/down、Makefile | 工程化实践     |

### 面试策略建议

- 根据岗位方向重点准备 **3\~5 个亮点**，不需要全部展示
- **后端方向**重点准备：1(三层架构)、2(AC自动机)、7(并发设计)、10(异步写入)、16(优雅关闭)、17(统一响应)
- **Agent/LLM 方向**重点准备：4(自动学习闭环)、6(LLM策略模式)、11(超时降级)、12(异步LLM)、13(结构化输出)、15(可观测性)
- 每个亮点都用 **"问题 → 方案对比 → 最终选择 → 效果"** 的结构讲述
- 准备好被追问的细节（如"为什么不用 Kafka""增量更新的并发安全怎么保证""Fast Pass 对敏感词路径有什么影响""channel 满了怎么办""LLM 超时降级后精度损失多少""异步审核如何保证不丢消息""JSON 解析失败怎么办""限流策略是 fail-open 还是 fail-close"）
- 展示性能数据（QPS 3200/580/2.5、P99 8ms/45ms/1200ms、Fast Pass 后非敏感词 \~5ms、异步写入后 QPS 9→700+、LLM 超时后最大耗时 3s、异步 LLM 后 QPS ~600+）增强说服力
- 展示工程化能力：单元测试覆盖率、优雅关闭、数据库迁移、Prometheus 监控、统一错误码

