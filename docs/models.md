
# Support System Database Models

This document outlines the database schema for a support ticket system.

## Tables

### `users`

Stores information about internal users (agents and administrators).

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the user.                                                       |
| `name`           | `VARCHAR(255)`| The full name of the user.                                                                              |
| `email`          | `VARCHAR(255)`| The user's email address. Must be unique.                                                              |
| `password_hash`  | `VARCHAR(255)`| Hashed password for authentication. Can be null if using OAuth.                                         |
| `type`           | `VARCHAR(50)` | Type of the user. Enum: `administrator`, `agent`.                                                      |
| `created_at`     | `TIMESTAMP`   | Timestamp of when the user was created. Defaults to the current time.                                   |
| `updated_at`     | `TIMESTAMP`   | Timestamp of the last update to the user's record.                                                     |

### `user_oauth_accounts`

Stores OAuth information for users, allowing them to log in via third-party providers.

| Column Name        | Data Type     | Description                                                                                             |
| ------------------ | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`               | `SERIAL`      | **Primary Key** - Unique identifier for the OAuth record.                                               |
| `user_id`          | `INTEGER`     | **Foreign Key** to `users.id`.                                                                          |
| `provider`         | `VARCHAR(50)` | The OAuth provider (e.g., 'google', 'github', 'slack').                                              |
| `provider_user_id` | `VARCHAR(255)`| The user's unique ID as provided by the OAuth provider.                                                |
| `created_at`       | `TIMESTAMP`   | Timestamp of when the record was created. Defaults to the current time.                                 |
| **Unique**         |               | A composite unique key on (`provider`, `provider_user_id`).                                             |


### `clients`

Stores information about the clients who create tickets.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the client.                                                     |
| `name`           | `VARCHAR(255)`| The full name of the client.                                                                            |
| `data`           | `JSONB`       | A flexible field to store any additional custom data about the client as a JSON object.                 |
| `created_at`     | `TIMESTAMP`   | Timestamp of when the client was created. Defaults to the current time.                                 |
| `updated_at`     | `TIMESTAMP`   | Timestamp of the last update to the client's record.                                                   |

### `client_external_ids`

Stores multiple external identifiers for a single client.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the external ID record.                                         |
| `client_id`      | `INTEGER`     | **Foreign Key** to `clients.id`.                                                                        |
| `type`           | `VARCHAR(50)` | The type of external ID (e.g., 'email', 'phone', 'whatsapp', 'slack').                            |
| `value`          | `VARCHAR(255)`| The value of the external ID. The combination of `type` and `value` must be unique.                     |
| `created_at`     | `TIMESTAMP`   | Timestamp of when the record was created. Defaults to the current time.                                 |

### `departments`

Stores information about support departments.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the department.                                                 |
| `name`           | `VARCHAR(255)`| The name of the department (e.g., 'Sales', 'Technical Support'). Must be unique.                      |
| `description`    | `TEXT`        | A brief description of the department's responsibilities.                                              |
| `created_at`     | `TIMESTAMP`   | Timestamp of when the department was created. Defaults to the current time.                             |

### `tickets`

The core table for storing support tickets.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the ticket.                                                     |
| `title`          | `VARCHAR(255)`| A concise title for the ticket.                                                                         |
| `client_id`      | `INTEGER`     | **Foreign Key** to `clients.id`. The client who created the ticket.                                     |
| `department_id`  | `INTEGER`     | **Foreign Key** to `departments.id`. The department this ticket is assigned to. Can be nullable.        |
| `status`         | `VARCHAR(50)` | The current status of the ticket. Enum: `open`, `waiting_for_user`, `waiting_for_agent`, `closed`, `unresolved`. |
| `priority`       | `VARCHAR(50)` | The priority level of the ticket (e.g., 'low', 'medium', 'high', 'urgent').                           |
| `custom_fields`  | `JSONB`       | A flexible field to store any additional custom data as a JSON object.                                  |
| `created_at`     | `TIMESTAMP`   | Timestamp of when the ticket was created. Defaults to the current time.                                 |
| `updated_at`     | `TIMESTAMP`   | Timestamp of the last update to the ticket.                                                             |
| `closed_at`      | `TIMESTAMP`   | Timestamp of when the ticket was closed. Nullable.                                                      |

### `messages`

Stores all communication related to a ticket.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the message.                                                    |
| `ticket_id`      | `INTEGER`     | **Foreign Key** to `tickets.id`. The ticket this message belongs to.                                    |
| `user_id`        | `INTEGER`     | **Foreign Key** to `users.id`. The user who sent the message. Null if sent by a client or system.         |
| `client_id`      | `INTEGER`     | **Foreign Key** to `clients.id`. The client who sent the message. Null if sent by a user or system.       |
| `body`           | `TEXT`        | The content of the message.                                                                             |
| `is_system_message` | `BOOLEAN`  | `true` if the message is automated or from the system, `false` otherwise. Defaults to `false`.          |
| `created_at`     | `TIMESTAMP`   | Timestamp of when the message was sent. Defaults to the current time.                                   |
| *Constraint*     |               | `CHECK (user_id IS NOT NULL OR client_id IS NOT NULL OR is_system_message = true)`                      |


### `tags`

A table to store tags that can be applied to tickets.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `id`             | `SERIAL`      | **Primary Key** - Unique identifier for the tag.                                                        |
| `name`           | `VARCHAR(100)`| The name of the tag (e.g., 'billing', 'bug', 'feature-request'). Must be unique.                     |

---

## Junction Tables (Many-to-Many Relationships)

### `ticket_assignments`

Links tickets to the users assigned to them.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `ticket_id`      | `INTEGER`     | **Foreign Key** to `tickets.id`.                                                                        |
| `user_id`        | `INTEGER`     | **Foreign Key** to `users.id`.                                                                          |
| **Primary Key**  |               | A composite primary key on (`ticket_id`, `user_id`).                                                    |

### `ticket_tags`

Links tickets to their associated tags.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `ticket_id`      | `INTEGER`     | **Foreign Key** to `tickets.id`.                                                                        |
| `tag_id`         | `INTEGER`     | **Foreign Key** to `tags.id`.                                                                           |
| **Primary Key**  |               | A composite primary key on (`ticket_id`, `tag_id`).                                                     |

### `user_departments`

Links users with the 'agent' type to the departments they have access to.

| Column Name      | Data Type     | Description                                                                                             |
| ---------------- | ------------- | ------------------------------------------------------------------------------------------------------- |
| `user_id`        | `INTEGER`     | **Foreign Key** to `users.id`.                                                                          |
| `department_id`  | `INTEGER`     | **Foreign Key** to `departments.id`.                                                                    |
| **Primary Key**  |               | A composite primary key on (`user_id`, `department_id`).                                                |
