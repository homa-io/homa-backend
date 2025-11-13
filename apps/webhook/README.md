# Webhook App

Webhook system for Homa support platform that allows external systems to subscribe to events.

## Features

- **Webhook Management**: Full CRUD operations for webhook subscriptions
- **Event Broadcasting**: Automatic webhook notifications for system events
- **HMAC Signatures**: Secure webhook delivery with optional HMAC-SHA256 signatures
- **Delivery Tracking**: Complete logging of webhook delivery attempts
- **Event Filtering**: Subscribe to specific events or all events with wildcard `*`

## Models

### Webhook
- `id` - Unique identifier
- `name` - Webhook name
- `url` - Target URL for webhook delivery
- `events` - Array of event types to subscribe to
- `secret` - Optional secret for HMAC signature generation
- `enabled` - Enable/disable webhook
- `description` - Optional description
- `created_at`, `updated_at` - Timestamps

### WebhookDelivery
- `id` - Unique identifier
- `webhook_id` - Reference to webhook
- `event` - Event type that triggered delivery
- `success` - Whether delivery succeeded
- `response` - Error message or response details
- `created_at` - Delivery timestamp

## Available Events

Each webhook has boolean flags for event subscriptions:

| Event Field | Event Name | Description |
|------------|------------|-------------|
| `event_all` | All Events | Subscribe to all events (wildcard) |
| `event_ticket_created` | `ticket.created` | New ticket created |
| `event_ticket_updated` | `ticket.updated` | Ticket updated |
| `event_ticket_status_change` | `ticket.status_changed` | Ticket status changed |
| `event_ticket_closed` | `ticket.closed` | Ticket closed |
| `event_ticket_assigned` | `ticket.assigned` | Ticket assigned to user/department |
| `event_message_created` | `message.created` | New message added |
| `event_client_created` | `client.created` | New client created |
| `event_client_updated` | `client.updated` | Client updated |
| `event_user_created` | `user.created` | New user created |
| `event_user_updated` | `user.updated` | User updated |

## Admin APIs

**Note**: All webhook endpoints require admin authentication. The Webhook model uses `restify.API` which automatically provides REST endpoints.

### Create Webhook
```http
POST /api/webhooks
Content-Type: application/json

{
  "name": "My Integration",
  "url": "https://example.com/webhook",
  "secret": "optional-secret-key",
  "enabled": true,
  "description": "Webhook for ticket notifications",
  "event_ticket_created": true,
  "event_ticket_updated": true,
  "event_ticket_status_change": true
}
```

### List Webhooks
```http
GET /api/webhooks?page=1&limit=10
```

Supports filtering and search through restify query parameters.

### Get Webhook
```http
GET /api/webhooks/:id
```

### Update Webhook
```http
PUT /api/webhooks/:id
Content-Type: application/json

{
  "name": "Updated Name",
  "enabled": false,
  "event_ticket_closed": true
}
```

### Delete Webhook
```http
DELETE /api/webhooks/:id
```

### Test Webhook (Custom Endpoint)
```http
POST /api/webhooks/:id/test
```

Sends a test payload to verify webhook connectivity.

## Webhook Payload Structure

All webhooks receive a JSON payload with this structure:

```json
{
  "event": "ticket.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "ticket_id": 123,
    "title": "Support Request",
    "client_id": "uuid-here",
    "department_id": 5,
    "channel_id": "web",
    "status": "new",
    "priority": "medium"
  }
}
```

## Webhook Headers

Webhooks are delivered with the following headers:

- `Content-Type: application/json`
- `User-Agent: Homa-Webhook/1.0`
- `X-Webhook-Event: ticket.created` - Event type
- `X-Webhook-ID: 123` - Webhook ID
- `X-Webhook-Signature: abc123...` - HMAC-SHA256 signature (if secret configured)

## HMAC Signature Verification

If a secret is configured, webhooks include an HMAC-SHA256 signature in the `X-Webhook-Signature` header.

### Verify Signature (Node.js example)
```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(JSON.stringify(payload));
  const expectedSignature = hmac.digest('hex');

  return signature === expectedSignature;
}
```

### Verify Signature (Python example)
```python
import hmac
import hashlib
import json

def verify_webhook(payload, signature, secret):
    expected_signature = hmac.new(
        secret.encode(),
        json.dumps(payload).encode(),
        hashlib.sha256
    ).hexdigest()

    return signature == expected_signature
```

## Integration Examples

### Ticket Events
Webhooks are automatically triggered for:
- Ticket creation → `ticket.created`
- Ticket updates → `ticket.updated`
- Status changes → `ticket.status_changed`
- Ticket closure → `ticket.closed`

### Usage in Code
```go
import "github.com/getevo/homa/apps/webhook"

// Broadcast to all subscribed webhooks
webhook.BroadcastWebhook("ticket.created", map[string]any{
    "ticket_id": ticket.ID,
    "title": ticket.Title,
    "status": ticket.Status,
})

// Send to specific webhook
webhook.SendWebhook(&webhookInstance, "custom.event", data)
```

## Security Considerations

1. **HTTPS Required**: Always use HTTPS URLs for production webhooks
2. **Secret Keys**: Use strong, randomly generated secrets
3. **Signature Verification**: Always verify HMAC signatures in production
4. **Rate Limiting**: Implement rate limiting on webhook endpoints
5. **Timeout**: Webhook delivery has a 30-second timeout
6. **Retry Logic**: Currently no automatic retries (implement on receiver side)

## CLI Commands

### Generate Mock Webhook

Create a mock webhook for testing:

```bash
# Generate mock webhook
go run main.go --generate-webhook

# Generate with custom URL
go run main.go --generate-webhook --url https://webhook.site/your-unique-id

# Generate and send test webhook immediately
go run main.go --generate-webhook --send

# Generate with custom URL and send test
go run main.go --generate-webhook --url https://webhook.site/your-unique-id --send
```

The mock generator will:
- Create a webhook with random name and configuration
- Randomly subscribe to various events
- Generate a secure random secret key
- Optionally send a test payload to verify connectivity

### Testing Webhooks

1. Visit [webhook.site](https://webhook.site) to get a unique URL
2. Generate a mock webhook with that URL
3. Use `--send` flag to immediately test the webhook
4. Check webhook.site to see the received payload

## Database Schema

The webhook system adds two new tables:
- `webhooks` - Webhook subscriptions
- `webhook_deliveries` - Delivery attempt logs

Run migration to create tables:
```bash
go run main.go --migration-do
```

## Admin Authentication

All webhook management endpoints require admin authentication. The middleware checks:
- User is authenticated
- User type is `administrator`

Unauthorized requests receive `401 Unauthorized` response.
