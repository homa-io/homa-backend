# CORS Configuration Required for Production

## Issue

The production frontend at `https://dashboard.getevo.dev` is being blocked by CORS policy when making API requests to `https://api.getevo.dev`.

**Error:**
```
Access to fetch at 'https://api.getevo.dev/api/agent/conversations/search?page=1&limit=50&sort_order=desc'
from origin 'https://dashboard.getevo.dev' has been blocked by CORS policy:
No 'Access-Control-Allow-Origin' header is present on the requested resource.
```

## Root Cause

The backend API server does not have CORS (Cross-Origin Resource Sharing) middleware configured to allow requests from the frontend domain.

## Solution

Add CORS middleware to the EVO/Fiber application to allow cross-origin requests from the dashboard domain.

### Implementation Steps

#### 1. Install Fiber CORS Middleware

The project uses Fiber framework, which has a built-in CORS middleware package.

```bash
go get github.com/gofiber/fiber/v2/middleware/cors
```

#### 2. Update `apps/system/app.go`

Add CORS middleware in the `Register()` function:

```go
package system

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/getevo/restify"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/cors"  // ADD THIS
	"strings"
	"time"
)

var StartupTime = time.Now()
var BasePath = ""

type App struct {
}

func (a App) Register() error {
	var logLevel = settings.Get("APP.LOG_LEVEL", "info").String()
	switch strings.ToLower(logLevel) {
	case "debug", "dev", "development":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn", "warning":
		log.SetLevel(log.WarningLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "critical", "crit":
		log.SetLevel(log.CriticalLevel)
	default:
		log.SetLevel(log.WarningLevel)
	}

	var app = evo.GetFiber()

	// ADD CORS MIDDLEWARE
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://dashboard.getevo.dev,http://localhost:3000",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Requested-With",
		AllowCredentials: true,
		ExposeHeaders: "Content-Length,Content-Type",
		MaxAge: 86400, // 24 hours
	}))

	if settings.Get("APP.LOG_REQUESTS").Bool() {
		// Enable request logging
		app.Use(logger.New())
	}

	restify.SetPrefix("/api/restify")

	return nil
}

// ... rest of the code
```

#### 3. Alternative: Configuration-Based Approach

For better flexibility, add CORS configuration to `config.yml`:

```yaml
# ... existing config ...

CORS:
  ENABLED: true
  ALLOW_ORIGINS:
    - "https://dashboard.getevo.dev"
    - "http://localhost:3000"
  ALLOW_METHODS:
    - "GET"
    - "POST"
    - "PUT"
    - "PATCH"
    - "DELETE"
    - "OPTIONS"
  ALLOW_HEADERS:
    - "Origin"
    - "Content-Type"
    - "Accept"
    - "Authorization"
    - "X-Requested-With"
  ALLOW_CREDENTIALS: true
  MAX_AGE: 86400
```

Then update `apps/system/app.go` to read from config:

```go
func (a App) Register() error {
	// ... existing log level code ...

	var app = evo.GetFiber()

	// Configure CORS from settings
	if settings.Get("CORS.ENABLED", true).Bool() {
		allowOrigins := settings.Get("CORS.ALLOW_ORIGINS", []string{
			"https://dashboard.getevo.dev",
			"http://localhost:3000",
		}).StringSlice()

		app.Use(cors.New(cors.Config{
			AllowOrigins:     strings.Join(allowOrigins, ","),
			AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
			AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
			AllowCredentials: true,
			MaxAge:           86400,
		}))

		log.Info("CORS enabled for origins: %v", allowOrigins)
	}

	// ... rest of the code
}
```

## Configuration Explanation

### AllowOrigins
- `https://dashboard.getevo.dev` - Production frontend domain
- `http://localhost:3000` - Local development (Next.js default port)

### AllowMethods
All HTTP methods the frontend needs to use:
- `GET` - Fetching conversations, messages, departments, tags
- `POST` - Creating new conversations, sending messages
- `PUT` - Full updates
- `PATCH` - Partial updates (e.g., marking as read)
- `DELETE` - Deleting resources
- `OPTIONS` - Preflight requests (required for CORS)

### AllowHeaders
Headers that the frontend sends with requests:
- `Origin` - Required for CORS
- `Content-Type` - For JSON payloads
- `Accept` - Specifying expected response format
- `Authorization` - For JWT tokens (when auth is implemented)
- `X-Requested-With` - Common for AJAX requests

### AllowCredentials
Set to `true` to allow cookies and authorization headers to be sent cross-origin.

### MaxAge
Cache preflight requests for 24 hours (86400 seconds) to reduce OPTIONS request overhead.

## Testing

After implementing CORS, test with:

### 1. Production Frontend
Visit `https://dashboard.getevo.dev/conversations` and verify:
- Conversations load without CORS errors
- Messages display when selecting a conversation
- No CORS errors in browser console

### 2. Local Development
Visit `http://localhost:3000/conversations` and verify:
- API requests to `http://127.0.0.1:8033` work
- No CORS errors

### 3. Manual cURL Test with Origin Header
```bash
curl -X OPTIONS 'https://api.getevo.dev/api/agent/conversations/search' \
  -H 'Origin: https://dashboard.getevo.dev' \
  -H 'Access-Control-Request-Method: GET' \
  -H 'Access-Control-Request-Headers: Content-Type' \
  -v
```

Expected response headers:
```
Access-Control-Allow-Origin: https://dashboard.getevo.dev
Access-Control-Allow-Methods: GET,POST,PUT,PATCH,DELETE,OPTIONS
Access-Control-Allow-Headers: Origin,Content-Type,Accept,Authorization,X-Requested-With
Access-Control-Allow-Credentials: true
```

## Security Considerations

1. **Specific Origins**: Only allow specific trusted domains, not `*` (wildcard)
2. **Credentials**: Only enable `AllowCredentials` if absolutely necessary
3. **Methods**: Only allow methods actually used by the API
4. **Headers**: Only allow headers that are needed

## Priority

**CRITICAL** - This is blocking all production frontend functionality. The dashboard cannot fetch any data from the API without CORS headers.

## Related Files

- `/home/evo/homa-backend/apps/system/app.go` - Main application setup
- `/home/evo/homa-backend/config.yml` - Configuration file
- `/home/evo/homa-backend/main.go` - Application entry point

## Additional Resources

- [Fiber CORS Middleware Documentation](https://docs.gofiber.io/api/middleware/cors)
- [MDN CORS Guide](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [CORS Security Best Practices](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/11-Client-side_Testing/07-Testing_Cross_Origin_Resource_Sharing)
