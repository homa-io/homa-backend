# AI Bot Feature Analysis and Architecture Proposal

## Executive Summary

This document analyzes the proposed AI bot features for Homa and provides architectural recommendations, critiques, and suggestions for implementation.

---

## 1. Current System Understanding

### 1.1 Existing Architecture

Homa is built on a modular app architecture with:
- **Event-driven messaging**: NATS pub/sub for real-time updates
- **Webhook system**: Async HTTP callbacks with delivery logging
- **Client management**: UUID-based clients with language/timezone support
- **Knowledge Base**: Existing models for articles, categories, chunks, and media
- **Multi-authentication**: JWT, API keys, conversation secrets

### 1.2 Relevant Existing Components

| Component | Location | Relevance to AI |
|-----------|----------|-----------------|
| `apps/ai/` | AI app shell | Base for AI features |
| `KnowledgeBaseArticle` | models | Content source for RAG |
| `KnowledgeBaseChunk` | models | Already has chunking concept |
| `Client.Language` | models | User language detection |
| `NATS` | apps/nats | Real-time message delivery |
| `Webhook` | apps/webhook | External event broadcasting |

### 1.3 Message Flow Entry Points

```
Client Message → POST /api/client/conversations/{id}/{secret}/messages
                        ↓
              GORM AfterCreate Hook
                        ↓
              NATS publish "conversation.{id}"
                        ↓
              Webhook broadcast "message.created"
```

**Key Insight**: AI bot should intercept at the AfterCreate hook level before human handover.

---

## 2. Proposed Feature Analysis

### 2.1 Feature: AI Bot Response via OpenAI API

**Requirement**: On user request, AI bot responds using OpenAI API.

**Recommended Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Response Pipeline                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Client Message                                                  │
│       ↓                                                          │
│  Message AfterCreate Hook                                        │
│       ↓                                                          │
│  ┌─────────────────┐                                            │
│  │ AI Interceptor  │ ─→ Check: Is conversation AI-enabled?      │
│  └────────┬────────┘    Check: Is department AI-configured?     │
│           ↓                                                      │
│  ┌─────────────────┐                                            │
│  │ Context Builder │ ─→ Gather: Conversation history            │
│  └────────┬────────┘    Gather: Client info (language, etc.)    │
│           ↓             Gather: KB context (from Qdrant)        │
│  ┌─────────────────┐                                            │
│  │ Workflow Engine │ ─→ Execute: Pre-configured workflow        │
│  └────────┬────────┘    Check: JS Func requirements             │
│           ↓                                                      │
│  ┌─────────────────┐                                            │
│  │  OpenAI Client  │ ─→ Send: System prompt + context           │
│  └────────┬────────┘    Send: Conversation history              │
│           ↓             Receive: AI response                    │
│  ┌─────────────────┐                                            │
│  │ Post-Processor  │ ─→ Translate to user language              │
│  └────────┬────────┘    Apply tone/style                        │
│           ↓             Execute JS Func if needed               │
│  ┌─────────────────┐                                            │
│  │ Response Writer │ ─→ Create bot message                      │
│  └────────┬────────┘    Publish to NATS                         │
│           ↓             Broadcast webhook                       │
│  Client receives response                                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Data Models Required**:

```go
// AIConfiguration - Per department or global AI settings
type AIConfiguration struct {
    ID              uint   `gorm:"primaryKey"`
    DepartmentID    *uint  `gorm:"index"` // null = global default
    Enabled         bool
    Provider        string // "openai", "anthropic", "azure_openai"
    Model           string // "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"
    APIKey          string `json:"-"` // Encrypted storage
    Temperature     float32
    MaxTokens       int
    SystemPrompt    string // Base system prompt
    ToneInstructions string // Tone configuration
    BehaviorGuide   string // How AI should behave
    MaxHistoryMsgs  int    // How many messages to include as context
    EnableKB        bool   // Use knowledge base for RAG
    EnableJSFunc    bool   // Allow JS function calls
    EnableWorkflows bool   // Enable visual workflows
    HandoverTriggers string // JSON: conditions for human handover
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// AIConversationState - Track AI state per conversation
type AIConversationState struct {
    ConversationID uint   `gorm:"primaryKey"`
    Mode           string // "ai", "human", "hybrid"
    WorkflowID     *uint  // Current workflow if any
    WorkflowState  string // JSON: current workflow state
    HandoverReason *string
    HandoverAt     *time.Time
    AIMessageCount int
    LastAIResponse time.Time
}
```

**Critique & Improvements**:

1. **Rate Limiting**: Add per-client rate limiting to prevent API abuse
2. **Cost Control**: Track token usage per department/conversation
3. **Fallback Chain**: OpenAI → Anthropic → Azure (configurable)
4. **Caching**: Cache frequent KB queries to reduce Qdrant calls
5. **Streaming**: Consider SSE/WebSocket streaming for long responses

---

### 2.2 Feature: Knowledge Base with Qdrant Vector Search

**Requirement**: Use Qdrant to index KB articles and retrieve relevant context.

**Recommended Architecture**:

```
┌──────────────────────────────────────────────────────────────────┐
│                    Knowledge Base Vector Pipeline                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    INGESTION PIPELINE                        │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │                                                              │ │
│  │  KB Article Created/Updated                                  │ │
│  │       ↓                                                      │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │ Content Extractor│ ─→ Extract: Title, body, metadata     │ │
│  │  └────────┬─────────┘    Clean: HTML, formatting            │ │
│  │           ↓                                                  │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │ Semantic Chunker │ ─→ Strategy: Paragraph-based          │ │
│  │  └────────┬─────────┘    Strategy: Sentence window          │ │
│  │           ↓              Strategy: Recursive split          │ │
│  │  ┌──────────────────┐    Max chunk: ~500 tokens             │ │
│  │  │ Embedding Generator│ ─→ Model: text-embedding-3-small    │ │
│  │  └────────┬─────────┘     or: text-embedding-3-large        │ │
│  │           ↓                                                  │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │  Qdrant Upsert   │ ─→ Collection: homa_kb_{tenant_id}    │ │
│  │  └──────────────────┘    Payload: article_id, chunk_id,     │ │
│  │                                   title, url, category      │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    RETRIEVAL PIPELINE                        │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │                                                              │ │
│  │  User Query                                                  │ │
│  │       ↓                                                      │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │ Query Embedding  │ ─→ Same model as ingestion            │ │
│  │  └────────┬─────────┘                                       │ │
│  │           ↓                                                  │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │  Qdrant Search   │ ─→ Top-K: 5-10 chunks                 │ │
│  │  └────────┬─────────┘    Score threshold: 0.7               │ │
│  │           ↓              Filter: category, tags             │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │ Context Ranker   │ ─→ Re-rank by relevance               │ │
│  │  └────────┬─────────┘    Deduplicate by article             │ │
│  │           ↓                                                  │ │
│  │  ┌──────────────────┐                                       │ │
│  │  │ Context Formatter│ ─→ Format for LLM prompt              │ │
│  │  └──────────────────┘    Include: URL, title, excerpt       │ │
│  │                                                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

**Data Models Required**:

```go
// KBVectorIndex - Track vector indexing status
type KBVectorIndex struct {
    ArticleID       uint      `gorm:"primaryKey"`
    ChunkCount      int
    LastIndexedAt   time.Time
    IndexVersion    int       // For re-indexing on model change
    EmbeddingModel  string    // Track which model was used
    Status          string    // "pending", "indexed", "failed"
    ErrorMessage    *string
}

// Extend existing KnowledgeBaseChunk
type KnowledgeBaseChunk struct {
    ID              uint   `gorm:"primaryKey"`
    ArticleID       uint   `gorm:"index"`
    ChunkIndex      int    // Order within article
    Content         string
    TokenCount      int
    QdrantPointID   string // UUID from Qdrant
    ChunkingMethod  string // "paragraph", "sentence_window", "recursive"
}
```

**Chunking Strategy Recommendation**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Semantic Chunking Strategy                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Article Content                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ # Title                                                   │   │
│  │                                                           │   │
│  │ Introduction paragraph that sets context...               │   │
│  │                                                           │   │
│  │ ## Section 1                                              │   │
│  │ Content about topic A with details...                     │   │
│  │ More content continuing the thought...                    │   │
│  │                                                           │   │
│  │ ## Section 2                                              │   │
│  │ Different topic B with its own context...                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│       ↓                                                          │
│  Chunking Rules:                                                 │
│  1. Split on headers (##, ###) - preserve section boundaries     │
│  2. Keep paragraphs together when under 500 tokens               │
│  3. Add 2-sentence overlap between chunks for context            │
│  4. Include title + section header in each chunk metadata        │
│       ↓                                                          │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Chunk 1: [Title] + Introduction                            │ │
│  │ Metadata: {article_id, section: "intro", position: 0}      │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ Chunk 2: [Title] + [Section 1] + Content                   │ │
│  │ Metadata: {article_id, section: "Section 1", position: 1}  │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ Chunk 3: [Title] + [Section 2] + Content                   │ │
│  │ Metadata: {article_id, section: "Section 2", position: 2}  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Critique & Improvements**:

1. **Hybrid Search**: Combine vector search with BM25 keyword search for better results
2. **Multi-language Embeddings**: Use multilingual embedding model (e.g., `multilingual-e5-large`)
3. **Incremental Updates**: Only re-embed changed chunks, not entire articles
4. **Metadata Filtering**: Filter by category/tags before vector search
5. **Answer Highlighting**: Return specific sentences that answer the query

---

### 2.3 Feature: Automatic KB Sync with Qdrant

**Requirement**: On any KB change, update Qdrant vectors.

**Recommended Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB Sync Architecture                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Option A: Synchronous (Simple, immediate)                       │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  KB Article AfterCreate/AfterUpdate Hook                     ││
│  │       ↓                                                      ││
│  │  Queue job to background worker (don't block request)        ││
│  │       ↓                                                      ││
│  │  Worker: Chunk → Embed → Upsert Qdrant                       ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Option B: Event-Driven (Scalable, decoupled)                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  KB Article AfterCreate/AfterUpdate Hook                     ││
│  │       ↓                                                      ││
│  │  NATS Publish: "kb.article.updated" {article_id}             ││
│  │       ↓                                                      ││
│  │  KB Indexer Service (separate or same process)               ││
│  │  - Subscribe to "kb.article.*"                               ││
│  │  - Process: Chunk → Embed → Upsert Qdrant                    ││
│  │  - Update KBVectorIndex status                               ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Option C: Batch Processing (Cost-effective)                    │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  KB Article changes recorded in pending queue                ││
│  │       ↓                                                      ││
│  │  Cron job every 5 minutes                                    ││
│  │  - Batch all pending articles                                ││
│  │  - Process in single embedding API call                      ││
│  │  - Upsert all to Qdrant                                      ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  RECOMMENDED: Option B with Option C fallback                    │
│  - Real-time updates via NATS for immediate availability        │
│  - Batch job as fallback for missed events                      │
│  - Full re-index capability for embedding model upgrades        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Implementation Pattern**:

```go
// In apps/ai/kb_indexer.go

type KBIndexer struct {
    qdrant      *qdrant.Client
    embedder    *openai.Client
    batchSize   int
    workers     int
}

func (k *KBIndexer) ProcessArticle(articleID uint) error {
    // 1. Fetch article
    var article models.KnowledgeBaseArticle
    if err := db.First(&article, articleID).Error; err != nil {
        return err
    }

    // 2. Delete existing chunks from Qdrant
    k.qdrant.Delete(collectionName, qdrant.Filter{
        Must: []qdrant.Condition{{
            Field: "article_id",
            Match: qdrant.MatchValue(articleID),
        }},
    })

    // 3. Chunk content
    chunks := k.chunkContent(article.Content, ChunkConfig{
        MaxTokens:    500,
        OverlapSents: 2,
        PreserveHeaders: true,
    })

    // 4. Generate embeddings (batch)
    texts := make([]string, len(chunks))
    for i, c := range chunks {
        texts[i] = c.Text
    }
    embeddings, err := k.embedder.CreateEmbeddings(ctx, texts)

    // 5. Upsert to Qdrant
    points := make([]qdrant.Point, len(chunks))
    for i, chunk := range chunks {
        points[i] = qdrant.Point{
            ID:     uuid.New().String(),
            Vector: embeddings[i],
            Payload: map[string]interface{}{
                "article_id":  articleID,
                "chunk_index": i,
                "title":       article.Title,
                "url":         article.URL,
                "category_id": article.CategoryID,
                "content":     chunk.Text,
            },
        }
    }
    return k.qdrant.Upsert(collectionName, points)
}
```

**Critique & Improvements**:

1. **Idempotency**: Use deterministic chunk IDs (hash of article_id + chunk_index)
2. **Version Tracking**: Track embedding model version for bulk re-indexing
3. **Soft Delete**: Mark articles as deleted in Qdrant instead of hard delete
4. **Progress Tracking**: Show indexing progress in admin UI
5. **Error Recovery**: Retry failed indexing with exponential backoff

---

### 2.4 Feature: Multi-language Response Translation

**Requirement**: Always respond in user's language regardless of KB language.

**Recommended Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Translation Pipeline                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Strategy A: LLM-based Translation (Recommended)                │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  System Prompt includes:                                     ││
│  │  "Always respond in {client.language}. If the knowledge     ││
│  │   base content is in a different language, translate it     ││
│  │   accurately while preserving technical terms."              ││
│  │                                                              ││
│  │  Advantages:                                                 ││
│  │  - Single API call (no separate translation)                 ││
│  │  - Context-aware translation                                 ││
│  │  - Preserves tone and style                                  ││
│  │  - Handles technical terminology better                      ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Strategy B: Separate Translation API                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  AI Response (in KB language)                                ││
│  │       ↓                                                      ││
│  │  Detect: response.language != client.language                ││
│  │       ↓                                                      ││
│  │  Translate via:                                              ││
│  │  - DeepL API (highest quality)                               ││
│  │  - Google Translate API                                      ││
│  │  - Azure Translator                                          ││
│  │                                                              ││
│  │  Disadvantages:                                              ││
│  │  - Extra API cost                                            ││
│  │  - Extra latency                                             ││
│  │  - May lose context/tone                                     ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Strategy C: Hybrid (Recommended for complex cases)             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  1. LLM generates response with translation                  ││
│  │  2. If response.confidence < threshold:                      ││
│  │     - Use dedicated translation API                          ││
│  │  3. Cache translated KB chunks for common languages          ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Language Detection**:

```go
// Priority order for detecting user language:
// 1. Explicit client.language field (set during client creation)
// 2. Accept-Language header from HTTP request
// 3. Detect from message content using langdetect
// 4. Default to English

type LanguageDetector struct {
    detector *langdetect.Detector
}

func (l *LanguageDetector) Detect(client *models.Client, req *http.Request, message string) string {
    // 1. Client preference
    if client.Language != "" {
        return client.Language
    }

    // 2. Accept-Language header
    if lang := req.Header.Get("Accept-Language"); lang != "" {
        parsed := parseAcceptLanguage(lang)
        if len(parsed) > 0 {
            return parsed[0]
        }
    }

    // 3. Detect from message
    if detected, err := l.detector.Detect(message); err == nil {
        return detected
    }

    // 4. Default
    return "en"
}
```

**Critique & Improvements**:

1. **Language Memory**: Remember detected language for the conversation
2. **RTL Support**: Handle right-to-left languages properly
3. **Cultural Adaptation**: Not just translation, but cultural context
4. **Glossary**: Maintain technical term glossary per language
5. **Quality Check**: Validate translation quality periodically

---

### 2.5 Feature: Configurable AI Tone and Behavior

**Requirement**: Configure how AI responds (tone, style, guidelines).

**Recommended Configuration Model**:

```go
type AIPersonality struct {
    ID              uint   `gorm:"primaryKey"`
    Name            string // "Professional", "Friendly", "Technical"
    Description     string

    // Tone configuration
    Formality       string // "formal", "semi-formal", "casual"
    Empathy         string // "high", "medium", "low"
    Verbosity       string // "concise", "balanced", "detailed"

    // Behavior guidelines
    SystemPrompt    string // Base personality prompt
    DoInstructions  string // Things AI should do
    DontInstructions string // Things AI should avoid

    // Response formatting
    UseMarkdown     bool
    UseBulletPoints bool
    MaxResponseLen  int

    // Escalation behavior
    ApologizeOnError     bool
    OfferHumanHandover   bool
    HandoverPhrase       string // "Would you like to speak with a human agent?"

    // Knowledge base behavior
    AlwaysCiteSource     bool
    IncludeKBLinks       bool
    MaxKBChunks          int

    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**System Prompt Template**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    System Prompt Structure                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [BASE IDENTITY]                                                 │
│  You are {company_name}'s AI support assistant. Your name is    │
│  {bot_name}.                                                     │
│                                                                  │
│  [PERSONALITY]                                                   │
│  {personality.SystemPrompt}                                      │
│                                                                  │
│  [BEHAVIOR GUIDELINES]                                           │
│  DO:                                                             │
│  {personality.DoInstructions}                                    │
│                                                                  │
│  DON'T:                                                          │
│  {personality.DontInstructions}                                  │
│                                                                  │
│  [LANGUAGE]                                                      │
│  Always respond in {client.language}. If knowledge base         │
│  content is in a different language, translate accurately.       │
│                                                                  │
│  [KNOWLEDGE BASE CONTEXT]                                        │
│  Use the following information to help answer questions:         │
│  {kb_context}                                                    │
│                                                                  │
│  When citing sources, always include the article URL.           │
│                                                                  │
│  [ESCALATION]                                                    │
│  If you cannot help or the user asks to speak with a human,     │
│  respond with: {handover_phrase}                                 │
│                                                                  │
│  [AVAILABLE TOOLS]                                               │
│  You can call these functions when needed:                       │
│  {js_func_definitions}                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Critique & Improvements**:

1. **A/B Testing**: Support multiple personalities for testing
2. **Department Override**: Different personality per department
3. **Time-based**: Different tone for business hours vs after-hours
4. **Learning**: Track which responses get positive feedback
5. **Version Control**: Keep history of prompt changes

---

### 2.6 Feature: JS Func - JavaScript Tool Integration

**Requirement**: Install JavaScript tools that AI can call.

**CRITICAL ANALYSIS - This is the most complex feature**

```
┌─────────────────────────────────────────────────────────────────┐
│                    JS Func Architecture                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Challenge: Running untrusted JS securely in a Go backend        │
│                                                                  │
│  Option A: Embedded V8/QuickJS (goja)                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Pros:                                                       ││
│  │  - Fast execution                                            ││
│  │  - No network overhead                                       ││
│  │  - Full control over runtime                                 ││
│  │                                                              ││
│  │  Cons:                                                       ││
│  │  - Security sandbox is complex                               ││
│  │  - Memory limits hard to enforce                             ││
│  │  - No npm ecosystem access                                   ││
│  │                                                              ││
│  │  Libraries: github.com/dop251/goja (pure Go JS interpreter) ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Option B: Isolated Deno Runtime (RECOMMENDED)                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Pros:                                                       ││
│  │  - Built-in security sandbox                                 ││
│  │  - TypeScript support                                        ││
│  │  - Permission-based (--allow-net, --allow-read)             ││
│  │  - npm compatibility                                         ││
│  │                                                              ││
│  │  Cons:                                                       ││
│  │  - Subprocess overhead                                       ││
│  │  - Deno must be installed                                    ││
│  │                                                              ││
│  │  Execution: deno run --allow-net=api.example.com script.ts  ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Option C: Cloudflare Workers / AWS Lambda                      │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Pros:                                                       ││
│  │  - Complete isolation                                        ││
│  │  - Horizontally scaled automatically                         ││
│  │  - Built-in security                                         ││
│  │                                                              ││
│  │  Cons:                                                       ││
│  │  - Network latency                                           ││
│  │  - External dependency                                       ││
│  │  - Cost per execution                                        ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Option D: Docker Container Execution                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Pros:                                                       ││
│  │  - Full isolation                                            ││
│  │  - Any runtime (Node, Deno, Bun)                            ││
│  │  - Resource limits (CPU, memory, network)                    ││
│  │                                                              ││
│  │  Cons:                                                       ││
│  │  - Container startup overhead                                ││
│  │  - Complex orchestration                                     ││
│  │                                                              ││
│  │  Can use: Docker API or lightweight container (gVisor)       ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  RECOMMENDATION: Deno for development, Docker for production    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Data Models**:

```go
// JSFunc - JavaScript function definition
type JSFunc struct {
    ID              string `gorm:"primaryKey"` // UUID
    Name            string `gorm:"uniqueIndex"` // "get_order_status"
    DisplayName     string // "Get Order Status"
    Description     string // For AI to understand when to use
    Category        string // "orders", "payments", "shipping"

    // Code
    Code            string // JavaScript/TypeScript source
    Runtime         string // "deno", "node", "goja"

    // Input/Output Schema (JSON Schema format)
    InputSchema     string // JSON Schema for input validation
    OutputSchema    string // JSON Schema for output validation

    // OpenAI Function Calling format
    FunctionDef     string // JSON: {name, description, parameters}

    // Permissions (for Deno)
    AllowNet        string // Comma-separated domains
    AllowRead       string // Comma-separated paths
    AllowEnv        string // Comma-separated env vars

    // Execution limits
    TimeoutMs       int    // Max execution time
    MaxMemoryMB     int    // Max memory usage

    // State
    Enabled         bool
    Version         int
    LastModified    time.Time
    LastExecutedAt  *time.Time
    ExecutionCount  int64
    ErrorCount      int64

    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// JSFuncExecution - Audit log of executions
type JSFuncExecution struct {
    ID              uint   `gorm:"primaryKey"`
    FuncID          string `gorm:"index"`
    ConversationID  uint   `gorm:"index"`

    Input           string // JSON input
    Output          string // JSON output
    Error           *string

    DurationMs      int
    Success         bool

    CreatedAt       time.Time
}
```

**Input/Output Schema Definition**:

```json
{
  "name": "get_order_status",
  "description": "Get the current status of a customer order by order ID",
  "inputSchema": {
    "type": "object",
    "properties": {
      "order_id": {
        "type": "string",
        "description": "The order ID to look up",
        "pattern": "^ORD-[0-9]{6}$"
      }
    },
    "required": ["order_id"]
  },
  "outputSchema": {
    "type": "object",
    "properties": {
      "status": {
        "type": "string",
        "enum": ["pending", "processing", "shipped", "delivered"]
      },
      "tracking_number": {
        "type": "string"
      },
      "estimated_delivery": {
        "type": "string",
        "format": "date"
      }
    }
  }
}
```

**Execution Flow**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    JS Func Execution Flow                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  AI decides to call function                                     │
│       ↓                                                          │
│  ┌──────────────────┐                                           │
│  │ Input Validation │ ─→ Validate against inputSchema           │
│  └────────┬─────────┘    Return error if invalid                │
│           ↓                                                      │
│  ┌──────────────────┐                                           │
│  │ Permission Check │ ─→ Is func enabled?                       │
│  └────────┬─────────┘    Is user allowed to trigger?            │
│           ↓                                                      │
│  ┌──────────────────┐                                           │
│  │ Context Injection│ ─→ Add: conversation_id, client_id        │
│  └────────┬─────────┘    Add: authenticated secrets             │
│           ↓                                                      │
│  ┌──────────────────┐                                           │
│  │ Runtime Executor │ ─→ Spawn Deno/Docker with limits          │
│  └────────┬─────────┘    Pass input as JSON                     │
│           ↓              Capture stdout                         │
│  ┌──────────────────┐                                           │
│  │ Output Validation│ ─→ Validate against outputSchema          │
│  └────────┬─────────┘    Parse JSON response                    │
│           ↓                                                      │
│  ┌──────────────────┐                                           │
│  │  Audit Logging   │ ─→ Log execution to JSFuncExecution       │
│  └────────┬─────────┘                                           │
│           ↓                                                      │
│  Return to AI for response generation                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Critique & Improvements**:

1. **Secret Management**: JS Funcs need access to API keys - use encrypted vault
2. **Rate Limiting**: Per-function rate limits to prevent abuse
3. **Caching**: Cache idempotent function results
4. **Retry Logic**: Configurable retry for transient failures
5. **Dependency Management**: Allow importing npm packages safely
6. **Testing Environment**: Sandbox for testing before deployment

---

### 2.7 Feature: Visual Workflow Designer

**Requirement**: Define workflows visually - if X then Y, call JS Func, etc.

**Recommended Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Workflow Engine Architecture                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Workflow Definition (JSON/YAML stored in DB)                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  {                                                           ││
│  │    "id": "order_inquiry_flow",                               ││
│  │    "name": "Order Inquiry Workflow",                         ││
│  │    "trigger": {                                              ││
│  │      "type": "intent",                                       ││
│  │      "conditions": ["order_status", "where_is_my_order"]     ││
│  │    },                                                        ││
│  │    "nodes": [                                                ││
│  │      {                                                       ││
│  │        "id": "ask_order_id",                                 ││
│  │        "type": "prompt",                                     ││
│  │        "message": "What is your order number?",              ││
│  │        "variable": "order_id",                               ││
│  │        "validation": "^ORD-[0-9]{6}$",                       ││
│  │        "next": "lookup_order"                                ││
│  │      },                                                      ││
│  │      {                                                       ││
│  │        "id": "lookup_order",                                 ││
│  │        "type": "js_func",                                    ││
│  │        "function": "get_order_status",                       ││
│  │        "input": {"order_id": "{{order_id}}"},               ││
│  │        "next": "check_status"                                ││
│  │      },                                                      ││
│  │      {                                                       ││
│  │        "id": "check_status",                                 ││
│  │        "type": "condition",                                  ││
│  │        "conditions": [                                       ││
│  │          {"if": "{{result.status}} == 'shipped'",           ││
│  │           "next": "show_tracking"},                          ││
│  │          {"if": "{{result.status}} == 'pending'",           ││
│  │           "next": "explain_pending"},                        ││
│  │          {"else": "next": "ai_response"}                     ││
│  │        ]                                                     ││
│  │      },                                                      ││
│  │      {                                                       ││
│  │        "id": "show_tracking",                                ││
│  │        "type": "message",                                    ││
│  │        "template": "Your order has shipped! Track: {{..}}",  ││
│  │        "next": "end"                                         ││
│  │      },                                                      ││
│  │      {                                                       ││
│  │        "id": "ai_response",                                  ││
│  │        "type": "ai",                                         ││
│  │        "context": "{{result}}",                              ││
│  │        "prompt": "Explain order status to customer",         ││
│  │        "next": "end"                                         ││
│  │      }                                                       ││
│  │    ]                                                         ││
│  │  }                                                           ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Visual Editor (Frontend Component)                             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐                  ││
│  │  │ Trigger │───▶│  Prompt │───▶│ JS Func │                  ││
│  │  │ (Intent)│    │(Ask ID) │    │(Lookup) │                  ││
│  │  └─────────┘    └─────────┘    └────┬────┘                  ││
│  │                                      │                       ││
│  │                            ┌─────────▼─────────┐            ││
│  │                            │    Condition      │            ││
│  │                            │  (Check Status)   │            ││
│  │                            └─────────┬─────────┘            ││
│  │                    ┌─────────────────┼─────────────────┐    ││
│  │                    ▼                 ▼                 ▼    ││
│  │              ┌─────────┐       ┌─────────┐       ┌─────────┐││
│  │              │ Message │       │ Message │       │   AI    │││
│  │              │(Shipped)│       │(Pending)│       │Response │││
│  │              └─────────┘       └─────────┘       └─────────┘││
│  │                                                              ││
│  │  Libraries: React Flow, Rete.js, or custom SVG-based        ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Node Types**:

```go
const (
    NodeTypeTrigger   = "trigger"   // Entry point (intent, keyword, etc.)
    NodeTypePrompt    = "prompt"    // Ask user for input
    NodeTypeMessage   = "message"   // Send static/template message
    NodeTypeCondition = "condition" // Branch based on conditions
    NodeTypeJSFunc    = "js_func"   // Execute JS function
    NodeTypeAI        = "ai"        // Get AI response with context
    NodeTypeHandover  = "handover"  // Transfer to human agent
    NodeTypeSetVar    = "set_var"   // Set conversation variable
    NodeTypeWait      = "wait"      // Wait for user response
    NodeTypeEnd       = "end"       // End workflow
)
```

**Data Models**:

```go
// Workflow - Visual workflow definition
type Workflow struct {
    ID              string `gorm:"primaryKey"` // UUID
    Name            string
    Description     string

    // Trigger configuration
    TriggerType     string // "intent", "keyword", "always", "manual"
    TriggerConfig   string // JSON configuration

    // Flow definition
    Definition      string // JSON workflow definition

    // Assignment
    DepartmentID    *uint  // null = all departments
    Priority        int    // Higher priority workflows checked first

    // State
    Enabled         bool
    Version         int

    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// WorkflowExecution - Track workflow state per conversation
type WorkflowExecution struct {
    ID              uint   `gorm:"primaryKey"`
    WorkflowID      string `gorm:"index"`
    ConversationID  uint   `gorm:"index"`

    CurrentNodeID   string
    Variables       string // JSON: collected variables
    History         string // JSON: node execution history

    Status          string // "active", "completed", "abandoned", "handed_over"
    StartedAt       time.Time
    CompletedAt     *time.Time
}
```

**Critique & Improvements**:

1. **Workflow Versioning**: Don't break active conversations when workflow changes
2. **Debugging**: Visualize execution path for troubleshooting
3. **Timeout Handling**: What happens if user doesn't respond to prompt?
4. **Parallel Branches**: Support parallel execution paths
5. **Subflows**: Allow workflows to call other workflows
6. **Templates**: Pre-built workflow templates for common scenarios

---

### 2.8 Feature: Horizontal Scaling - Shared State

**Requirement**: JS Funcs and workflows shared across all backend instances.

**Architecture Options**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Horizontal Scaling Architecture               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Challenge: Multiple backend instances need shared state         │
│                                                                  │
│  Already Shared (via MySQL):                                    │
│  ✓ Workflows (definition in database)                           │
│  ✓ JS Funcs (code in database)                                  │
│  ✓ AI Configuration                                             │
│  ✓ Conversations and Messages                                   │
│                                                                  │
│  Needs Synchronization:                                         │
│  • Cache invalidation when JS Func changes                      │
│  • Active workflow state (which node is user at?)               │
│  • Runtime JS execution (where does it run?)                    │
│                                                                  │
│  Solution Architecture:                                         │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │   ┌──────────┐ ┌──────────┐ ┌──────────┐                   ││
│  │   │Backend 1 │ │Backend 2 │ │Backend 3 │                   ││
│  │   └────┬─────┘ └────┬─────┘ └────┬─────┘                   ││
│  │        │            │            │                          ││
│  │        └────────────┼────────────┘                          ││
│  │                     │                                        ││
│  │              ┌──────▼──────┐                                 ││
│  │              │    NATS     │  ← Pub/Sub for events          ││
│  │              └──────┬──────┘                                 ││
│  │                     │                                        ││
│  │  ┌──────────────────┼──────────────────┐                    ││
│  │  │                  │                  │                     ││
│  │  ▼                  ▼                  ▼                     ││
│  │ MySQL            Redis              Qdrant                   ││
│  │ (Source of       (Cache +           (Vector                  ││
│  │  Truth)          Session)            Search)                 ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Event Flow for JS Func Update:                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  Admin updates JS Func via Backend 1                         ││
│  │       ↓                                                      ││
│  │  Backend 1: Update MySQL                                     ││
│  │       ↓                                                      ││
│  │  Backend 1: Publish NATS "jsfunc.updated" {func_id}         ││
│  │       ↓                                                      ││
│  │  All Backends: Receive event, invalidate local cache         ││
│  │       ↓                                                      ││
│  │  Next execution: Reload from MySQL                           ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Workflow State Management:                                     │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  Option A: Database-based (Simple, consistent)               ││
│  │  - Store WorkflowExecution in MySQL                          ││
│  │  - Any backend can continue execution                        ││
│  │  - Slightly higher latency                                   ││
│  │                                                              ││
│  │  Option B: Redis-based (Fast, requires careful handling)     ││
│  │  - Store active workflow state in Redis                      ││
│  │  - Use conversation_id as key                                ││
│  │  - Persist to MySQL on completion                            ││
│  │                                                              ││
│  │  RECOMMENDATION: Option A for reliability                    ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Cache Invalidation Pattern**:

```go
// In apps/ai/cache.go

type AICache struct {
    local    *sync.Map          // Local in-memory cache
    nats     *nats.Conn
    subjects []string
}

func (c *AICache) Init() {
    // Subscribe to invalidation events
    c.nats.Subscribe("jsfunc.updated", func(msg *nats.Msg) {
        funcID := string(msg.Data)
        c.local.Delete("jsfunc:" + funcID)
    })

    c.nats.Subscribe("workflow.updated", func(msg *nats.Msg) {
        workflowID := string(msg.Data)
        c.local.Delete("workflow:" + workflowID)
    })

    c.nats.Subscribe("ai_config.updated", func(msg *nats.Msg) {
        deptID := string(msg.Data)
        c.local.Delete("ai_config:" + deptID)
    })
}

// Called after updating JS Func in database
func (c *AICache) InvalidateJSFunc(funcID string) {
    c.local.Delete("jsfunc:" + funcID)
    c.nats.Publish("jsfunc.updated", []byte(funcID))
}
```

**Critique & Improvements**:

1. **Sticky Sessions**: Consider routing same conversation to same backend
2. **Graceful Degradation**: Handle NATS outage gracefully
3. **Cache Warming**: Pre-load frequently used JS Funcs on startup
4. **Distributed Locks**: Prevent concurrent workflow state updates
5. **Health Checks**: Monitor cache sync lag across instances

---

### 2.9 Feature: Human Handover System

**Requirement**: Handover to human when AI fails, user requests, or user is frustrated.

**Detection Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Handover Detection System                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Trigger Categories:                                            │
│                                                                  │
│  1. Explicit Request Detection                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Keywords/Phrases (configurable):                            ││
│  │  - "speak to a human"                                        ││
│  │  - "talk to agent"                                           ││
│  │  - "real person"                                             ││
│  │  - "transfer me"                                             ││
│  │  - "this isn't helping"                                      ││
│  │  - "I need help from a person"                               ││
│  │                                                              ││
│  │  Implementation: Regex + intent classification               ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  2. Frustration Detection (Sentiment Analysis)                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Signals:                                                    ││
│  │  - Negative sentiment score (via LLM or dedicated model)    ││
│  │  - Profanity detection                                       ││
│  │  - ALL CAPS messages                                         ││
│  │  - Repeated exclamation marks (!!!)                         ││
│  │  - Short, terse responses after long exchanges              ││
│  │                                                              ││
│  │  Scoring: Cumulative frustration score per conversation      ││
│  │  Threshold: Configurable (e.g., score > 0.7 = handover)     ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  3. AI Failure Detection                                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Conditions:                                                 ││
│  │  - AI confidence score < threshold                           ││
│  │  - No relevant KB articles found                             ││
│  │  - Same question asked 3+ times                              ││
│  │  - AI responds with "I don't know" patterns                  ││
│  │  - Workflow reaches dead end                                 ││
│  │  - JS Func fails repeatedly                                  ││
│  │                                                              ││
│  │  Meta-detection: Ask AI "Are you able to help with this?"   ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  4. Complexity Detection                                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Indicators:                                                 ││
│  │  - Multi-topic conversation                                  ││
│  │  - Legal/compliance questions                                ││
│  │  - Financial disputes                                        ││
│  │  - Technical issues beyond KB scope                          ││
│  │  - Mentions of other channels (phone, email, previous chat) ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Handover Flow**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Handover Execution Flow                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Handover Triggered                                              │
│       ↓                                                          │
│  ┌──────────────────┐                                           │
│  │ Confirm Handover │ ─→ "I'll connect you with a human agent. │
│  └────────┬─────────┘    Is that okay?"                         │
│           ↓              (Skip if frustration > high threshold) │
│  ┌──────────────────┐                                           │
│  │ Generate Summary │ ─→ AI creates conversation summary        │
│  └────────┬─────────┘    - Key issues discussed                 │
│           ↓              - What was tried                       │
│  ┌──────────────────┐    - Customer sentiment                   │
│  │ Select Department│ ─→ Based on:                              │
│  └────────┬─────────┘    - Topic classification                 │
│           ↓              - Current department config            │
│  ┌──────────────────┐    - Agent availability                   │
│  │ Update Status    │ ─→ conversation.status = "wait_for_agent" │
│  └────────┬─────────┘    conversation.mode = "human"            │
│           ↓                                                      │
│  ┌──────────────────┐                                           │
│  │ Notify System    │ ─→ NATS: "conversation.handover"          │
│  └────────┬─────────┘    Webhook: handover event                │
│           ↓              Push notification to agents            │
│  ┌──────────────────┐                                           │
│  │ User Message     │ ─→ "An agent will be with you shortly.   │
│  └──────────────────┘    Average wait time: X minutes"          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Data Models**:

```go
// HandoverConfig - Configuration for handover triggers
type HandoverConfig struct {
    ID              uint   `gorm:"primaryKey"`
    DepartmentID    *uint  `gorm:"index"` // null = global

    // Explicit triggers
    HandoverKeywords    string // JSON array of phrases

    // Frustration detection
    EnableSentiment     bool
    SentimentThreshold  float32 // 0-1, higher = more negative

    // AI failure
    MaxAIAttempts       int    // Handover after N failed attempts
    ConfidenceThreshold float32

    // Timing
    MaxAIConversationMins int  // Auto-handover after X minutes

    // Behavior
    RequireConfirmation bool   // Ask user before handover
    IncludeSummary      bool   // Generate AI summary for agent

    CreatedAt           time.Time
    UpdatedAt           time.Time
}

// HandoverEvent - Audit log of handovers
type HandoverEvent struct {
    ID              uint   `gorm:"primaryKey"`
    ConversationID  uint   `gorm:"index"`

    TriggerType     string // "explicit", "frustration", "ai_failure", "complexity"
    TriggerDetails  string // JSON: what triggered handover

    AISummary       string // AI-generated summary
    SentimentScore  float32

    FromDepartmentID *uint
    ToDepartmentID   uint
    AssignedUserID   *string // Agent assigned (if auto-assigned)

    CreatedAt       time.Time
}
```

**Critique & Improvements**:

1. **Warm Handover**: AI stays in conversation to assist agent
2. **Queue Position**: Show user their position in queue
3. **Callback Option**: Offer to call back when agent available
4. **Business Hours**: Different behavior outside business hours
5. **VIP Detection**: Priority handover for important customers
6. **Skill-based Routing**: Match handover to agent skills

---

## 3. Additional Feature Suggestions

Based on my analysis, here are additional features to consider:

### 3.1 Conversation Analytics & Insights

```
┌─────────────────────────────────────────────────────────────────┐
│  • AI resolution rate                                            │
│  • Average handling time (AI vs human)                          │
│  • Common unresolved topics                                      │
│  • KB article effectiveness (which articles resolve issues)     │
│  • JS Func usage statistics                                      │
│  • Handover reasons breakdown                                    │
│  • Customer satisfaction correlation                             │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Proactive AI Engagement

```
┌─────────────────────────────────────────────────────────────────┐
│  • Trigger AI based on user behavior (page visit, idle time)    │
│  • Suggest relevant KB articles based on context                │
│  • Pre-emptive issue detection (order delay, payment failure)   │
│  • Follow-up after resolved conversations                       │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 AI Learning & Improvement

```
┌─────────────────────────────────────────────────────────────────┐
│  • Feedback loop: Agent corrections train AI                    │
│  • Automatic KB gap detection (questions without answers)       │
│  • Response quality scoring                                      │
│  • A/B testing for prompts and workflows                        │
└─────────────────────────────────────────────────────────────────┘
```

### 3.4 Multi-Modal Support

```
┌─────────────────────────────────────────────────────────────────┐
│  • Image understanding (receipt photos, screenshots)            │
│  • Voice message transcription                                   │
│  • File attachment handling                                      │
│  • Screen sharing assistance                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 3.5 Agent Assist Mode

```
┌─────────────────────────────────────────────────────────────────┐
│  • AI suggests responses for human agents                       │
│  • Auto-populate customer context                               │
│  • Real-time translation for agents                             │
│  • Canned response suggestions based on context                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Critical Analysis & Recommendations

### 4.1 Architecture Concerns

| Concern | Risk | Mitigation |
|---------|------|------------|
| **JS Func Security** | Code injection, resource abuse | Use Deno sandbox with strict permissions |
| **AI Costs** | OpenAI API costs can explode | Implement token budgets per department |
| **Latency** | Multiple API calls (embed, search, LLM) | Parallel execution, caching, streaming |
| **Data Privacy** | Customer data sent to OpenAI | Azure OpenAI in your region, PII filtering |
| **Single Points of Failure** | Qdrant down = no KB search | Fallback to keyword search |
| **Workflow Complexity** | Users create infinite loops | Validation, execution limits |

### 4.2 Implementation Priority

**Phase 1 - Foundation (Weeks 1-4)**
1. AI Configuration model and admin UI
2. OpenAI integration with basic response
3. System prompt configuration

**Phase 2 - Knowledge Base (Weeks 5-8)**
4. Qdrant integration
5. KB sync pipeline
6. RAG implementation

**Phase 3 - Advanced Features (Weeks 9-12)**
7. JS Func runtime (Deno)
8. Basic workflow engine
9. Handover system

**Phase 4 - Polish (Weeks 13-16)**
10. Visual workflow editor
11. Analytics dashboard
12. Performance optimization

### 4.3 Technology Choices

| Component | Recommendation | Alternatives |
|-----------|---------------|--------------|
| **Vector DB** | Qdrant | Pinecone, Weaviate, Milvus |
| **LLM Provider** | OpenAI GPT-4 | Anthropic Claude, Azure OpenAI |
| **Embeddings** | text-embedding-3-small | Cohere, local models |
| **JS Runtime** | Deno | goja (embedded), Docker |
| **Workflow Storage** | JSON in MySQL | Temporal.io, separate workflow engine |
| **Translation** | GPT-4 (inline) | DeepL, Google Translate |

### 4.4 Potential Issues

1. **Cold Start**: First Qdrant query after idle period may be slow
2. **Context Window**: Very long conversations may exceed GPT-4 limits
3. **Hallucination**: AI may generate false information without KB match
4. **Rate Limits**: OpenAI rate limits during traffic spikes
5. **Workflow State**: User closes browser mid-workflow - how to resume?

---

## 5. Confirmed Requirements

Based on stakeholder feedback:

| Requirement | Decision |
|-------------|----------|
| **Multi-tenant** | NO - Single tenant system |
| **Data residency/GDPR** | NO - No compliance requirements |
| **Cost budget** | YES - Handover to human when budget reached |
| **Languages** | AI-based translation (KB in Italian → respond in Persian) |
| **Timeout** | YES - Configurable timeout, handover if AI doesn't respond |

---

## 6. Budget & Timeout System (CRITICAL FEATURES)

### 6.1 Token/Cost Budget Tracking

**Requirement**: Track AI costs per conversation and handover to human when budget exceeded.

**Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI Cost Budget System                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Cost Tracking Flow:                                            │
│                                                                  │
│  AI Request                                                      │
│       ↓                                                          │
│  ┌──────────────────┐                                           │
│  │ Budget Check     │ ─→ Load AIConversationUsage               │
│  └────────┬─────────┘    Compare: current_cost vs max_budget    │
│           │                                                      │
│           ├─→ [Budget OK] ─→ Continue to AI                     │
│           │                                                      │
│           └─→ [Budget Exceeded] ─→ Trigger Handover             │
│                                    Message: "Let me connect     │
│                                    you with a specialist..."    │
│                                                                  │
│  After AI Response:                                             │
│       ↓                                                          │
│  ┌──────────────────┐                                           │
│  │ Usage Recording  │ ─→ Record: prompt_tokens, completion_tokens│
│  └────────┬─────────┘    Calculate: cost based on model rates   │
│           ↓              Update: AIConversationUsage            │
│  ┌──────────────────┐                                           │
│  │ Budget Warning   │ ─→ If usage > 80% budget:                 │
│  └──────────────────┘    - Log warning                          │
│                          - Consider shorter responses            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Data Models**:

```go
// AIBudgetConfig - Global and per-department budget settings
type AIBudgetConfig struct {
    ID                  uint    `gorm:"primaryKey"`
    DepartmentID        *uint   `gorm:"index"` // null = global default

    // Budget limits
    MaxTokensPerConversation   int     // Max tokens per conversation (0 = unlimited)
    MaxCostPerConversation     float64 // Max USD cost per conversation (0 = unlimited)
    MaxTokensPerDay            int     // Daily token limit across all conversations
    MaxCostPerDay              float64 // Daily cost limit

    // Model pricing (USD per 1K tokens) - updated as OpenAI changes prices
    GPT4InputPrice             float64 // e.g., 0.03
    GPT4OutputPrice            float64 // e.g., 0.06
    GPT35InputPrice            float64 // e.g., 0.0015
    GPT35OutputPrice           float64 // e.g., 0.002
    EmbeddingPrice             float64 // e.g., 0.0001

    // Behavior when budget exceeded
    HandoverOnBudgetExceeded   bool    // true = handover, false = stop AI
    BudgetExceededMessage      string  // Message to user when budget hit

    // Warnings
    WarningThresholdPercent    int     // Warn at this % of budget (e.g., 80)

    CreatedAt           time.Time
    UpdatedAt           time.Time
}

// AIConversationUsage - Track usage per conversation
type AIConversationUsage struct {
    ConversationID      uint      `gorm:"primaryKey"`

    // Token counts
    TotalPromptTokens   int
    TotalCompletionTokens int
    TotalEmbeddingTokens int

    // Cost tracking (USD)
    TotalCost           float64

    // Request counts
    AIRequestCount      int
    KBSearchCount       int
    JSFuncCallCount     int

    // Timing
    TotalAILatencyMs    int64     // Cumulative AI response time
    FirstRequestAt      time.Time
    LastRequestAt       time.Time

    // State
    BudgetExceeded      bool
    BudgetExceededAt    *time.Time

    UpdatedAt           time.Time
}

// AIDailyUsage - Track daily usage for global limits
type AIDailyUsage struct {
    Date                time.Time `gorm:"primaryKey"` // Date only, no time
    DepartmentID        *uint     `gorm:"primaryKey"` // null = global

    TotalTokens         int
    TotalCost           float64
    TotalConversations  int
    TotalHandovers      int       // Handovers due to budget

    UpdatedAt           time.Time
}
```

**Implementation**:

```go
// In apps/ai/budget.go

type BudgetManager struct {
    config *AIBudgetConfig
}

// CheckBudget returns (canProceed, remainingTokens, remainingCost)
func (b *BudgetManager) CheckBudget(conversationID uint) (bool, int, float64, error) {
    var usage AIConversationUsage
    if err := db.FirstOrCreate(&usage, AIConversationUsage{
        ConversationID: conversationID,
    }).Error; err != nil {
        return false, 0, 0, err
    }

    // Check if already exceeded
    if usage.BudgetExceeded {
        return false, 0, 0, nil
    }

    // Calculate remaining budget
    config := b.getConfig(conversationID)

    remainingTokens := config.MaxTokensPerConversation -
                       (usage.TotalPromptTokens + usage.TotalCompletionTokens)
    remainingCost := config.MaxCostPerConversation - usage.TotalCost

    // Check limits
    if config.MaxTokensPerConversation > 0 && remainingTokens <= 0 {
        b.markBudgetExceeded(conversationID, "token_limit")
        return false, 0, 0, nil
    }

    if config.MaxCostPerConversation > 0 && remainingCost <= 0 {
        b.markBudgetExceeded(conversationID, "cost_limit")
        return false, 0, 0, nil
    }

    return true, remainingTokens, remainingCost, nil
}

// RecordUsage records token usage after AI response
func (b *BudgetManager) RecordUsage(conversationID uint, usage OpenAIUsage) error {
    config := b.getConfig(conversationID)

    // Calculate cost
    inputCost := float64(usage.PromptTokens) / 1000 * config.GPT4InputPrice
    outputCost := float64(usage.CompletionTokens) / 1000 * config.GPT4OutputPrice
    totalCost := inputCost + outputCost

    // Update conversation usage
    return db.Model(&AIConversationUsage{}).
        Where("conversation_id = ?", conversationID).
        Updates(map[string]interface{}{
            "total_prompt_tokens":     gorm.Expr("total_prompt_tokens + ?", usage.PromptTokens),
            "total_completion_tokens": gorm.Expr("total_completion_tokens + ?", usage.CompletionTokens),
            "total_cost":              gorm.Expr("total_cost + ?", totalCost),
            "ai_request_count":        gorm.Expr("ai_request_count + 1"),
            "last_request_at":         time.Now(),
        }).Error
}

func (b *BudgetManager) markBudgetExceeded(conversationID uint, reason string) {
    now := time.Now()
    db.Model(&AIConversationUsage{}).
        Where("conversation_id = ?", conversationID).
        Updates(map[string]interface{}{
            "budget_exceeded":    true,
            "budget_exceeded_at": &now,
        })

    // Trigger handover
    TriggerHandover(conversationID, "budget_exceeded", reason)
}
```

**Budget Check Integration Point**:

```go
// In apps/ai/responder.go

func (r *AIResponder) HandleMessage(ctx context.Context, msg *Message) error {
    // 1. Check budget FIRST
    canProceed, remainingTokens, remainingCost, err := r.budget.CheckBudget(msg.ConversationID)
    if err != nil {
        return err
    }

    if !canProceed {
        // Budget exceeded - handover already triggered
        return r.sendBudgetExceededMessage(msg.ConversationID)
    }

    // 2. Optional: Adjust max_tokens based on remaining budget
    maxTokens := min(r.config.MaxTokens, remainingTokens)

    // 3. Proceed with AI call
    response, usage, err := r.openai.CreateChatCompletion(ctx, OpenAIRequest{
        Model:     r.config.Model,
        Messages:  messages,
        MaxTokens: maxTokens,
        // ...
    })

    // 4. Record usage
    r.budget.RecordUsage(msg.ConversationID, usage)

    // 5. Check if this response pushed us over budget (for next message)
    // This is handled in next CheckBudget call

    return r.sendResponse(msg.ConversationID, response)
}
```

---

### 6.2 AI Response Timeout with Handover

**Requirement**: If AI doesn't respond within X minutes (configurable), handover to human.

**Architecture**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI Timeout System                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Timeout Flow:                                                  │
│                                                                  │
│  Client Message Received                                         │
│       ↓                                                          │
│  ┌──────────────────┐                                           │
│  │ Start Timer      │ ─→ Context with timeout from config       │
│  └────────┬─────────┘    (e.g., 2 minutes)                      │
│           │                                                      │
│           ├─→ [AI Responds in Time] ─→ Cancel timer, send response│
│           │                                                      │
│           └─→ [Timeout Reached] ─→ Cancel AI request            │
│                                    Trigger handover              │
│                                    Message: "I'm having trouble  │
│                                    responding. Let me connect    │
│                                    you with a human agent..."    │
│                                                                  │
│  Configuration:                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  AITimeoutConfig:                                            ││
│  │  - response_timeout_seconds: 120  (2 minutes)                ││
│  │  - openai_request_timeout_seconds: 60                        ││
│  │  - qdrant_search_timeout_seconds: 10                         ││
│  │  - jsfunc_execution_timeout_seconds: 30                      ││
│  │  - handover_on_timeout: true                                 ││
│  │  - timeout_message: "I apologize for the delay..."          ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Data Models**:

```go
// AITimeoutConfig - Timeout settings (add to AIConfiguration)
type AITimeoutConfig struct {
    ID                          uint  `gorm:"primaryKey"`
    DepartmentID                *uint `gorm:"index"`

    // Overall response timeout
    ResponseTimeoutSeconds      int   // Total time AI has to respond (default: 120)

    // Component timeouts
    OpenAIRequestTimeoutSeconds int   // OpenAI API call timeout (default: 60)
    QdrantSearchTimeoutSeconds  int   // Vector search timeout (default: 10)
    JSFuncExecutionTimeoutSeconds int // JS function timeout (default: 30)
    EmbeddingTimeoutSeconds     int   // Embedding generation timeout (default: 15)

    // Behavior on timeout
    HandoverOnTimeout           bool   // true = handover, false = retry or error
    MaxRetries                  int    // Retry count before handover (default: 1)
    TimeoutMessage              string // Message to user on timeout

    CreatedAt                   time.Time
    UpdatedAt                   time.Time
}
```

**Implementation**:

```go
// In apps/ai/timeout.go

type TimeoutManager struct {
    config *AITimeoutConfig
}

// ExecuteWithTimeout wraps AI response generation with timeout
func (t *TimeoutManager) ExecuteWithTimeout(
    ctx context.Context,
    conversationID uint,
    fn func(ctx context.Context) (*AIResponse, error),
) (*AIResponse, error) {

    config := t.getConfig(conversationID)
    timeout := time.Duration(config.ResponseTimeoutSeconds) * time.Second

    // Create timeout context
    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Channel for result
    resultCh := make(chan struct {
        response *AIResponse
        err      error
    }, 1)

    // Execute AI function in goroutine
    go func() {
        response, err := fn(timeoutCtx)
        resultCh <- struct {
            response *AIResponse
            err      error
        }{response, err}
    }()

    // Wait for result or timeout
    select {
    case result := <-resultCh:
        return result.response, result.err

    case <-timeoutCtx.Done():
        // Timeout reached
        t.handleTimeout(conversationID)
        return nil, ErrAITimeout
    }
}

func (t *TimeoutManager) handleTimeout(conversationID uint) {
    config := t.getConfig(conversationID)

    // Log timeout event
    LogAIEvent(conversationID, "timeout", map[string]interface{}{
        "timeout_seconds": config.ResponseTimeoutSeconds,
    })

    if config.HandoverOnTimeout {
        // Trigger handover
        TriggerHandover(conversationID, "timeout", "AI response timeout")

        // Send timeout message to user
        SendSystemMessage(conversationID, config.TimeoutMessage)
    }
}
```

**Integration with AI Responder**:

```go
// In apps/ai/responder.go

func (r *AIResponder) HandleMessage(ctx context.Context, msg *Message) error {
    // Wrap entire AI response in timeout
    response, err := r.timeout.ExecuteWithTimeout(ctx, msg.ConversationID,
        func(timeoutCtx context.Context) (*AIResponse, error) {
            // 1. Check budget
            canProceed, _, _, err := r.budget.CheckBudget(msg.ConversationID)
            if err != nil || !canProceed {
                return nil, ErrBudgetExceeded
            }

            // 2. Search KB (with component timeout)
            kbCtx, kbCancel := context.WithTimeout(timeoutCtx,
                time.Duration(r.config.QdrantSearchTimeoutSeconds)*time.Second)
            defer kbCancel()
            kbContext, err := r.searchKB(kbCtx, msg.Body)

            // 3. Build prompt
            prompt := r.buildPrompt(msg, kbContext)

            // 4. Call OpenAI (with component timeout)
            aiCtx, aiCancel := context.WithTimeout(timeoutCtx,
                time.Duration(r.config.OpenAIRequestTimeoutSeconds)*time.Second)
            defer aiCancel()
            response, usage, err := r.openai.CreateChatCompletion(aiCtx, prompt)
            if err != nil {
                return nil, err
            }

            // 5. Record usage
            r.budget.RecordUsage(msg.ConversationID, usage)

            return &AIResponse{
                Content: response,
                Usage:   usage,
            }, nil
        },
    )

    if err == ErrAITimeout {
        // Timeout handled by TimeoutManager
        return nil
    }
    if err == ErrBudgetExceeded {
        return r.sendBudgetExceededMessage(msg.ConversationID)
    }
    if err != nil {
        return r.handleError(msg.ConversationID, err)
    }

    return r.sendResponse(msg.ConversationID, response)
}
```

---

### 6.3 Language Detection and Translation

**Requirement**: KB may be in Italian, but if user asks in Persian, respond in Persian using AI translation.

**Flow**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Language Translation Flow                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Step 1: Detect User Language                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Priority:                                                   ││
│  │  1. client.language field (if set explicitly)                ││
│  │  2. Detect from current message content                      ││
│  │  3. Detect from conversation history                         ││
│  │  4. Accept-Language header                                   ││
│  │  5. Default: English                                         ││
│  │                                                              ││
│  │  Detection: Use langdetect library or ask GPT                ││
│  │  Store: Save detected language in conversation state         ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Step 2: Search KB (in original language)                       │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  - Use multilingual embeddings (text-embedding-3-small       ││
│  │    supports 100+ languages)                                  ││
│  │  - Query embedding in user's language                        ││
│  │  - KB articles stored in original language (Italian)         ││
│  │  - Vector similarity works cross-language!                   ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Step 3: Generate Response with Translation                     │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │  System Prompt (in user's language detection):               ││
│  │  """                                                         ││
│  │  The user is communicating in {detected_language}.           ││
│  │  You MUST respond ONLY in {detected_language}.               ││
│  │                                                              ││
│  │  The knowledge base content below may be in a different      ││
│  │  language. Translate all relevant information to             ││
│  │  {detected_language} when responding.                        ││
│  │                                                              ││
│  │  Important:                                                  ││
│  │  - Preserve technical terms that don't translate well        ││
│  │  - Keep URLs and links unchanged                             ││
│  │  - Maintain the same helpful tone                            ││
│  │  """                                                         ││
│  │                                                              ││
│  │  [KB Context - may be in Italian]                            ││
│  │  {kb_chunks}                                                 ││
│  │                                                              ││
│  │  [User Message - in Persian]                                 ││
│  │  {user_message}                                              ││
│  │                                                              ││
│  │  → GPT-4 generates response in Persian                       ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Implementation**:

```go
// In apps/ai/language.go

type LanguageManager struct {
    detector *langdetect.Detector
}

// DetectLanguage detects user language with fallback chain
func (l *LanguageManager) DetectLanguage(
    client *models.Client,
    conversation *models.Conversation,
    currentMessage string,
    request *http.Request,
) string {
    // 1. Explicit client preference
    if client != nil && client.Language != "" {
        return client.Language
    }

    // 2. Previously detected language for this conversation
    if state := l.getConversationState(conversation.ID); state != nil && state.DetectedLanguage != "" {
        return state.DetectedLanguage
    }

    // 3. Detect from current message
    if detected := l.detectFromText(currentMessage); detected != "" {
        l.saveDetectedLanguage(conversation.ID, detected)
        return detected
    }

    // 4. Accept-Language header
    if request != nil {
        if lang := l.parseAcceptLanguage(request); lang != "" {
            return lang
        }
    }

    // 5. Default
    return "en"
}

func (l *LanguageManager) detectFromText(text string) string {
    // Use langdetect library
    detected, err := l.detector.DetectLanguage(text)
    if err != nil {
        return ""
    }
    return detected.Lang
}

// BuildLanguageInstructions creates system prompt instructions for translation
func (l *LanguageManager) BuildLanguageInstructions(targetLanguage string, kbLanguage string) string {
    languageNames := map[string]string{
        "en": "English", "it": "Italian", "fa": "Persian",
        "ar": "Arabic", "de": "German", "fr": "French",
        "es": "Spanish", "zh": "Chinese", "ja": "Japanese",
        // ... add more
    }

    targetName := languageNames[targetLanguage]
    if targetName == "" {
        targetName = targetLanguage
    }

    return fmt.Sprintf(`
LANGUAGE INSTRUCTIONS:
- The user is communicating in %s.
- You MUST respond ONLY in %s.
- The knowledge base content below may be in a different language (%s).
- Translate all relevant information to %s when responding.
- Preserve technical terms, brand names, and URLs unchanged.
- Maintain a natural, helpful tone in %s.
`, targetName, targetName, kbLanguage, targetName, targetName)
}
```

---

### 6.4 Combined Handover Triggers Summary

All conditions that trigger handover to human:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Complete Handover Triggers                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. BUDGET EXCEEDED                                             │
│     Condition: conversation.total_cost >= config.max_cost       │
│                OR conversation.total_tokens >= config.max_tokens│
│     Message: "I've reached my assistance limit for this         │
│              conversation. Let me connect you with a specialist.│
│                                                                  │
│  2. TIMEOUT                                                     │
│     Condition: AI response time > config.response_timeout_seconds│
│     Message: "I apologize for the delay. A human agent will     │
│              assist you shortly."                               │
│                                                                  │
│  3. EXPLICIT REQUEST                                            │
│     Condition: User message matches handover keywords           │
│     Keywords: "human", "agent", "person", "transfer", etc.      │
│     Message: "I'll connect you with a human agent right away."  │
│                                                                  │
│  4. FRUSTRATION DETECTED                                        │
│     Condition: Sentiment score > config.sentiment_threshold     │
│     Signals: Profanity, ALL CAPS, "!!!", negative sentiment     │
│     Message: "I understand this is frustrating. Let me get      │
│              someone who can help you better."                  │
│                                                                  │
│  5. AI FAILURE                                                  │
│     Condition: - Low confidence score                           │
│                - No KB results found                            │
│                - Same question 3+ times                         │
│                - AI responds "I don't know"                     │
│     Message: "I'm not able to fully assist with this. A human   │
│              agent will take over."                             │
│                                                                  │
│  6. WORKFLOW DEAD END                                           │
│     Condition: Workflow reaches node with no valid next step    │
│     Message: "Let me connect you with someone who can help."    │
│                                                                  │
│  7. JS FUNC FAILURE                                             │
│     Condition: JS function fails repeatedly (> max_retries)     │
│     Message: "I'm having technical difficulties. A human agent  │
│              will assist you."                                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Implementation Roadmap

### Phase 1: Foundation
| Task | Description | Dependencies |
|------|-------------|--------------|
| 1.1 | Create AI app structure (`apps/ai/`) | None |
| 1.2 | Implement `AIConfiguration` model | 1.1 |
| 1.3 | Implement `AIBudgetConfig` and `AIConversationUsage` models | 1.1 |
| 1.4 | Implement `AITimeoutConfig` model | 1.1 |
| 1.5 | Create OpenAI client wrapper with budget/timeout | 1.2, 1.3, 1.4 |
| 1.6 | Implement basic AI responder (intercept client messages) | 1.5 |
| 1.7 | Add admin APIs for AI configuration | 1.2 |
| 1.8 | Implement language detection | 1.6 |

### Phase 2: Knowledge Base Integration
| Task | Description | Dependencies |
|------|-------------|--------------|
| 2.1 | Set up Qdrant client | Phase 1 |
| 2.2 | Implement semantic chunker for KB articles | 2.1 |
| 2.3 | Create `KBVectorIndex` model | 2.1 |
| 2.4 | Implement KB indexing pipeline (NATS event-driven) | 2.2, 2.3 |
| 2.5 | Implement RAG search with multilingual embeddings | 2.4 |
| 2.6 | Add KB context to AI prompts with translation | 2.5 |
| 2.7 | Implement hybrid search (vector + keyword) | 2.5 |

### Phase 3: Handover System
| Task | Description | Dependencies |
|------|-------------|--------------|
| 3.1 | Create `HandoverConfig` and `HandoverEvent` models | Phase 1 |
| 3.2 | Implement explicit handover detection (keywords) | 3.1 |
| 3.3 | Implement frustration detection (sentiment) | 3.1 |
| 3.4 | Implement AI failure detection | 3.1 |
| 3.5 | Implement budget-exceeded handover | 1.3, 3.1 |
| 3.6 | Implement timeout handover | 1.4, 3.1 |
| 3.7 | Add handover webhooks and NATS events | 3.2-3.6 |
| 3.8 | Generate AI summary for handover | 3.7 |

### Phase 4: JS Func System
| Task | Description | Dependencies |
|------|-------------|--------------|
| 4.1 | Create `JSFunc` and `JSFuncExecution` models | Phase 1 |
| 4.2 | Implement Deno runtime executor | 4.1 |
| 4.3 | Implement input/output schema validation | 4.2 |
| 4.4 | Create OpenAI function calling integration | 4.3 |
| 4.5 | Implement secret injection for JS Funcs | 4.2 |
| 4.6 | Add admin APIs for JS Func management | 4.1 |
| 4.7 | Implement NATS-based cache invalidation | 4.6 |

### Phase 5: Workflow Engine
| Task | Description | Dependencies |
|------|-------------|--------------|
| 5.1 | Create `Workflow` and `WorkflowExecution` models | Phase 1 |
| 5.2 | Implement workflow parser (JSON → nodes) | 5.1 |
| 5.3 | Implement node executors (prompt, message, condition, etc.) | 5.2 |
| 5.4 | Integrate JS Func nodes into workflow | 4.4, 5.3 |
| 5.5 | Implement workflow state persistence | 5.3 |
| 5.6 | Add handover node type | 3.7, 5.3 |
| 5.7 | Create admin APIs for workflow management | 5.1 |
| 5.8 | Implement workflow versioning | 5.7 |

### Phase 6: Visual Workflow Editor (Frontend)
| Task | Description | Dependencies |
|------|-------------|--------------|
| 6.1 | Design workflow editor UI (React Flow) | 5.7 |
| 6.2 | Implement node palette and drag-drop | 6.1 |
| 6.3 | Implement connection validation | 6.2 |
| 6.4 | Add workflow testing/preview mode | 6.3 |
| 6.5 | Implement workflow import/export | 6.3 |

### Phase 7: Analytics & Optimization
| Task | Description | Dependencies |
|------|-------------|--------------|
| 7.1 | Create AI analytics models | All phases |
| 7.2 | Implement dashboard for AI metrics | 7.1 |
| 7.3 | Add cost tracking reports | 7.1 |
| 7.4 | Implement A/B testing for prompts | 7.1 |
| 7.5 | Add KB gap detection | 2.5, 7.1 |

---

## 8. Database Schema Overview

### New Tables Required

```sql
-- AI Configuration
ai_configurations
ai_budget_configs
ai_timeout_configs
ai_personalities

-- Usage Tracking
ai_conversation_usages
ai_daily_usages
ai_conversation_states

-- Knowledge Base Vectors
kb_vector_indexes
kb_chunks (extend existing)

-- JS Functions
js_funcs
js_func_executions
js_func_secrets

-- Workflows
workflows
workflow_executions

-- Handover
handover_configs
handover_events
```

### Key Relationships

```
Conversation (1) ──── (1) AIConversationUsage
Conversation (1) ──── (1) AIConversationState
Conversation (1) ──── (0..1) WorkflowExecution
Conversation (1) ──── (0..*) JSFuncExecution
Conversation (1) ──── (0..1) HandoverEvent

Department (1) ──── (0..1) AIConfiguration
Department (1) ──── (0..1) AIBudgetConfig
Department (1) ──── (0..1) HandoverConfig

KnowledgeBaseArticle (1) ──── (1) KBVectorIndex
KnowledgeBaseArticle (1) ──── (*) KnowledgeBaseChunk

Workflow (1) ──── (*) WorkflowExecution
JSFunc (1) ──── (*) JSFuncExecution
```

---

## 9. Configuration Example

```yaml
# config.yml additions

AI:
  # Provider settings
  Provider: "openai"
  Model: "gpt-4-turbo"
  APIKey: "${OPENAI_API_KEY}"

  # Response settings
  Temperature: 0.7
  MaxTokens: 1000
  MaxHistoryMessages: 20

  # Budget defaults
  MaxTokensPerConversation: 50000
  MaxCostPerConversation: 1.00  # USD

  # Timeout defaults (seconds)
  ResponseTimeout: 120
  OpenAIRequestTimeout: 60
  QdrantSearchTimeout: 10
  JSFuncExecutionTimeout: 30

  # Handover settings
  HandoverOnBudgetExceeded: true
  HandoverOnTimeout: true

Qdrant:
  URL: "http://localhost:6333"
  Collection: "homa_kb"
  EmbeddingModel: "text-embedding-3-small"
  EmbeddingDimension: 1536

JSFunc:
  Runtime: "deno"  # or "goja", "docker"
  DefaultTimeout: 30000  # ms
  MaxMemoryMB: 128
  AllowedDomains: []  # Empty = all allowed
```

---

## 10. API Endpoints Summary

### Admin APIs (`/api/admin/ai/`)

```
# AI Configuration
GET    /api/admin/ai/config
PUT    /api/admin/ai/config
GET    /api/admin/ai/config/departments/:id
PUT    /api/admin/ai/config/departments/:id

# Budget Configuration
GET    /api/admin/ai/budget
PUT    /api/admin/ai/budget

# Personalities
GET    /api/admin/ai/personalities
POST   /api/admin/ai/personalities
PUT    /api/admin/ai/personalities/:id
DELETE /api/admin/ai/personalities/:id

# JS Functions
GET    /api/admin/ai/jsfuncs
POST   /api/admin/ai/jsfuncs
GET    /api/admin/ai/jsfuncs/:id
PUT    /api/admin/ai/jsfuncs/:id
DELETE /api/admin/ai/jsfuncs/:id
POST   /api/admin/ai/jsfuncs/:id/test

# Workflows
GET    /api/admin/ai/workflows
POST   /api/admin/ai/workflows
GET    /api/admin/ai/workflows/:id
PUT    /api/admin/ai/workflows/:id
DELETE /api/admin/ai/workflows/:id
POST   /api/admin/ai/workflows/:id/test

# Handover Configuration
GET    /api/admin/ai/handover
PUT    /api/admin/ai/handover

# Analytics
GET    /api/admin/ai/analytics/usage
GET    /api/admin/ai/analytics/costs
GET    /api/admin/ai/analytics/handovers
GET    /api/admin/ai/analytics/jsfuncs
```

### Agent APIs (`/api/agent/ai/`)

```
# Conversation AI State
GET    /api/agent/conversations/:id/ai-state
POST   /api/agent/conversations/:id/ai/enable
POST   /api/agent/conversations/:id/ai/disable

# Manual handover
POST   /api/agent/conversations/:id/handover
```

### Internal/Webhook Events

```
# NATS Subjects
conversation.{id}           # AI responses published here
ai.handover                 # Handover events
kb.article.created          # Trigger KB indexing
kb.article.updated          # Trigger KB re-indexing
kb.article.deleted          # Remove from Qdrant
jsfunc.updated              # Cache invalidation
workflow.updated            # Cache invalidation

# Webhook Events
ai.response.created         # AI sent a message
ai.handover.triggered       # Handover occurred
ai.budget.exceeded          # Budget limit reached
ai.timeout.occurred         # Response timeout
```

---

## 11. Security Considerations

### JS Func Sandboxing

```
┌─────────────────────────────────────────────────────────────────┐
│                    JS Func Security Layers                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Layer 1: Deno Permissions                                      │
│  - --allow-net=specific.domains.only                            │
│  - --allow-read=/specific/paths                                 │
│  - --allow-env=SPECIFIC_VARS                                    │
│  - --no-prompt (deny interactive permissions)                   │
│                                                                  │
│  Layer 2: Resource Limits                                       │
│  - Timeout enforcement (kill after X seconds)                   │
│  - Memory limit (via cgroups/Docker)                            │
│  - CPU limit (via cgroups/Docker)                               │
│                                                                  │
│  Layer 3: Network Isolation                                     │
│  - Whitelist external API domains                               │
│  - Block internal network access                                │
│  - No localhost/127.0.0.1 access                                │
│                                                                  │
│  Layer 4: Secret Management                                     │
│  - Secrets injected as environment variables                    │
│  - Never logged or stored in output                             │
│  - Encrypted at rest                                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Protection

- **PII Filtering**: Optionally filter PII before sending to OpenAI
- **Audit Logging**: All AI interactions logged with timestamps
- **Encryption**: API keys encrypted at rest
- **Rate Limiting**: Per-client, per-conversation limits

---

## 12. Summary

The proposed AI bot system is ambitious but achievable. Key architectural decisions:

1. **Use OpenAI Function Calling** for tool/JS Func integration
2. **Use Qdrant** with chunked KB articles for RAG
3. **Use Deno** for secure JS execution
4. **Store workflows as JSON** in MySQL for horizontal scaling
5. **Use NATS** for cross-instance cache invalidation
6. **Implement 7-layer handover detection** (budget, timeout, explicit, frustration, AI failure, workflow dead end, JS Func failure)
7. **Use multilingual embeddings** for cross-language KB search
8. **Inline translation via GPT** for multi-language responses

### Confirmed Requirements

| Requirement | Implementation |
|-------------|----------------|
| Single tenant | No tenant isolation needed |
| No GDPR | Standard data handling |
| Cost budget | `AIBudgetConfig` + `AIConversationUsage` |
| Multi-language | Language detection + GPT translation |
| Timeout handover | `AITimeoutConfig` with configurable limits |

### Technology Stack

| Component | Choice |
|-----------|--------|
| LLM Provider | OpenAI (GPT-4) |
| Vector Database | Qdrant |
| Embeddings | text-embedding-3-small (multilingual) |
| JS Runtime | Deno (sandboxed) |
| Messaging | NATS (existing) |
| Database | MySQL (existing) |
| Workflow Storage | JSON in MySQL |

The system integrates naturally with Homa's existing event-driven architecture via GORM hooks and NATS pub/sub.

---

*Document Version: 1.0*
*Generated: 2025-12-28*
*Author: AI Architecture Analysis*
*For: Homa Backend - AI Bot Feature Implementation*
