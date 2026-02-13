# Collaborative Markdown Editor

A real-time collaborative markdown editor backend built with Go, featuring document synchronization, user management, and multi-user collaboration capabilities.

## Features

- **User Management**: Registration, authentication, and profile management with JWT tokens
- **Document Management**: Create, read, update, and delete documents with ownership and sharing
- **Real-time Collaboration**: Multi-user editing with document updates and snapshots
- **Role-based Access Control**: Owner, editor, and viewer roles for document collaborators
- **User Search**: Search for users to add as collaborators
- **Document Sync**: Integration with external sync server for real-time updates
- **Redis Caching**: Session management and token storage

## Tech Stack

- **Language**: Go 1.24.2
- **Framework**: Gin (HTTP web framework)
- **Database**: PostgreSQL with GORM ORM
- **Cache**: Redis
- **Authentication**: JWT (JSON Web Tokens)
- **Testing**: testify/mock, testify/assert

## Project Structure

```
.
├── auth/                    # Authentication & JWT handling
│   ├── jwt.go              # JWT token generation and verification
│   └── middleware.go       # Authentication middleware
├── cmd/
│   └── server/
│       └── main.go         # Application entry point
├── internal/
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
│   ├── errors/             # Error handling
│   │   └── errors.go
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
  "name": "John Doe",
  "email": "john@example.com",
  "password": "password123"
}
```

#### Login
```
POST /login
Content-Type: application/json

{
  "email": "john@example.com",
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
```

### Document Routes

#### Create Document
```
POST /documents
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "Title": "My Document"
}
```

#### List User Documents
```
GET /documents?page=1&per_page=10
Authorization: Bearer <jwt_token>
```

#### List Shared Documents
```
GET /documents/shared?page=1&per_page=10
Authorization: Bearer <jwt_token>
```

#### Get Document
```
GET /documents/:id
Authorization: Bearer <jwt_token>
```

#### Delete Document
```
DELETE /documents/:id
Authorization: Bearer <jwt_token>
```

### Collaborator Routes

#### List Collaborators
```
GET /documents/:id/collaborators
Authorization: Bearer <jwt_token>
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
```

#### Change Collaborator Role
```
PUT /documents/:id/collaborators
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "user_id": 2,
  "role": "viewer"
}
```

#### Remove Collaborator
```
DELETE /documents/:id/collaborators/:userId
Authorization: Bearer <jwt_token>
```

### Internal Routes (Sync Server)

These endpoints are protected by internal secret authentication.

```
GET /internal/documents/:id/permission?user_id=<user_id>
GET /internal/documents/:id/last-state
POST /internal/documents/:id/update
POST /internal/documents/:id/snapshot
```

## Environment Variables

Create a `.env` file in the root directory:

```env
# Server
PORT=8080
ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=markdown_editor

# Redis
REDIS_ADDRESS=localhost:6379

# JWT
JWT_SECRET=your_jwt_secret_key

# Sync Server
SYNC_ADDRESS=http://localhost:8787
SYNC_SECRET=your_sync_secret

# Internal Communication
INTERNAL_SECRET=your_internal_secret
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
# Edit .env with your configuration
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
- `created_at`, `updated_at`: timestamp

### Documents Table
- `id`: uint64 (primary key)
- `title`: string
- `user_id`: uint64 (foreign key)
- `update_seq`: uint64
- `created_at`, `updated_at`: timestamp

### Document Updates Table
- `id`: uint64 (primary key)
- `document_id`: uint64 (foreign key)
- `seq`: uint64
- `update_binary`: bytea
- `user_id`: uint64
- `created_at`: timestamp

### Document Snapshots Table
- `id`: uint64 (primary key)
- `document_id`: uint64 (foreign key)
- `seq`: uint64
- `snapshot_binary`: bytea
- `created_at`: timestamp

### Document Versions Table
- `id`: uint64 (primary key)
- `document_id`: uint64 (foreign key)
- `name`: string
- `seq`: uint64
- `created_by`: uint64
- `created_at`: timestamp

### Document Collaborators Table
- `document_id`: uint64 (primary key)
- `user_id`: uint64 (primary key)
- `role`: string (owner, editor, viewer)
- `added_at`: timestamp

## Key Concepts

### Authentication
- Users authenticate via JWT tokens
- Tokens are validated and stored in Redis
- Token expiration: 3 days
- Supports both header (`Authorization: Bearer`) and query parameter (`?token=`) authentication

### Authorization
- **Owner**: Full control over document, can add/remove collaborators
- **Editor**: Can edit the document and create updates
- **Viewer**: Can only view the document, cannot make changes
- **None**: No access to the document

### Document Syncing
- Documents track updates with sequence numbers
- Snapshots are created every 200 updates for optimization
- The external sync server handles real-time collaboration
- Updates are stored as binary Yjs updates

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

### Code Organization
- **domain**: Core business models
- **repository**: Data access layer
- **service**: Business logic layer
- **handler**: HTTP request/response handling

### Step To Add New Features
1. Define domain models in `internal/domain/`
2. Create repository interface and implementation
3. Create service interface and implementation
4. Create HTTP handlers
5. Register routes in `cmd/server/main.go`
6. Write unit tests