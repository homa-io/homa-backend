# Project Improvements Analysis

**Date:** January 17, 2026
**Scope:** Full codebase analysis of backend (Go) and dashboard (Next.js)

---

## Table of Contents

1. [Backend (Go) Analysis](#backend-go-analysis)
   - [Security Issues](#1-security-issues---critical)
   - [Large Files to Split](#2-large-files-to-split)
   - [Code Duplication](#3-code-duplication)
   - [Performance Issues](#4-performance-issues)
   - [Incomplete Features](#5-incomplete-features)
2. [Dashboard (Next.js) Analysis](#dashboard-nextjs-analysis)
   - [Security Issues](#1-security-issues)
   - [Large Components to Split](#2-large-components-to-split)
   - [Code Quality Issues](#3-code-quality-issues)
   - [Performance Issues](#4-performance-issues-1)
   - [Missing Features](#5-missing-features)
3. [Priority Matrix](#priority-matrix)
4. [Action Plan](#action-plan)

---

## Backend (Go) Analysis

**Total Files:** 118
**Total Lines:** 31,252
**Framework:** Evo v2 with GORM ORM

### 1. Security Issues - CRITICAL

#### A. Missing Encryption for Integration Configs

**File:** `apps/integrations/driver.go` lines 120-129

```go
// EncryptConfig encrypts configuration JSON (placeholder - implement proper encryption).
func EncryptConfig(config string) string {
    // TODO: Implement proper encryption using AES-256-GCM
    return config  // Currently returns plain text!
}
```

**Risk:** API keys, tokens, and secrets for Gmail, Outlook, Slack, WhatsApp, Telegram stored in plain text.

**Fix Required:**
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

#### B. Input Validation Gaps

**File:** `apps/admin/controller.go` lines 137-144

```go
search := request.Query("search").String()
if search != "" {
    query = query.Where("title LIKE ?", "%"+search+"%")
}
```

**Issues:**
- No max length validation (DoS risk with very long strings)
- No sanitization of special characters
- 6 subqueries with unvalidated input

**Fix:** Add validation before use:
```go
if len(search) > 100 {
    return response.Error(response.ErrInvalidInput)
}
search = strings.TrimSpace(search)
```

#### C. API Key Management

**File:** `apps/admin/controller.go` lines 1304-1318

- Blocking user doesn't revoke existing tokens
- No API key expiration mechanism
- No per-key rate limiting

---

### 2. Large Files to Split

| File | Lines | Priority | Recommendation |
|------|-------|----------|----------------|
| `conversation/agent_controller.go` | 3036 | HIGH | Split into `conversation_repository.go`, `conversation_search.go`, `conversation_formatter.go` |
| `swagger/generator.go` | 1930 | MEDIUM | Split into `openapi_schemas.go`, `openapi_paths.go`, `openapi_generators.go` |
| `admin/controller.go` | 1765 | HIGH | Split by domain: `ticket_controller.go`, `department_controller.go`, `user_controller.go`, `attribute_controller.go`, `channel_controller.go` |
| `integrations/webhooks.go` | 1016 | MEDIUM | Split by channel: `handlers/slack.go`, `handlers/gmail.go`, `handlers/whatsapp.go` |

---

### 3. Code Duplication

#### A. Access Control Logic (3 occurrences)

**Locations:** `apps/admin/controller.go` lines 30-53, 72-92, 106-115

```go
// Repeated in GetUnreadTickets, GetUnreadTicketsCount, ListTickets
var userDepartments []uint
err := db.Model(&models.UserDepartment{}).
    Where("user_id = ?", user.UserID).
    Pluck("department_id", &userDepartments).Error
```

**Fix:** Extract to helper function:
```go
func (c *AdminController) getAgentDepartments(userID uuid.UUID) ([]uint, error) {
    var departments []uint
    err := c.db.Model(&models.UserDepartment{}).
        Where("user_id = ?", userID).
        Pluck("department_id", &departments).Error
    return departments, err
}
```

#### B. Duplicate Error Handling Pattern

**Locations:** Lines 648-651, 728-731, 929-931, 1004-1005

```go
if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
    duplicateErr := response.NewError(response.ErrorCodeConflict, "...", 409)
    return response.Error(duplicateErr)
}
```

**Fix:** Create `handleDuplicateError()` helper.

#### C. Ticket Access Verification

Already implemented as `hasTicketAccess()` at line 1728 but not consistently used across all ticket operations.

---

### 4. Performance Issues

#### A. N+1 Query Problems

**File:** `conversation/agent_controller.go` lines 654-675

```go
db.Preload("Client").
    Preload("Department").
    Preload("Channel").
    Preload("Assignments").
    Preload("Assignments.User").
    // Many more preloads...
    Find(&conversations)
```

**Issues:**
- Excessive preload chains without conditions
- No JOIN optimization
- Each preload generates separate query

**Fix:** Use conditional preloads and JOINs:
```go
db.Joins("Client").
    Joins("Department").
    Preload("Assignments", func(db *gorm.DB) *gorm.DB {
        return db.Limit(10)
    }).
    Find(&conversations)
```

#### B. Inefficient Search

**File:** `apps/admin/controller.go` lines 139-145

```go
query = query.Where(
    "id = ? OR title LIKE ? OR id IN (SELECT conversation_id FROM messages WHERE body LIKE ?) OR "+
    "client_id IN (SELECT id FROM clients WHERE name LIKE ?) OR "+
    "client_id IN (SELECT client_id FROM client_external_ids WHERE value LIKE ?) OR "+
    "id IN (SELECT conversation_id FROM conversation_tags JOIN tags ON ... WHERE tags.name LIKE ?)",
    parseIntOrZero(search), "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%",
)
```

**Issues:**
- 6 separate subqueries
- `LIKE "%search%"` cannot use indexes
- No result limit on subqueries

**Fix:** Implement full-text search:
```go
// PostgreSQL example
query = query.Where("search_vector @@ plainto_tsquery(?)", search)
```

#### C. Missing Database Indexes

Add composite indexes for frequently used queries:

```go
// In migration
db.Exec("CREATE INDEX idx_messages_conv_created ON messages(conversation_id, created_at)")
db.Exec("CREATE INDEX idx_conversations_dept_status ON conversations(department_id, status)")
db.Exec("CREATE INDEX idx_assignments_user_dept ON conversation_assignments(user_id, department_id)")
```

---

### 5. Incomplete Features

#### A. TODO Jobs Not Implemented

**File:** `apps/jobs/jobs.go` lines 90-182

| Job | Line | Status |
|-----|------|--------|
| `handleSendCSATEmails` | 93 | TODO |
| `handleCalculateMetrics` | 108 | TODO |
| `handleCloseUnresponded` | 125 | TODO |
| `handleArchiveOldTickets` | 141 | TODO |
| `handleDeleteOldTickets` | 154 | TODO |

#### B. Ignored Database Errors

**File:** `apps/admin/controller.go`

```go
db.Create(&message)  // Line 229 - error ignored
db.Create(&message)  // Line 369 - error ignored
db.Create(&message)  // Line 422 - error ignored
```

**Fix:** Always handle errors:
```go
if err := db.Create(&message).Error; err != nil {
    log.Error("Failed to create message: %v", err)
    return response.Error(response.ErrDatabaseOperation)
}
```

#### C. Resource Cleanup

Found 26 manual cleanup statements without `defer`. Potential resource leaks in error paths.

---

## Dashboard (Next.js) Analysis

**Total Components:** 200+
**Framework:** Next.js 13+ with App Router

### 1. Security Issues

#### A. XSS Vulnerability

**File:** `src/components/knowledge-base/ArticleEditor.tsx` line 1100

```typescript
dangerouslySetInnerHTML={{
  __html: editor?.getHTML() || '<p>No content yet...</p>'
}}
```

**Risk:** If editor content contains malicious HTML, it could execute scripts.

**Fix:** Sanitize with DOMPurify:
```typescript
import DOMPurify from 'dompurify'

dangerouslySetInnerHTML={{
  __html: DOMPurify.sanitize(editor?.getHTML() || '<p>No content yet...</p>')
}}
```

#### B. Weak Password Generation

**File:** `src/services/users.ts`

```typescript
password: Math.random().toString(36).substring(2, 18) + Math.random().toString(36).substring(2, 18)
```

**Risk:** `Math.random()` is NOT cryptographically secure.

**Fix:**
```typescript
import { randomBytes } from 'crypto'
const password = randomBytes(16).toString('base64')
```

---

### 2. Large Components to Split

| Component | Lines | Split Recommendation |
|-----------|-------|---------------------|
| `ArticleEditor.tsx` | 1138 | `ArticleEditor` (container), `ArticleForm` (metadata), `ArticleContent` (editor), `MediaGallery`, `FeaturedImageUpload` |
| `SDKSettings.tsx` | 1066 | `SDKSettingsForm`, `SDKPreview`, `SDKColorPicker`, `SDKPositionSelector` |
| `VisitorInformation.tsx` | 833 | `VisitorDetails`, `VisitorHistory`, `PreviousConversations` |
| `AIAgentEditPage.tsx` | 814 | `AIAgentForm`, `ToolsManager`, `AgentValidation` |
| `WysiwygEditor.tsx` | 796 | `EditorToolbar`, `EditorContent`, `AIFeatures`, `SlashCommandMenu` |
| `RAGSettings.tsx` | 751 | `RAGDocuments`, `RAGConfiguration`, `RAGStatus` |
| `ConversationModal.tsx` | 669 | `ModalHeader`, `ModalContent`, `ModalActions` |
| `users/index.tsx` | 649 | `UserList`, `UserFilters`, `UserActions` |
| `AttributeManager.tsx` | 612 | `AttributeList`, `AttributeForm`, `AttributePreview` |

---

### 3. Code Quality Issues

#### A. Type Safety (162 `any` instances)

**Common patterns:**
```typescript
error: any      // Should be: error: Error | AxiosError
data: any       // Should be: data: ConversationResponse
response: any   // Should be: response: APIResponse<T>
```

**Fix:** Create proper interfaces:
```typescript
interface APIResponse<T> {
  success: boolean
  data: T
  error?: string
}

interface ConversationMessage {
  id: number
  body: string
  type: 'message' | 'action'
  language: string
  // ...
}
```

#### B. Duplicate Token Logic

**File:** `src/services/conversation.service.ts` (4 instances)

```typescript
// Repeated pattern instead of using centralized utility
const cookies = document.cookie.split('; ')
const tokenCookie = cookies.find(c => c.startsWith('access_token='))
const token = tokenCookie ? tokenCookie.split('=')[1] : null
```

**Fix:** Use existing utility:
```typescript
import { getAccessToken } from '@/lib/cookies'
const token = getAccessToken()
```

#### C. Dead Code

**File:** `src/data/mockKnowledgeBase.working.ts` (344 lines)

Not imported anywhere in codebase. Delete this file.

---

### 4. Performance Issues

#### A. Missing Memoization

**WysiwygEditor.tsx:** 15+ useState hooks in one component without proper memoization.

```typescript
// EditorToolbar rendered on every keystroke
// Should wrap with React.memo():
const EditorToolbar = React.memo(({ editor, onAction }) => {
  // ...
})
```

#### B. Missing Code Splitting

Heavy dependencies loaded eagerly:
- `@tiptap/*` (7 extensions)
- `vanta` + `three`
- `recharts`
- `@uppy/*`

**Fix:** Dynamic imports:
```typescript
const ArticleEditor = dynamic(
  () => import('@/components/knowledge-base/ArticleEditor'),
  { loading: () => <EditorSkeleton /> }
)
```

---

### 5. Missing Features

#### A. Testing (0 test files)

No unit tests, integration tests, or component tests found.

**Priority test targets:**
- `conversation.service.ts` - critical business logic
- `ArticleEditor.tsx` - complex user interactions
- `auth` utilities - security critical

#### B. Accessibility (28 ARIA attributes total)

Missing:
- Keyboard navigation
- Screen reader support
- Focus management in modals
- Color contrast verification

**Fix:** Add ARIA attributes to interactive elements:
```typescript
<Dialog
  aria-modal="true"
  aria-labelledby="dialog-title"
  aria-describedby="dialog-description"
>
```

#### C. Error Boundaries

Only one error boundary exists. Add boundaries for:
- Conversations section
- Settings pages
- Knowledge Base editor
- File upload operations

---

## Priority Matrix

### Critical (Do First)

| Task | Project | Effort | Impact |
|------|---------|--------|--------|
| Implement config encryption | Backend | Medium | Security |
| Sanitize HTML output | Dashboard | Low | Security |
| Fix password generation | Dashboard | Low | Security |
| Add input validation | Backend | Low | Security |

### High Priority

| Task | Project | Effort | Impact |
|------|---------|--------|--------|
| Split `agent_controller.go` | Backend | High | Maintainability |
| Split `ArticleEditor.tsx` | Dashboard | High | Maintainability |
| Split `SDKSettings.tsx` | Dashboard | High | Maintainability |
| Fix N+1 queries | Backend | Medium | Performance |
| Add database indexes | Backend | Low | Performance |

### Medium Priority

| Task | Project | Effort | Impact |
|------|---------|--------|--------|
| Extract duplicate code | Backend | Medium | Maintainability |
| Refactor token logic | Dashboard | Low | Code Quality |
| Add TypeScript interfaces | Dashboard | Medium | Type Safety |
| Implement error handling | Backend | Medium | Reliability |

### Low Priority

| Task | Project | Effort | Impact |
|------|---------|--------|--------|
| Add unit tests | Both | High | Quality |
| Improve accessibility | Dashboard | Medium | UX |
| Add documentation | Both | Medium | Maintainability |
| Delete dead code | Dashboard | Low | Cleanup |

---

## Action Plan

### Week 1: Security Fixes
1. [ ] Implement AES-256-GCM encryption for integration configs
2. [ ] Add DOMPurify sanitization to ArticleEditor and WysiwygEditor
3. [ ] Replace Math.random() with crypto module for password generation
4. [ ] Add input validation for search queries

### Week 2: Code Splitting (Backend)
1. [ ] Split `agent_controller.go` into modules
2. [ ] Extract duplicate access control logic
3. [ ] Implement consistent error handling

### Week 3: Code Splitting (Dashboard)
1. [ ] Split `ArticleEditor.tsx` into smaller components
2. [ ] Split `SDKSettings.tsx` into sections
3. [ ] Refactor conversation.service.ts to use centralized auth

### Week 4: Performance
1. [ ] Add database composite indexes
2. [ ] Optimize N+1 queries with JOINs
3. [ ] Implement code splitting for heavy components
4. [ ] Add React.memo() to frequently rendered components

### Ongoing
- Add unit tests for new code
- Improve accessibility incrementally
- Document API endpoints and component props

---

## Appendix: Commands

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
