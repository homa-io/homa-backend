#!/bin/bash

# Webhook Test Script
# This script automates the full webhook testing process

set -e

echo "🧪 Homa Webhook Test Script"
echo "============================"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if test server is running
if ! curl -s http://localhost:9000 > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Test server not running on port 9000${NC}"
    echo -e "${BLUE}Starting test server in background...${NC}"
    cd _webhook_test
    go run server.go > server.log 2>&1 &
    SERVER_PID=$!
    cd ..
    sleep 2
    echo -e "${GREEN}✅ Test server started (PID: $SERVER_PID)${NC}"
else
    echo -e "${GREEN}✅ Test server already running${NC}"
fi

echo ""
echo -e "${BLUE}📡 Creating test webhook...${NC}"

# Create test webhook using mock generator
go run main.go --generate-webhook --url http://localhost:9000/webhook --send

echo ""
echo -e "${BLUE}🎫 Creating test ticket to trigger webhook...${NC}"

# Create a test ticket
TICKET_RESPONSE=$(curl -s -X POST http://localhost:8000/api/client/tickets \
-H "Content-Type: application/json" \
-d '{
  "title": "Webhook Test Ticket",
  "client_name": "Test User",
  "client_email": "test@webhook.local",
  "status": "new",
  "priority": "medium",
  "message": "This ticket was created by the automated test script"
}')

echo "$TICKET_RESPONSE" | jq '.' 2>/dev/null || echo "$TICKET_RESPONSE"

echo ""
echo -e "${GREEN}✅ Test ticket created!${NC}"
echo ""
echo -e "${BLUE}📊 Checking webhook logs...${NC}"
sleep 1

# List log files
if [ -d "_webhook_test/logs" ]; then
    LOG_COUNT=$(ls -1 _webhook_test/logs/*.json 2>/dev/null | wc -l)
    if [ "$LOG_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✅ Found $LOG_COUNT webhook log(s)${NC}"
        echo ""
        echo "📄 Recent webhook logs:"
        ls -lt _webhook_test/logs/*.json | head -5 | while read line; do
            echo "   $line"
        done

        echo ""
        echo "📋 Latest webhook content:"
        LATEST_LOG=$(ls -t _webhook_test/logs/*.json | head -1)
        cat "$LATEST_LOG" | jq '.' 2>/dev/null || cat "$LATEST_LOG"
    else
        echo -e "${YELLOW}⚠️  No webhook logs found yet${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Logs directory not found${NC}"
fi

echo ""
echo "================================"
echo -e "${GREEN}✅ Test Complete!${NC}"
echo ""
echo "📁 Check logs: _webhook_test/logs/"
echo "🖥️  View server console: tail -f _webhook_test/server.log"
echo "🌐 Open browser: http://localhost:9000"
echo ""

if [ ! -z "$SERVER_PID" ]; then
    echo "🛑 To stop test server: kill $SERVER_PID"
fi
