# Webhook Test Server

A local test server to receive and log webhooks from Homa.

## Features

- ✅ Receives webhooks on `http://localhost:9000/webhook`
- ✅ Verifies HMAC-SHA256 signatures
- ✅ Logs all webhooks to JSON files
- ✅ Displays webhooks in console with colors
- ✅ Web UI at `http://localhost:9000`

## Quick Start

### 1. Start the Test Server

```bash
cd _webhook_test
go run server.go
```

You should see:
```
🚀 Webhook Test Server Started
================================
📡 Listening on http://localhost:9000
📝 Webhook endpoint: http://localhost:9000/webhook
📁 Logs directory: _webhook_test/logs
🔑 Test secret: test-secret-key-123
================================

Waiting for webhooks...
```

### 2. Create a Test Webhook in Homa

**Option A: Using Mock Generator**
```bash
# In another terminal
cd ..
go run main.go --generate-webhook --url http://localhost:9000/webhook --send
```

**Option B: Using Admin API** (requires admin token)
```bash
curl -X POST http://localhost:8000/api/admin/webhooks \
-H "Content-Type: application/json" \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
-d '{
  "name": "Test Webhook Server",
  "url": "http://localhost:9000/webhook",
  "secret": "test-secret-key-123",
  "enabled": true,
  "event_ticket_created": true,
  "event_ticket_updated": true,
  "event_ticket_status_change": true,
  "event_ticket_closed": true
}'
```

### 3. Trigger a Webhook Event

Create a test ticket:
```bash
curl -X POST http://localhost:8000/api/client/tickets \
-H "Content-Type: application/json" \
-d '{
  "title": "Test Webhook Ticket",
  "client_name": "Test User",
  "client_email": "test@example.com",
  "status": "new",
  "priority": "medium",
  "message": "This is a test to trigger webhooks"
}'
```

### 4. Check the Results

**Console Output:**
The test server will display:
```
================================================================================
📨 WEBHOOK RECEIVED: ticket.created
================================================================================
⏰ Received At: 2024-01-15T10:30:00Z
📍 Event: ticket.created
🕐 Event Timestamp: 2024-01-15T10:30:00Z
✅ Signature: VERIFIED

📋 Headers:
   X-Webhook-Signature: abc123...
   X-Webhook-Event: ticket.created
   X-Webhook-Id: 1
   User-Agent: Homa-Webhook/1.0

📦 Data:
   {
     "ticket_id": 123,
     "title": "Test Webhook Ticket",
     "client_id": "uuid-here",
     "channel_id": "web",
     "status": "new",
     "priority": "medium"
   }
================================================================================
💾 Saved to: _webhook_test/logs/20240115_103000_ticket.created.json
```

**Log Files:**
Check `_webhook_test/logs/` directory for JSON files:
```
_webhook_test/logs/
├── 20240115_103000_ticket.created.json
├── 20240115_103015_ticket.updated.json
└── 20240115_103030_ticket.status_changed.json
```

## Log File Format

Each webhook is saved as a JSON file with this structure:

```json
{
  "received_at": "2024-01-15T10:30:00Z",
  "event": "ticket.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "ticket_id": 123,
    "title": "Test Webhook Ticket",
    "client_id": "uuid-here",
    "channel_id": "web",
    "status": "new",
    "priority": "medium"
  },
  "headers": {
    "Content-Type": "application/json",
    "User-Agent": "Homa-Webhook/1.0",
    "X-Webhook-Event": "ticket.created",
    "X-Webhook-Id": "1",
    "X-Webhook-Signature": "abc123..."
  },
  "signature": "abc123...",
  "signature_verified": true,
  "raw_payload": "{...}"
}
```

## Testing Different Events

### Test All Events

1. **Create Ticket** → triggers `ticket.created`
2. **Update Ticket Status** → triggers `ticket.status_changed` + `ticket.updated`
3. **Close Ticket** → triggers `ticket.closed` + `ticket.status_changed`

### Test Webhook

Use the test endpoint:
```bash
curl -X POST http://localhost:8000/api/admin/webhooks/1/test \
-H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

This sends a `webhook.test` event.

## Configuration

### Change Port

Edit `server.go` line 40:
```go
port := ":9000"  // Change to your desired port
```

### Change Secret

Edit `server.go` line 28:
```go
TestSecret = "test-secret-key-123"  // Change to match your webhook
```

Make sure it matches the secret in your Homa webhook configuration!

## Troubleshooting

### Webhook not received

1. Check test server is running on port 9000
2. Verify webhook URL is `http://localhost:9000/webhook`
3. Check webhook is enabled in Homa
4. Verify event subscription matches the triggered event

### Signature verification fails

1. Ensure secret matches: `test-secret-key-123`
2. Check webhook configuration has the correct secret
3. Verify Homa is sending the signature header

### No logs created

1. Check `_webhook_test/logs` directory exists
2. Verify write permissions
3. Check console for error messages

## Web Interface

Visit `http://localhost:9000` in your browser to see the test server status page with:
- Server status
- Webhook endpoint URL
- Test secret
- Instructions

## Cleanup

To remove all logs:
```bash
rm -rf _webhook_test/logs/*.json
```

Or start fresh:
```bash
rm -rf _webhook_test/logs
```

The logs directory will be recreated when the server starts.
