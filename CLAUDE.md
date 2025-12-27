# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **Homa**, an intelligent support system built with the Evo v2 framework. The application follows a modular architecture with apps as self-contained modules that register themselves with the framework.

## Essential Commands

### Running as a Systemd Service (Production)

This project runs as a systemd service. Use these commands to manage it:

```bash
# Start the service
sudo systemctl start homa-backend.service

# Stop the service
sudo systemctl stop homa-backend.service

# Restart the service (rebuilds and restarts)
sudo systemctl restart homa-backend.service

# Check status
sudo systemctl status homa-backend.service

# View logs
sudo journalctl -u homa-backend.service -f

# View recent logs
sudo journalctl -u homa-backend.service -n 100 --no-pager
```

The service automatically:
1. Runs `go build -o homa-backend main.go` to compile the application
2. Runs `./homa-backend -c /home/evo/config/homa-backend/config.yml`
3. Restarts on failure
4. Starts on system boot
5. Listens on port 8033

**Related Service**: The dashboard frontend runs as `homa-dashboard.service` on port 3000.

### Running Manually (Development)
```bash
# Run the application (starts HTTP server on port 8033)
go run main.go -c /home/evo/config/homa-backend/config.yml

# Run with database migration
go run main.go -c /home/evo/config/homa-backend/config.yml --migration-do

# Build the application
go build -o homa-backend main.go
./homa-backend -c /home/evo/config/homa-backend/config.yml
```

### Development Commands
```bash
# Install dependencies
go mod tidy

# Format code
go fmt ./...

# Download dependencies
go mod download
```

## Architecture Overview

### Framework Integration
- Built on **Evo v2** framework (`github.com/getevo/evo/v2`)
- Uses **Restify** for automatic REST API generation (`github.com/getevo/restify`)
- GORM for database operations with automatic model registration
- Configuration-driven database and HTTP server setup

### Application Structure
The project uses a **modular app architecture** where each app is self-contained:

```
apps/
├── models/     # Database models and schema management
├── system/     # Health checks and system endpoints
├── auth/       # Authentication, OAuth, and user management
├── ticket/     # Ticket management and custom attributes processing
└── [future]/   # Additional feature modules
```

Each app implements the standard interface:
- `Register()` - Initialize resources
- `Router()` - Register HTTP routes  
- `WhenReady()` - Post-initialization tasks
- `Name()` - App identifier

### Data Model Architecture

**Core Entities:**
- **Users** (UUID-based): Agents and administrators with OAuth support
- **Clients** (UUID-based): External users creating tickets  
- **Tickets**: Core business entities linked to clients, channels, and departments with secret-based client access
- **Channels**: Communication channels (WhatsApp, Slack, etc.) with string IDs
- **Departments**: Organizational units for ticket routing
- **Messages**: Communication within tickets
- **CustomAttributes**: Configurable form fields for dynamic ticket/client data collection

**Key Relationships:**
- Users ↔ Departments (many-to-many via junction table)
- Tickets → Channel (each ticket has one channel + external_id for integration)
- Tickets → Client (UUID foreign key)
- Tickets ↔ Tags (many-to-many)
- Tickets ↔ Users (assignments via junction table)

**Important Patterns:**
- All models embed `restify.API` for automatic REST endpoints
- UUID primary keys for Users and Clients (not auto-incrementing)
- String primary keys for Channels (e.g., "whatsapp", "slack")
- JSONB fields for flexible data (`configuration`, `custom_fields`, `data`)
- Enum constants with database constraints for `user.type` and `ticket.status`

### Database Migration
- Models are auto-registered via `db.UseModel()` in `apps/models/app.go`
- Migration triggered with `--migration-do` flag
- Uses GORM auto-migration features

### Configuration
- YAML-based configuration (`config.yml`, `config.dev.yml`)
- Database: MySQL by default (configurable)
- HTTP: Fiber-based server on port 8000

### Custom Attributes System

The **CustomAttribute** model enables dynamic form field creation for tickets and clients, allowing administrators to define additional data collection points without code changes.

**Key Features:**
- **Scope-based**: Attributes can be defined for either "client" or "ticket" entities
- **Type-safe**: Supports int, float, date, and string data types with automatic casting
- **Validation**: Configurable validation rules using Evo validation syntax
- **Visibility Control**: Three levels - everyone, administrator, or hidden
- **JSON Storage**: Values stored as JSON in target entity's custom_fields/data columns

**Usage Pattern:**
1. **Define CustomAttribute**: Create attribute definition via admin interface
2. **Use in APIs**: Pass custom attribute values in create/update requests
3. **Automatic Processing**: System validates, casts, and stores values as JSON

**Example Flow:**
```json
// 1. CustomAttribute definition
{
  "scope": "ticket",
  "name": "priority_level", 
  "data_type": "int",
  "validation": "min:1,max:5",
  "title": "Priority Level",
  "visibility": "everyone"
}

// 2. Ticket creation with custom attribute
POST /api/tickets
{
  "title": "Sample Ticket",
  "client_id": "uuid-here",
  "channel_id": "web",
  "status": "new",
  "priority": "medium",
  "parameters": {
    "priority_level": 3  // Custom attribute value
  }
}

// 3. Stored in ticket.custom_fields as JSON
{
  "priority_level": 3
}
```

**Technical Implementation:**
- Located in `apps/models/custom_attributes.go`
- Composite primary key: scope + name
- Validation via `apps/ticket/functions.go` processing logic
- API integration through `apps/ticket/controller.go`
- Automatic type casting and JSON serialization

### Ticket Secret System

The **Ticket Secret** field enables secure client access to tickets without requiring full authentication. Clients can add messages to tickets using only the ticket ID and secret.

**Key Features:**
- **Auto-Generated**: Secret is automatically generated during ticket creation (32-character hex)
- **Client Access**: Allows clients to add messages without agent authentication
- **Security**: Secret is hidden from JSON responses except during creation
- **Unique**: Each ticket gets a unique random secret for access control

**Usage Pattern:**
1. **Ticket Creation**: Secret is auto-generated and returned in response
2. **Client Access**: Use ticket ID + secret to add messages
3. **Message Creation**: System validates secret before allowing message creation

**Example Flow:**
```json
// 1. Ticket creation (secret auto-generated)
POST /api/tickets
{
  "title": "Support Request",
  "client_id": "uuid-here",
  "channel_id": "web",
  "status": "new",
  "priority": "medium",
  "message": "Initial issue description"
}

// Response includes auto-generated secret
{
  "id": 123,
  "title": "Support Request",
  "secret": "a1b2c3d4e5f6789012345678901234ef",
  ...
}

// 2. Client adds message using secret
POST /api/tickets/123/messages
{
  "secret": "a1b2c3d4e5f6789012345678901234ef",
  "message": "Additional information from client"
}
```

**Security Notes:**
- Secret is stored as plain text for validation purposes
- Secret field is excluded from all JSON responses for security
- Exactly 32 character requirement for consistent security
- No authentication tokens required for client message API

**Available Client Message APIs:**
- `POST /api/tickets/:id/messages` - Secret in request body
- `PUT /api/tickets/:ticket_id/:secret/message` - Secret in URL path

## Development Guidelines

Refer to `docs/guideline.md` for comprehensive technical standards including:
- Model development with proper GORM tags and relationships
- API handler patterns and response formats
- File organization and naming conventions
- Enum usage and constants

## Key Files for Understanding

- `main.go` - Application entry point and app registration
- `docs/models.md` - Complete database schema specification
- `docs/guideline.md` - Technical development standards
- `apps/models/app.go` - Model registration and migration logic
- `config.yml` - Application configuration
- docs/guideline.md contains all coding standards which should be followed. if i pass new coding standard update or add docs/guideline.md