# Device Information Fields - Backend Update Required

## Overview
The frontend now displays device information (IP address, Browser, Operating System) in the conversation detail view. The backend needs to be updated to include these fields in the API response.

## Database Schema
The `conversations` table already has these fields:
- `ip` - VARCHAR - IP address of the user
- `browser` - VARCHAR - Browser information (e.g., "Chrome 120.0", "Firefox 121.0")
- `operating_system` - VARCHAR - Operating system (e.g., "Windows 11", "macOS 14.0", "iOS 17.2")

## Required Changes

### Update GET /api/agent/conversations/search Response

Add the following fields to each conversation object in the response:

```json
{
  "id": 32,
  "conversation_number": "CONV-32",
  "title": "Unable to login to dashboard",
  // ... existing fields ...
  "ip": "192.168.1.100",
  "browser": "Chrome 120.0",
  "operating_system": "Windows 11"
}
```

### Implementation

Update the SQL query in the conversations search endpoint to include these fields:

```sql
SELECT
  t.id,
  t.conversation_number,
  t.title,
  t.status,
  t.priority,
  -- ... other existing fields ...
  t.ip,
  t.browser,
  t.operating_system,
  -- ... rest of the fields ...
FROM conversations t
-- ... rest of the query ...
```

## Frontend Integration

The frontend is already configured to display this information:
- `/home/evo/homa-dashboard/src/types/conversation.types.ts` - Types updated
- `/home/evo/homa-dashboard/app/conversations/ConversationsContent.tsx` - UI updated to display device info

Device information is displayed in the right sidebar under "Device Information" section:
- IP Address (with country flag badge)
- Operating System (with appropriate icon)
- Browser (with globe icon)

## Example Mock Data

From the database mock_data.sql:

```sql
INSERT INTO conversations (..., ip, browser, operating_system, ...) VALUES
('Unable to login to dashboard', ..., '192.168.1.100', 'Chrome 120.0', 'Windows 11', ...),
('Pricing inquiry for Enterprise plan', ..., '85.123.45.67', 'Firefox 121.0', 'macOS 14.0', ...),
('Payment failed - Need assistance', ..., '114.245.67.89', 'Safari 17.0', 'iOS 17.2', ...);
```

## Field Requirements

- All three fields should be nullable (can be `null` if information is not available)
- Default to `null` instead of empty strings
- Frontend will display "N/A" when fields are null

## Priority

**Medium** - The frontend is ready and will work with or without this data, but will show "N/A" until the backend is updated.

## Testing

After implementation, verify:
1. GET `/api/agent/conversations/search` returns ip, browser, operating_system fields
2. Fields are properly populated from the database
3. Null values are handled correctly
4. Frontend displays the information correctly in the Device Information section
