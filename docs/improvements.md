# Project Improvements Analysis

**Date:** January 17, 2026
**Last Updated:** January 17, 2026
**Scope:** Full codebase analysis of backend (Go) and dashboard (Next.js)

---

## Progress Summary

| Category | Total Issues | Fixed | Remaining |
|----------|-------------|-------|-----------|
| Security (Critical) | 6 | 4 | 2 |
| Code Duplication | 3 | 3 | 0 |
| Performance | 3 | 2 | 1 |
| Code Quality | 4 | 3 | 1 |
| Large Files | 13 | 0 | 13 |
| Missing Features | 7 | 0 | 7 |

---

## Table of Contents

1. [Completed Fixes](#completed-fixes)
2. [Remaining Issues](#remaining-issues)
3. [Feature Suggestions](#feature-suggestions)
4. [Backend Analysis](#backend-go-analysis)
5. [Dashboard Analysis](#dashboard-nextjs-analysis)
6. [Priority Matrix](#priority-matrix)
7. [Action Plan](#action-plan)

---

## Completed Fixes

### Backend (Go)

#### 1. Input Validation - FIXED (Commit: 0b0d4ef)

**File:** `apps/admin/controller.go`

Added `sanitizeSearch()` helper function that:
- Limits search input to 100 characters
- Escapes SQL special characters (`\`, `%`, `_`)
- Added `ESCAPE '\\'` clause to all LIKE queries
- Applied to all 5 search endpoints

```go
// sanitizeSearch validates and sanitizes search input to prevent DoS and injection
func sanitizeSearch(s string) string {
    s = strings.TrimSpace(s)
    if len(s) > 100 {
        s = s[:100]
    }
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, "%", "\\%")
    s = strings.ReplaceAll(s, "_", "\\_")
    return s
}
```

#### 2. Access Control Logic Duplication - FIXED (Commit: 92a84a6)

**File:** `apps/admin/controller.go`

Extracted duplicate access control logic into `getAgentDepartments()` helper:

```go
// getAgentDepartments returns the department IDs assigned to an agent
func getAgentDepartments(userID uuid.UUID) ([]uint, error) {
    var departments []uint
    err := db.Model(&models.UserDepartment{}).
        Where("user_id = ?", userID).
        Pluck("department_id", &departments).Error
    return departments, err
}
```

Replaced 3 duplicate blocks with helper function calls.

#### 3. Duplicate Error Handling Pattern - FIXED (Commit: 92a84a6)

**File:** `apps/admin/controller.go`

Created `isDuplicateError()` helper to simplify duplicate constraint checking:

```go
// isDuplicateError checks if an error is a duplicate/unique constraint violation
func isDuplicateError(err error) bool {
    return err != nil && (strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique"))
}
```

Simplified 7 duplicate error handling patterns.

#### 4. N+1 Query Problems - FIXED (Commit: 92a84a6)

**File:** `apps/conversation/agent_controller.go`

- Fixed GetTags N+1 query with single JOIN query
- Optimized GetConversations using `Joins()` for 1-to-1 relationships
- Added conditional preloads with limits for 1-to-many relationships

```go
// Before: N+1 queries for each conversation
// After: Single JOIN query
if err := query.
    Joins("Client").
    Joins("Department").
    Joins("Channel").
    Preload("Tags").
    Preload("Assignments", func(db *gorm.DB) *gorm.DB {
        return db.Order("priority ASC").Limit(10)
    }).
    Preload("Assignments.User").
    Find(&conversations).Error; err != nil {
```

#### 5. Ignored Database Errors - FIXED (Commit: d287319)

**File:** `apps/admin/controller.go`

Fixed 3 `db.Create(&message)` calls that were ignoring errors:
- Line 229: Message creation in ticket resolution
- Line 369: Message creation in ticket assignment
- Line 422: Message creation in ticket close

Added proper error logging for all cases.

#### 6. Transaction Safety - FIXED (Commit: d287319)

**File:** `apps/admin/controller.go`

Improved 4 transaction blocks with:
- `tx.Error` check after `db.Begin()`
- `defer` with panic recovery
- `tx.Commit().Error` check

```go
tx := db.Begin()
if tx.Error != nil {
    log.Error("Failed to start transaction: %v", tx.Error)
    return response.Error(response.ErrInternalError)
}
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
        panic(r)
    }
}()
// ... operations ...
if err = tx.Commit().Error; err != nil {
    log.Error("Failed to commit transaction: %v", err)
    return response.Error(response.ErrInternalError)
}
```

---

### Dashboard (Next.js)

#### 1. XSS Vulnerability - FIXED (Commit: 8e68398)

**File:** `src/components/knowledge-base/ArticleEditor.tsx`

Added DOMPurify sanitization:

```typescript
import DOMPurify from "dompurify"

dangerouslySetInnerHTML={{
    __html: DOMPurify.sanitize(editor?.getHTML() || '<p>No content yet...</p>')
}}
```

#### 2. Weak Password Generation - FIXED (Commit: 8e68398)

**File:** `src/services/users.ts`

Replaced insecure `Math.random()` with cryptographically secure generation:

```typescript
function generateSecureRandomString(length: number): string {
  const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  const randomValues = new Uint8Array(length);
  crypto.getRandomValues(randomValues);
  return Array.from(randomValues, (byte) => charset[byte % charset.length]).join('');
}
```

#### 3. Dead Code Removed - FIXED (Commit: 8e68398)

Deleted `src/data/mockKnowledgeBase.working.ts` (344 lines of unused code).

Updated imports in 3 knowledge-base pages to use correct module path.

#### 4. Duplicate Token Logic - FIXED (Commit: 8e68398)

**File:** `src/services/conversation.service.ts`

Refactored to use centralized `getAccessToken()` from `@/lib/cookies`:

```typescript
import { getAccessToken } from '@/lib/cookies'
// Replaced 4 duplicate implementations with single import
const token = getAccessToken()
```

---

## Remaining Issues

### Backend - Critical

#### 1. Missing Encryption for Integration Configs

**File:** `apps/integrations/driver.go` lines 120-129

**Status:** NOT FIXED - High Priority

```go
// EncryptConfig encrypts configuration JSON (placeholder - implement proper encryption).
func EncryptConfig(config string) string {
    // TODO: Implement proper encryption using AES-256-GCM
    return config  // Currently returns plain text!
}
```

**Risk:** API keys, tokens, and secrets for Gmail, Outlook, Slack, WhatsApp, Telegram stored in plain text.

**Required Fix:**
```go
func EncryptConfig(config string) (string, error) {
    key := []byte(os.Getenv("ENCRYPTION_KEY"))
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    ciphertext := gcm.Seal(nonce, nonce, []byte(config), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

#### 2. API Key Management

**File:** `apps/admin/controller.go` lines 1304-1318

**Status:** NOT FIXED - Medium Priority

Issues:
- Blocking user doesn't revoke existing tokens
- No API key expiration mechanism
- No per-key rate limiting

---

### Backend - Large Files to Split

| File | Lines | Priority | Status |
|------|-------|----------|--------|
| `conversation/agent_controller.go` | 3,034 | HIGH | NOT STARTED |
| `swagger/generator.go` | 1,930 | MEDIUM | NOT STARTED |
| `admin/controller.go` | 1,765 | HIGH | NOT STARTED |
| `integrations/webhooks.go` | 1,016 | MEDIUM | NOT STARTED |

---

### Backend - Incomplete Features

#### TODO Jobs Not Implemented

**File:** `apps/jobs/jobs.go` lines 90-182

| Job | Line | Status |
|-----|------|--------|
| `handleSendCSATEmails` | 93 | TODO |
| `handleCalculateMetrics` | 108 | TODO |
| `handleCloseUnresponded` | 125 | TODO |
| `handleArchiveOldTickets` | 141 | TODO |
| `handleDeleteOldTickets` | 154 | TODO |

---

### Dashboard - Remaining Issues

#### 1. ESLint Errors (3 errors)

**File:** `src/components/knowledge-base/ArticleEditor.tsx`

```
Line 12:3  'ImageIcon' redeclares variable from line 8
Line 15:3  'Settings' redeclares variable from line 10
Line 16:3  'Tag' redeclares variable from line 11
```

**Fix:** Rename imported icons to avoid conflicts with tiptap extension names.

#### 2. Type Safety Issues (65 `any` instances)

Remaining `any` type usages that should be properly typed.

#### 3. Large Components to Split

| Component | Lines | Priority | Status |
|-----------|-------|----------|--------|
| `ArticleEditor.tsx` | 1,139 | HIGH | NOT STARTED |
| `SDKSettings.tsx` | 1,066 | HIGH | NOT STARTED |
| `VisitorInformation.tsx` | 833 | MEDIUM | NOT STARTED |
| `AIAgentEditPage.tsx` | 814 | MEDIUM | NOT STARTED |
| `WysiwygEditor.tsx` | 796 | MEDIUM | NOT STARTED |
| `RAGSettings.tsx` | 751 | MEDIUM | NOT STARTED |
| `ConversationModal.tsx` | 669 | LOW | NOT STARTED |
| `users/index.tsx` | 649 | LOW | NOT STARTED |
| `AttributeManager.tsx` | 612 | LOW | NOT STARTED |

#### 4. Missing Error Boundaries

**File:** `app/` directory

**Status:** 0% coverage

Need error boundaries for:
- Conversations section
- Settings pages
- Knowledge Base editor
- File upload operations

#### 5. Missing Tests

**Status:** 0 test files found

Priority test targets:
- `conversation.service.ts` - critical business logic
- `ArticleEditor.tsx` - complex user interactions
- Auth utilities - security critical

---

## Feature Suggestions

### High Priority Features

#### 1. Full-Text Search

Replace current LIKE-based search with PostgreSQL full-text search:

```go
// Current: Slow LIKE queries with subqueries
query = query.Where("title LIKE ?", "%"+search+"%")

// Suggested: Full-text search
query = query.Where("search_vector @@ plainto_tsquery(?)", search)
```

Benefits:
- 10-100x faster search performance
- Better relevance ranking
- Support for stemming and fuzzy matching

#### 2. Webhook Retry Mechanism

**File:** `apps/integrations/webhooks.go`

Current behavior: Single attempt, no retry on failure.

Suggested implementation:
- Exponential backoff (1s, 2s, 4s, 8s, 16s)
- Max 5 retry attempts
- Dead letter queue for failed webhooks
- Webhook status dashboard

#### 3. Rate Limiting per API Key

**File:** `apps/admin/controller.go`

Current: No rate limiting on API endpoints.

Suggested:
- Redis-based sliding window rate limiter
- Per-key limits stored in database
- Rate limit headers in responses
- Automatic temporary blocks

#### 4. Real-Time Notifications (WebSocket)

Currently implemented but could be enhanced:
- Browser notifications for new messages
- Sound notifications (configurable)
- Desktop notification badges
- Unread count sync across tabs

#### 5. Analytics Dashboard

Missing analytics features:
- Response time metrics
- Agent performance metrics
- Channel usage statistics
- Customer satisfaction trends
- Peak hour analysis

### Medium Priority Features

#### 6. Bulk Operations

- Bulk ticket assignment
- Bulk status change
- Bulk tag application
- Bulk export (CSV/Excel)

#### 7. Canned Responses

- Pre-defined response templates
- Category organization
- Variable substitution ({{customer_name}})
- Usage analytics

#### 8. SLA Management

- SLA policy definition
- First response time targets
- Resolution time targets
- Escalation rules
- SLA breach notifications

#### 9. Knowledge Base Enhancements

- Article versioning
- Draft autosave
- Collaborative editing
- Article analytics
- Search suggestions

#### 10. Integration Marketplace

- Pre-built integrations catalog
- One-click installation
- Configuration wizard
- Health monitoring

### Low Priority Features

#### 11. Dark Mode

Dashboard currently lacks dark mode support.

#### 12. Mobile App

Progressive Web App (PWA) for mobile agents.

#### 13. AI Enhancements

- Sentiment analysis on conversations
- Auto-categorization of tickets
- Suggested responses
- Smart routing based on content

---

## Priority Matrix

### Critical (Do First)

| Task | Project | Status | Effort |
|------|---------|--------|--------|
| ~~Sanitize HTML output~~ | Dashboard | DONE | Low |
| ~~Fix password generation~~ | Dashboard | DONE | Low |
| ~~Add input validation~~ | Backend | DONE | Low |
| Implement config encryption | Backend | TODO | Medium |

### High Priority

| Task | Project | Status | Effort |
|------|---------|--------|--------|
| ~~Fix N+1 queries~~ | Backend | DONE | Medium |
| ~~Extract duplicate code~~ | Backend | DONE | Medium |
| ~~Refactor token logic~~ | Dashboard | DONE | Low |
| Split `agent_controller.go` | Backend | TODO | High |
| Split `ArticleEditor.tsx` | Dashboard | TODO | High |
| Fix ESLint errors | Dashboard | TODO | Low |

### Medium Priority

| Task | Project | Status | Effort |
|------|---------|--------|--------|
| Implement TODO jobs | Backend | TODO | High |
| Add database indexes | Backend | TODO | Low |
| Add TypeScript interfaces | Dashboard | TODO | Medium |
| Add error boundaries | Dashboard | TODO | Medium |

### Low Priority

| Task | Project | Status | Effort |
|------|---------|--------|--------|
| Add unit tests | Both | TODO | High |
| Improve accessibility | Dashboard | TODO | Medium |
| Add documentation | Both | TODO | Medium |
| Split remaining large files | Both | TODO | High |

---

## Action Plan

### Phase 1: Security (Immediate)
- [x] Add input validation for search queries
- [x] Add DOMPurify sanitization to ArticleEditor
- [x] Replace Math.random() with crypto module
- [ ] Implement AES-256-GCM encryption for integration configs

### Phase 2: Code Quality (Next)
- [x] Extract duplicate access control logic
- [x] Implement consistent error handling helpers
- [x] Fix ignored database errors
- [x] Add transaction safety
- [x] Refactor token logic to use centralized utility
- [x] Remove dead code
- [ ] Fix ESLint errors in ArticleEditor
- [ ] Replace remaining `any` types

### Phase 3: Performance
- [x] Optimize N+1 queries with JOINs
- [ ] Add database composite indexes
- [ ] Implement code splitting for heavy components
- [ ] Add React.memo() to frequently rendered components

### Phase 4: Code Splitting
- [ ] Split `agent_controller.go` into modules
- [ ] Split `ArticleEditor.tsx` into smaller components
- [ ] Split `SDKSettings.tsx` into sections

### Phase 5: Features
- [ ] Implement TODO jobs (CSAT, metrics, cleanup)
- [ ] Add full-text search
- [ ] Add error boundaries
- [ ] Add unit tests for critical paths

### Ongoing
- Improve accessibility incrementally
- Document API endpoints and component props
- Monitor performance metrics

---

## Appendix: Useful Commands

### Find Large Files (Backend)
```bash
find apps -name "*.go" -exec wc -l {} \; | sort -rn | head -20
```

### Find Large Components (Dashboard)
```bash
find src/components -name "*.tsx" -exec wc -l {} \; | sort -rn | head -20
```

### Count `any` Types
```bash
grep -r ": any" src --include="*.ts" --include="*.tsx" | wc -l
```

### Find Missing Error Handling
```bash
grep -rn "db.Create\|db.Save\|db.Delete" apps --include="*.go" | grep -v "Error"
```

### Check ESLint Errors
```bash
cd /home/evo/homa-dashboard && npm run lint
```

### Verify Build
```bash
cd /home/evo/homa-backend && go build ./...
cd /home/evo/homa-dashboard && npm run build
```
