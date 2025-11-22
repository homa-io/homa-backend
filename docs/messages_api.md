# Conversation Messages API Specification

## Overview
This document specifies the API endpoint needed to retrieve messages for a specific conversation. This is required for the agent dashboard to display the full conversation thread when an agent selects a conversation.

**Base URL:** `http://127.0.0.1:8033`

---

## Endpoint

### Get Conversation Messages

Retrieve all messages for a specific conversation in chronological order.

**Endpoint:** `GET /api/agent/conversations/{conversation_id}/messages`

**Description:** Returns all messages (customer messages, agent replies, system messages) for a given conversation, ordered by creation time.

---

## Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `conversation_id` | integer | Yes | The unique ID of the conversation |

---

## Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number for pagination |
| `limit` | integer | No | 50 | Messages per page (max 100) |
| `order` | string | No | `asc` | Sort order: `asc` (oldest first) or `desc` (newest first) |

---

## Example Requests

### Basic Request
```bash
curl -X GET "http://127.0.0.1:8033/api/agent/conversations/32/messages" \
  -H "Content-Type: application/json"
```

### With Pagination
```bash
curl -X GET "http://127.0.0.1:8033/api/agent/conversations/32/messages?page=1&limit=20&order=asc" \
  -H "Content-Type: application/json"
```

---

## Response Format

### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "conversation_id": 32,
    "page": 1,
    "limit": 50,
    "total": 5,
    "total_pages": 1,
    "messages": [
      {
        "id": 101,
        "body": "Unable to login to dashboard",
        "is_agent": false,
        "is_system_message": false,
        "created_at": "2025-11-21T18:27:44Z",
        "author": {
          "id": "895a1fca-c718-11f0-8d2f-920006a447b5",
          "name": "John Smith",
          "type": "customer",
          "avatar_url": null,
          "initials": "JS"
        },
        "attachments": []
      },
      {
        "id": 102,
        "body": "Thank you for contacting us. Can you please provide more details about the error you're seeing?",
        "is_agent": true,
        "is_system_message": false,
        "created_at": "2025-11-21T18:30:15Z",
        "author": {
          "id": "c724f225-9ac8-4e53-870e-fef78e2b5c98",
          "name": "Admin User",
          "type": "agent",
          "avatar_url": null,
          "initials": "AU"
        },
        "attachments": []
      },
      {
        "id": 103,
        "body": "It says 'Invalid credentials' but I'm sure my password is correct.",
        "is_agent": false,
        "is_system_message": false,
        "created_at": "2025-11-21T18:35:22Z",
        "author": {
          "id": "895a1fca-c718-11f0-8d2f-920006a447b5",
          "name": "John Smith",
          "type": "customer",
          "avatar_url": null,
          "initials": "JS"
        },
        "attachments": []
      },
      {
        "id": 104,
        "body": "I see. Let me help you reset your password. I'll send you a reset link to your email.",
        "is_agent": true,
        "is_system_message": false,
        "created_at": "2025-11-21T18:40:00Z",
        "author": {
          "id": "c724f225-9ac8-4e53-870e-fef78e2b5c98",
          "name": "Admin User",
          "type": "agent",
          "avatar_url": null,
          "initials": "AU"
        },
        "attachments": []
      },
      {
        "id": 105,
        "body": "Got it! Just reset my password and now I can login. Thank you so much!",
        "is_agent": false,
        "is_system_message": false,
        "created_at": "2025-11-21T18:57:44Z",
        "author": {
          "id": "895a1fca-c718-11f0-8d2f-920006a447b5",
          "name": "John Smith",
          "type": "customer",
          "avatar_url": null,
          "initials": "JS"
        },
        "attachments": []
      }
    ]
  }
}
```

---

## Response Fields

### Root Level
| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Whether the request was successful |
| `data` | object | Response data object |

### Data Object
| Field | Type | Description |
|-------|------|-------------|
| `conversation_id` | integer | ID of the conversation |
| `page` | integer | Current page number |
| `limit` | integer | Messages per page |
| `total` | integer | Total number of messages in conversation |
| `total_pages` | integer | Total number of pages |
| `messages` | array | Array of message objects |

### Message Object
| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique message ID |
| `body` | string | The message content/text |
| `is_agent` | boolean | `true` if sent by an agent, `false` if sent by customer |
| `is_system_message` | boolean | `true` if automated/system message, `false` otherwise |
| `created_at` | string | ISO 8601 timestamp when message was sent |
| `author` | object | Information about who sent the message |
| `attachments` | array | Array of attachment objects (empty array if none) |

### Author Object
| Field | Type | Description |
|-------|------|-------------|
| `id` | string | User/Customer UUID |
| `name` | string | Full name of the author |
| `type` | string | Either `"customer"`, `"agent"`, or `"system"` |
| `avatar_url` | string\|null | URL to avatar image (null if not available) |
| `initials` | string | 2-letter initials derived from name |

### Attachment Object (Future Implementation)
| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique attachment ID |
| `name` | string | File name |
| `size` | integer | File size in bytes |
| `type` | string | MIME type (e.g., "image/jpeg", "application/pdf") |
| `url` | string | URL to download/view the attachment |
| `created_at` | string | ISO 8601 timestamp when uploaded |

---

## Error Responses

### Conversation Not Found (404)
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Conversation not found",
    "details": "No conversation exists with ID 32"
  }
}
```

### Invalid Parameters (400)
```json
{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETERS",
    "message": "Invalid request parameters",
    "details": "Limit must be between 1 and 100"
  }
}
```

### Unauthorized (401)
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authentication required",
    "details": "Valid access token required"
  }
}
```

### Server Error (500)
```json
{
  "success": false,
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Failed to retrieve messages",
    "details": "Internal server error"
  }
}
```

---

## Database Implementation Notes

Based on the existing database schema:

### Tables Used
1. **`messages`** - Main table containing message data
   - `id` - Message ID
   - `ticket_id` - Foreign key to conversations (tickets)
   - `user_id` - Foreign key to users (agents)
   - `client_id` - Foreign key to clients (customers)
   - `body` - Message content
   - `is_system_message` - Boolean flag
   - `created_at` - Timestamp

2. **`users`** - Agent information
   - Join when `user_id IS NOT NULL` to get agent details

3. **`clients`** - Customer information
   - Join when `client_id IS NOT NULL` to get customer details

### SQL Query Pattern
```sql
SELECT
    m.id,
    m.body,
    CASE WHEN m.user_id IS NOT NULL THEN true ELSE false END as is_agent,
    m.is_system_message,
    m.created_at,
    COALESCE(u.id, c.id) as author_id,
    COALESCE(u.name, c.name) as author_name,
    CASE
        WHEN m.user_id IS NOT NULL THEN 'agent'
        WHEN m.client_id IS NOT NULL THEN 'customer'
        ELSE 'system'
    END as author_type,
    COALESCE(u.avatar_url, c.avatar_url) as avatar_url
FROM messages m
LEFT JOIN users u ON m.user_id = u.id
LEFT JOIN clients c ON m.client_id = c.id
WHERE m.ticket_id = $1
ORDER BY m.created_at ASC
LIMIT $2 OFFSET $3;
```

### Initials Generation
Generate 2-letter initials from the author's name:
- Split name by spaces
- Take first letter of first word and first letter of last word
- Convert to uppercase
- Example: "John Smith" → "JS", "Admin User" → "AU"

---

## Implementation Checklist

- [ ] Create GET endpoint `/api/agent/conversations/{id}/messages`
- [ ] Implement pagination (page, limit parameters)
- [ ] Join messages with users and clients tables
- [ ] Calculate `is_agent` field based on `user_id` presence
- [ ] Determine `author.type` (customer, agent, or system)
- [ ] Generate initials from author name
- [ ] Order messages by `created_at` (ascending by default)
- [ ] Return empty array for attachments (future implementation)
- [ ] Handle conversation not found (404)
- [ ] Handle invalid parameters (400)
- [ ] Add authentication check
- [ ] Return data in specified JSON format
- [ ] Test with existing conversation data

---

## Frontend Integration

Once implemented, the frontend will:
1. Call this endpoint when a conversation is selected
2. Display messages in chronological order
3. Show different UI for customer vs agent messages
4. Display timestamps and author information
5. Handle attachments when available

**Frontend Service Method:**
```typescript
async getConversationMessages(conversationId: number, page = 1, limit = 50): Promise<MessagesResponse> {
  const endpoint = `${this.baseURL}/api/agent/conversations/${conversationId}/messages?page=${page}&limit=${limit}`

  const response = await fetch(endpoint, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  const result = await response.json()
  return result.data
}
```

---

## Priority

**High Priority** - This endpoint is essential for the conversation detail view in the agent dashboard. Without it, agents cannot view the full conversation history.

---

## Testing

### Test Cases
1. ✅ Retrieve messages for existing conversation
2. ✅ Pagination with different page/limit values
3. ✅ Order by ascending (oldest first)
4. ✅ Order by descending (newest first)
5. ✅ Handle conversation with no messages
6. ✅ Handle conversation not found (404)
7. ✅ Handle invalid conversation ID
8. ✅ Mix of customer and agent messages
9. ✅ System messages display correctly
10. ✅ Author information populated correctly

---

## Notes

1. Messages should be returned in chronological order (oldest first) by default
2. The `last_message_preview` in the conversation list comes from the most recent message body (truncated to ~100 characters)
3. System messages (`is_system_message = true`) should have author type "system"
4. Attachments field included for future implementation but currently returns empty array
5. Consider adding WebSocket support for real-time message updates in future iterations
