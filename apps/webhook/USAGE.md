# Webhook Usage Guide

Quick guide to using webhooks in Homa.

## Quick Start

### 1. Setup Database

```bash
go run main.go --migration-do
```

### 2. Generate a Test Webhook

Visit [webhook.site](https://webhook.site) to get a unique URL, then:

```bash
go run main.go --generate-webhook --url https://webhook.site/YOUR-UNIQUE-ID --send
```

You should see output like:
```
✅ Mock webhook created successfully!
   ID: 1
   Name: Slack Integration - 456
   URL: https://webhook.site/YOUR-UNIQUE-ID
   Secret: aB3dE5fG7hJ9kL2mN4pQ6rS8tU1vW3xY
   Enabled: true

📋 Event Subscriptions:
   All Events: false
   Ticket Created: true
   Ticket Updated: true
   ...

📤 Sending test webhook...
✅ Test webhook sent successfully!

💡 Check your webhook URL to see the received payload
```

### 3. Create a Ticket to Trigger Webhook

```bash
curl -X POST http://localhost:8000/api/client/tickets \
-H "Content-Type: application/json" \
-d '{
  "title": "Test Ticket",
  "client_name": "John Doe",
  "client_email": "john@example.com",
  "status": "new",
  "priority": "medium",
  "message": "This is a test ticket"
}'
```

Check webhook.site - you should see the `ticket.created` event!

## Using the Admin API

### Create Webhook (requires admin auth)

```bash
curl -X POST http://localhost:8000/api/admin/webhooks \
-H "Content-Type: application/json" \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
-d '{
  "name": "Production Webhook",
  "url": "https://your-api.com/webhook",
  "secret": "your-secure-secret-key",
  "enabled": true,
  "description": "Production webhook for ticket events",
  "event_ticket_created": true,
  "event_ticket_updated": true,
  "event_ticket_status_change": true,
  "event_ticket_closed": true
}'
```

### List Webhooks

```bash
curl http://localhost:8000/api/admin/webhooks \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

### Test Webhook

```bash
curl -X POST http://localhost:8000/api/admin/webhooks/1/test \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

### Update Webhook

```bash
curl -X PUT http://localhost:8000/api/admin/webhooks/1 \
-H "Content-Type: application/json" \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
-d '{
  "enabled": false
}'
```

### Delete Webhook

```bash
curl -X DELETE http://localhost:8000/api/admin/webhooks/1 \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

## Webhook Payload Examples

### ticket.created Event

```json
{
  "event": "ticket.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "ticket_id": 123,
    "title": "Login Issue",
    "client_id": "c4ae2903-1127-4229-9e20-3225990af447",
    "department_id": 5,
    "channel_id": "web",
    "status": "new",
    "priority": "medium"
  }
}
```

### ticket.status_changed Event

```json
{
  "event": "ticket.status_changed",
  "timestamp": "2024-01-15T10:35:00Z",
  "data": {
    "ticket_id": 123,
    "old_status": "new",
    "new_status": "in_progress"
  }
}
```

### ticket.closed Event

```json
{
  "event": "ticket.closed",
  "timestamp": "2024-01-15T11:00:00Z",
  "data": {
    "ticket_id": 123,
    "closed_at": "2024-01-15T11:00:00Z"
  }
}
```

## Verifying Webhook Signatures

If you set a secret, all webhooks include an HMAC-SHA256 signature in the `X-Webhook-Signature` header.

### Node.js Example

```javascript
const crypto = require('crypto');
const express = require('express');
const app = express();

app.post('/webhook', express.json(), (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const secret = 'your-secret-key';

  // Calculate expected signature
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(JSON.stringify(req.body));
  const expectedSignature = hmac.digest('hex');

  // Verify signature
  if (signature !== expectedSignature) {
    return res.status(401).send('Invalid signature');
  }

  // Process webhook
  const { event, data } = req.body;
  console.log(`Received ${event}:`, data);

  res.status(200).send('OK');
});

app.listen(3000);
```

### Python Example

```python
import hmac
import hashlib
import json
from flask import Flask, request

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def webhook():
    signature = request.headers.get('X-Webhook-Signature')
    secret = 'your-secret-key'

    # Calculate expected signature
    payload = request.get_data()
    expected_signature = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    # Verify signature
    if signature != expected_signature:
        return 'Invalid signature', 401

    # Process webhook
    data = request.json
    print(f"Received {data['event']}: {data['data']}")

    return 'OK', 200

if __name__ == '__main__':
    app.run(port=3000)
```

## Monitoring Webhook Deliveries

Query the `webhook_deliveries` table to monitor webhook delivery history:

```sql
-- Get recent deliveries
SELECT * FROM webhook_deliveries
ORDER BY created_at DESC
LIMIT 10;

-- Get failed deliveries
SELECT * FROM webhook_deliveries
WHERE success = 0
ORDER BY created_at DESC;

-- Get deliveries for specific webhook
SELECT * FROM webhook_deliveries
WHERE webhook_id = 1
ORDER BY created_at DESC;
```

## Best Practices

1. **Always use HTTPS** for webhook URLs in production
2. **Verify signatures** to ensure webhooks are from Homa
3. **Respond quickly** - acknowledge receipt with 200 status
4. **Process asynchronously** - queue webhook data for processing
5. **Handle retries** - implement idempotency on your side
6. **Monitor failures** - check `webhook_deliveries` table regularly
7. **Rotate secrets** periodically for security
8. **Test thoroughly** - use webhook.site or similar tools

## Troubleshooting

### Webhook not triggering

1. Check webhook is enabled: `enabled = true`
2. Verify event subscription: check `event_*` boolean fields
3. Check webhook deliveries table for errors

### Signature verification fails

1. Ensure you're using the exact payload bytes
2. Verify secret matches what's in database
3. Check Content-Type is `application/json`

### Timeout errors

1. Ensure webhook URL is accessible
2. Check endpoint responds within 30 seconds
3. Return 200 status as quickly as possible

## Support

For issues or questions:
- Check `webhook_deliveries` table for delivery logs
- Use `--generate-webhook --send` to test connectivity
- Review webhook configuration via admin API
