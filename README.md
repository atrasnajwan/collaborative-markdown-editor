# Collaborative Markdown Editor

A real-time collaborative markdown editor backend built with Go. It provides auth, document CRUD, storage of encoded Yjs updates, and Y.Doc state hydration for an external sync server.

## Responsibilities

- **Auth** — Registration, login, JWT (access + refresh), profile and password management, user search for collaboration.
- **Document CRUD** — Create, read, update, and delete documents; ownership and sharing; role-based access (owner, editor, viewer).
- **Store encoded Yjs updates** — Persist CRDT updates and snapshots as binary; sequence numbers for ordering.
- **Hydrate Y.Doc state** — Serve full document state (snapshot + updates) so clients or the sync server can reconstruct `Y.Doc` from storage.

## Design Decisions

- **CRDT updates stored as binary** — Yjs updates and snapshots are stored as raw binary (e.g. `bytea`), not decoded or re-encoded. This keeps the backend simple and avoids interpreting CRDT structure.
- **Idempotent hydration** — Replaying the same snapshot and ordered updates always yields the same `Y.Doc` state. Sequence numbers and append-only updates make hydration deterministic and safe to retry.
- **Stateless HTTP layer** — No server-held document state or WebSocket handling here. HTTP handlers are stateless; real-time sync is delegated to an external sync server that uses this service for persistence and state fetch.

---

## Tech Stack

- **Language**: Go 1.24.2
- **Framework**: Gin (HTTP web framework)
- **Database**: PostgreSQL with GORM ORM
- **Cache**: Redis (optional)
- **Authentication**: JWT (JSON Web Tokens)
- **Testing**: testify/mock, testify/assert

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go         # Application entry point
├── internal/
|   ├── auth/            
│   |   └── jwt.go              # JWT token generation and verification
│   ├── config/
│   │   └── config.go       # Configuration management
│   ├── db/
│   │   ├── db.go           # Database connection
│   │   └── migrate.go      # Database migrations
│   ├── document/           # Document domain
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── domain/             # Domain models
│   │   ├── document.go
│   │   └── user.go
│   ├── middleware/
|   |   |── auth.go         # Authentication middleware
│   │   └── error_handler.go # Error handling
│   ├── sync/               # External sync server client
│   │   └── client.go
│   └── user/               # User domain
│       ├── handler.go
│       ├── handler_test.go
│       ├── repository.go
│       └── service.go
├── go.mod
└── README.md
```

## API Endpoints

### User Routes

#### Registration
```
POST /register
Content-Type: application/json

{
  "name": "Atras Najwan",
  "email": "atras@example.com",
  "password": "password123"
}
```

#### Login
```
POST /login
Content-Type: application/json

{
  "email": "atras@example.com",
  "password": "password123"
}

Response:
{
  "token": "jwt_token_here",
  "user": { ... }
}
```

#### Get Profile
```
GET /profile
Authorization: Bearer <jwt_token>

Response:
{
  "id": 1,
  "name": "Atras Najwan",
  "email": "atras@example.com",
  "is_active": true,
  "created_at": "2026-02-21T10:00:00Z"
}
```

#### Update Profile
```
PATCH /profile
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "John Updated",
  "email": "john.new@example.com"
}
```

#### Change Password
```
POST /change-password
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "current_password": "oldpassword123",
  "new_password": "newpassword123"
}
```

#### Refresh Token
```
POST /refresh
Cookie: refresh_token=<refresh_token>

Response:
{
  "access_token": "new_jwt_token_here"
}
```

#### Logout
```
DELETE /logout
Authorization: Bearer <jwt_token>
```

#### Search Users
```
GET /users?q=john
Authorization: Bearer <jwt_token>

Response:
[
  {
    "id": 2,
    "name": "John Smith",
    "email": "john.smith@example.com"
  }
]
```

### Document Routes

#### Create Document
```
POST /documents
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "title": "My Document"
}

Response:
{
  "id": 1,
  "title": "My Document",
  "user_id": 1,
  "update_seq": 0,
  "created_at": "2026-02-21T10:00:00Z",
  "updated_at": "2026-02-21T10:00:00Z"
}
```

#### Rename Document
```
PATCH /documents/:id
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "title": "Updated Document Title"
}
```

#### List User Documents
```
GET /documents?page=1&per_page=10
Authorization: Bearer <jwt_token>

Response:
{
  "data": [
    {
      "id": 1,
      "title": "My Document",
      "role": "owner",
      "created_at": "2026-02-21T10:00:00Z",
      "updated_at": "2026-02-21T10:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "total_page": 1,
    "total": 1,
    "per_page": 10
  }
}
```

#### List Shared Documents
```
GET /documents/shared?page=1&per_page=10
Authorization: Bearer <jwt_token>

Response:
{
  "data": [
    {
      "id": 2,
      "title": "Shared Document",
      "role": "editor",
      "created_at": "2026-02-21T10:00:00Z",
      "updated_at": "2026-02-21T10:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "total_page": 1,
    "total": 1,
    "per_page": 10
  }
}
```

#### Get Document
```
GET /documents/:id
Authorization: Bearer <jwt_token>

Response:
{
  "id": 1,
  "title": "My Document",
  "role": "owner",
  "created_at": "2026-02-21T10:00:00Z",
  "updated_at": "2026-02-21T10:00:00Z"
}
```

#### Get Document State
```
GET /documents/:id/state
Authorization: Bearer <jwt_token>

Response:
{
  "snapshot": "<binary_data>",
  "snapshot_seq": 100,
  "updates": [
    {
      "seq": 101,
      "user_id": 1,
      "update_binary": "<binary_data>",
      "created_at": "2026-02-21T10:00:00Z"
    }
  ]
}
```

#### Create Document Update
```
POST /documents/:id/updates
X-User-Id: <user_id>

<raw binary data of Yjs update>

Response: No Content (204)
```

#### Create Document Snapshot
```
POST /documents/:id/snapshots

<raw binary data of Yjs snapshot>

Response: No Content (204)
```

#### Delete Document
```
DELETE /documents/:id
Authorization: Bearer <jwt_token>

Response: No Content (204)
```

### Collaborator Routes

#### List Collaborators
```
GET /documents/:id/collaborators
Authorization: Bearer <jwt_token>

Response:
[
  {
    "user": {
      "id": 2,
      "name": "Jane Doe",
      "email": "jane@example.com"
    },
    "role": "editor"
  }
]
```

#### Add Collaborator
```
POST /documents/:id/collaborators
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "user_id": 2,
  "role": "editor"  // or "viewer"
}

Response:
{
  "user": {
    "id": 2,
    "name": "Jane Doe",
    "email": "jane@example.com"
  },
  "role": "editor"
}
```

#### Change Collaborator Role
```
PATCH /documents/:id/collaborators
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "user_id": 2,
  "role": "viewer"
}

Response:
{
  "user": {
    "id": 2,
    "name": "Jane Doe",
    "email": "jane@example.com"
  },
  "role": "viewer"
}
```

#### Remove Collaborator
```
DELETE /documents/:id/collaborators/:userId
Authorization: Bearer <jwt_token>

Response:
{
  "message": "collaborator removed"
}
```

### Internal Routes (Sync Server)

These endpoints are protected by internal secret authentication via `X-Internal-Secret` header.

```
GET /internal/documents/:id/permission?user_id=<user_id>
X-Internal-Secret: <internal_secret>

Response:
{
  "user_id": 1,
  "document_id": 1,
  "role": "owner"
}
```

```
GET /internal/documents/:id/state
X-Internal-Secret: <internal_secret>

Response:
{
  "snapshot": "<binary_data>",
  "snapshot_seq": 100,
  "updates": [...]
}
```

```
POST /internal/documents/:id/update
X-Internal-Secret: <internal_secret>

<binary update data>

Response: No Content (204)
```

```
POST /internal/documents/:id/snapshot
X-Internal-Secret: <internal_secret>

<binary snapshot data>

Response: No Content (204)
```

## Environment Variables

Create a `.env` file in the root directory:

```env
# Server
PORT=8080
ENV=development
WORKER_POOL_SIZE=5

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=markdown_editor

# Redis
REDIS_ADDRESS=localhost:6379
REDIS_POOL_SIZE=10

# JWT
JWT_SECRET=your_jwt_secret_key

# Sync Server
SYNC_ADDRESS=http://localhost:8787
SYNC_SECRET=your_sync_secret

# Internal Communication
INTERNAL_SECRET=your_internal_secret

SNAPSHOT_THRESHOLD=200
```

## Getting Started

### Prerequisites

- Go 1.24.2+
- PostgreSQL 12+
- Redis 6+

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd collaborative-markdown-editor
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
```

4. Run the development server:
```bash
./run-dev.sh
```

Or manually:
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## Database Schema

### Users Table
- `id`: uint64 (primary key)
- `name`: string
- `email`: string (unique)
- `password_hash`: string
- `is_active`: boolean
- `token_version`: uint64 (for session management)
- `created_at`, `updated_at`: timestamp

### Documents Table
- `id`: uint64 (primary key)
- `title`: string
- `user_id`: uint64 (foreign key)
- `update_seq`: uint64 (tracks current update sequence)
- `created_at`, `updated_at`: timestamp

### Document Updates Table
- `id`: uint64 (primary key)
- `document_id`: uint64 (foreign key)
- `seq`: uint64 (sequence number)
- `update_binary`: bytea (Yjs binary update)
- `user_id`: uint64 (user who made the update)
- `created_at`: timestamp

### Document Snapshots Table
- `id`: uint64 (primary key)
- `document_id`: uint64 (foreign key)
- `seq`: uint64 (sequence number at snapshot)
- `snapshot_binary`: bytea (Yjs binary snapshot)
- `created_at`: timestamp

### Document Collaborators Table
- `document_id`: uint64 (primary key)
- `user_id`: uint64 (primary key)
- `role`: string (owner, editor, viewer)
- `added_at`: timestamp

## Key Concepts

### Authentication
- Users authenticate via JWT tokens (access token + refresh token)
- **Access Token**: Short-lived JWT token for API requests
- **Refresh Token**: Long-lived HttpOnly cookie for obtaining new access tokens
- Token validation uses token version for invalidation across sessions
- Both header (`Authorization: Bearer <token>`) and cookie-based authentication supported
- Token version increment (e.g., on password change) invalidates old tokens

### Session Management
- Each user has a `token_version` field that gets incremented on:
  - Password change
  - Logout (to invalidate all sessions)
- Tokens include the token version at creation
- Mismatched token version = expired/invalid session

### Authorization
- **Owner**: Full control over document, can add/remove collaborators and delete document
- **Editor**: Can edit the document, create updates, view collaborators
- **Viewer**: Can only view the document and snapshots, no editing capabilities
- **None**: No access to the document

### Document Syncing
- Documents track updates with sequence numbers (incremental)
- Snapshots are created periodically (configurable threshold, default 200 updates) for optimization
- The external sync server handles real-time collaboration via WebSocket
- Updates are stored as binary Yjs updates for conflict-free collaborative editing
- Snapshots contain the full document state at a given sequence point

## Testing

Run tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Unit tests are available for:
- User handler and service
- Document handler and service

## Error Handling

The application uses custom error types with HTTP status codes:
- `400`: Invalid input
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Not found
- `409`: Conflict (duplicate data)
- `422`: Unprocessable entity
- `500`: Internal server error

## Development

### Project Structure
The project follows a clean architecture pattern with clear separation of concerns:

- **domain**: Core business models and entities
- **repository**: Data access layer interfaces and implementations (using GORM)
- **service**: Business logic layer with validation and orchestration
- **handler**: HTTP request/response handling using Gin framework

### Layers Overview
1. **Handler Layer**: Validates HTTP input, calls service, formats responses
2. **Service Layer**: Implements business logic, enforces authorization, manages cache
3. **Repository Layer**: Manages database operations, implements data persistence
4. **Domain Layer**: Defines core entities and value objects