# 🚀 Webhook Test - Quick Start

## One-Command Test (Automated)

### Windows:
```bash
_webhook_test\test.bat
```

### Linux/Mac:
```bash
chmod +x _webhook_test/test.sh
_webhook_test/test.sh
```

This will:
1. ✅ Start the test server on port 9000
2. ✅ Create a test webhook
3. ✅ Create a test ticket
4. ✅ Show the received webhook logs

---

## Manual Test (Step by Step)

### Step 1: Start Test Server

```bash
cd _webhook_test
go run server.go
```

**Expected Output:**
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

### Step 2: Create Test Webhook

Open a **new terminal** and run:

```bash
go run main.go --generate-webhook --url http://localhost:9000/webhook --send
```

**Expected Output:**
```
✅ Mock webhook created successfully!
   ID: 1
   Name: Slack Integration - 456
   URL: http://localhost:9000/webhook
   Secret: test-secret-key-123
   ...

📤 Sending test webhook...
✅ Test webhook sent successfully!
```

**In the test server terminal, you should see:**
```
================================================================================
📨 WEBHOOK RECEIVED: webhook.test
================================================================================
⏰ Received At: 2024-01-15T10:30:00Z
📍 Event: webhook.test
✅ Signature: VERIFIED
...
💾 Saved to: _webhook_test/logs/20240115_103000_webhook.test.json
```

### Step 3: Create Test Ticket

Trigger a real webhook event:

```bash
curl -X POST http://localhost:8000/api/client/tickets \
-H "Content-Type: application/json" \
-d '{
  "title": "Test Webhook Delivery",
  "client_name": "John Doe",
  "client_email": "john@example.com",
  "status": "new",
  "priority": "medium",
  "message": "Testing webhook system"
}'
```

**In the test server, you'll see:**
```
================================================================================
📨 WEBHOOK RECEIVED: ticket.created
================================================================================
⏰ Received At: 2024-01-15T10:30:15Z
📍 Event: ticket.created
✅ Signature: VERIFIED

📦 Data:
   {
     "ticket_id": 1,
     "title": "Test Webhook Delivery",
     "client_id": "...",
     "status": "new",
     "priority": "medium"
   }
💾 Saved to: _webhook_test/logs/20240115_103015_ticket.created.json
```

### Step 4: Check the Logs

```bash
# List all webhook logs
ls -la _webhook_test/logs/

# View latest log
cat _webhook_test/logs/*.json | tail -1 | jq '.'

# View specific event logs
cat _webhook_test/logs/*ticket.created.json | jq '.'
```

---

## What Gets Logged

Each webhook is saved with:

✅ **Complete payload** - Full event data
✅ **All headers** - Including signature
✅ **Signature verification** - Pass/Fail status
✅ **Timestamp** - When received
✅ **Raw payload** - Original JSON

**File naming:** `YYYYMMDD_HHMMSS_eventname.json`

Example: `20240115_103000_ticket.created.json`

---

## Test Different Events

### 1. Ticket Created
```bash
curl -X POST http://localhost:8000/api/client/tickets \
-H "Content-Type: application/json" \
-d '{"title":"New Ticket","client_name":"User","status":"new","priority":"medium"}'
```
**Triggers:** `ticket.created`

### 2. Ticket Status Change
```bash
# First create a ticket, then update its status
curl -X PUT http://localhost:8000/api/admin/tickets/1/status \
-H "Authorization: Bearer YOUR_TOKEN" \
-H "Content-Type: application/json" \
-d '{"status":"in_progress"}'
```
**Triggers:** `ticket.status_changed` + `ticket.updated`

### 3. Close Ticket
```bash
curl -X PUT http://localhost:8000/api/admin/tickets/1/status \
-H "Authorization: Bearer YOUR_TOKEN" \
-H "Content-Type: application/json" \
-d '{"status":"closed"}'
```
**Triggers:** `ticket.closed` + `ticket.status_changed` + `ticket.updated`

---

## Verify Signature

All webhooks include HMAC-SHA256 signature in `X-Webhook-Signature` header.

**Test Secret:** `test-secret-key-123`

The test server automatically verifies signatures and shows:
- ✅ **VERIFIED** - Signature is valid
- ❌ **FAILED** - Signature mismatch
- ⚠️ **NOT PROVIDED** - No signature sent

---

## View in Browser

Open: **http://localhost:9000**

You'll see a web interface with:
- ✅ Server status
- 📋 Endpoint URL
- 🔑 Test secret
- 📖 Instructions

---

## Troubleshooting

### No webhooks received?

1. Check test server is running: `curl http://localhost:9000`
2. Verify webhook exists: Check database or use API
3. Check webhook is enabled: `enabled = true`
4. Verify event subscriptions: Check event flags

### Signature fails?

1. Ensure webhook secret is: `test-secret-key-123`
2. Update webhook:
   ```bash
   curl -X PUT http://localhost:8000/api/admin/webhooks/1 \
   -H "Content-Type: application/json" \
   -d '{"secret":"test-secret-key-123"}'
   ```

### Port already in use?

Change port in `_webhook_test/server.go` line 40:
```go
port := ":9001"  // Use different port
```

---

## Clean Up

**Remove all logs:**
```bash
rm -rf _webhook_test/logs/*.json
```

**Stop test server:**
```bash
# Find process
ps aux | grep server.go

# Kill it
kill <PID>
```

---

## Success Checklist

- [ ] Test server running on port 9000
- [ ] Webhook created with correct URL
- [ ] Secret matches: `test-secret-key-123`
- [ ] Event subscriptions enabled
- [ ] Ticket created successfully
- [ ] Webhook received and logged
- [ ] Signature verified ✅
- [ ] Log files created in `_webhook_test/logs/`

🎉 **All Done!** Your webhook system is working correctly!
