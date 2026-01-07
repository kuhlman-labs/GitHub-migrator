# Authentication and Authorization

This package provides authentication and authorization for the GitHub Migrator application.

## Destination-Centric Authorization Model

The GitHub Migrator uses a **destination-centric** authentication and authorization model. This means:

1. **Single Authentication Provider**: All users authenticate via GitHub OAuth against the destination GitHub instance
2. **No Source-Specific OAuth**: Unlike traditional multi-source systems, users do not need to authenticate separately with each source system
3. **Simplified User Experience**: Users log in once and can access migrations from any configured source

### Why Destination-Centric?

In a multi-source, single-destination migration tool, supporting OAuth for N source systems would be complex:
- Each source (GitHub Enterprise Server, Azure DevOps, etc.) would require separate OAuth app configuration
- Users would need to log into each source before migrating repositories
- Session management would become complex with multiple token types

By centralizing authentication on the destination (GitHub), we simplify the architecture while maintaining security through identity mapping.

## Authorization Tiers

The system implements a three-tier authorization model:

### Tier 1: Full Migration Rights (Admin)

Users with full migration rights can initiate migrations for **any** repository. This includes:

- **Enterprise Admins**: Users who are administrators of the configured GitHub Enterprise
- **Migration Admin Teams**: Members of specifically configured GitHub teams
- **Organization Admins**: (Optional) Administrators of required organizations

Configuration:
```yaml
auth:
  authorization_rules:
    require_enterprise_admin: true
    require_enterprise_slug: "my-enterprise"
    migration_admin_teams:
      - "my-org/migration-admins"
      - "my-org/platform-team"
    allow_org_admin_migrations: true
    allow_enterprise_admin_migrations: true
```

### Tier 2: Identity-Mapped Self-Service

Users can initiate migrations for repositories where their **mapped source identity** has admin rights. This tier:

- Requires completion of identity mapping (linking source account to destination account)
- Checks that the mapped source identity has admin permissions on the source repository
- Enables self-service migrations without requiring full admin access

Configuration:
```yaml
auth:
  authorization_rules:
    enable_self_service: true
```

When `enable_self_service` is `false`, any authenticated user can attempt self-service migrations. The actual source repository admin check happens during migration initiation.

### Tier 3: Read-Only Access

All authenticated users have read-only access by default:

- View migration status and history
- Browse discovered repositories
- View batch progress

Users cannot initiate migrations unless they meet Tier 1 or Tier 2 requirements.

## Configuration Reference

### Environment Variables

```bash
# GitHub OAuth (destination)
GHMIG_AUTH_ENABLED=true
GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID=your_client_id
GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET=your_client_secret
GHMIG_AUTH_CALLBACK_URL=https://your-app.com/api/v1/auth/callback
GHMIG_AUTH_FRONTEND_URL=https://your-app.com
GHMIG_AUTH_SESSION_SECRET=your_secure_random_string

# Authorization Rules
GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ORG_MEMBERSHIP=my-org
GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_ADMIN=true
GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_SLUG=my-enterprise
GHMIG_AUTH_AUTHORIZATION_RULES_MIGRATION_ADMIN_TEAMS=my-org/admins,my-org/platform
GHMIG_AUTH_AUTHORIZATION_RULES_ALLOW_ORG_ADMIN_MIGRATIONS=true
GHMIG_AUTH_AUTHORIZATION_RULES_ALLOW_ENTERPRISE_ADMIN_MIGRATIONS=true
GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_IDENTITY_MAPPING_FOR_SELF_SERVICE=true
```

### Config File (YAML)

```yaml
auth:
  enabled: true
  github_oauth_client_id: "your_client_id"
  github_oauth_client_secret: "your_client_secret"
  callback_url: "https://your-app.com/api/v1/auth/callback"
  frontend_url: "https://your-app.com"
  session_secret: "your_secure_random_string"
  session_duration_hours: 24
  
  authorization_rules:
    # Require membership in these organizations to access the app
    require_org_membership:
      - "my-org"
    
    # Require membership in these teams (format: "org/team-slug")
    require_team_membership:
      - "my-org/developers"
    
    # Enterprise admin checks
    require_enterprise_admin: false
    require_enterprise_slug: "my-enterprise"
    
    # Teams that grant Tier 1 (full migration) access
    migration_admin_teams:
      - "my-org/migration-admins"
      - "my-org/platform-team"
    
    # Allow org admins to have Tier 1 access
    allow_org_admin_migrations: true
    
    # Allow enterprise admins to have Tier 1 access
    allow_enterprise_admin_migrations: true
    
    # Require identity mapping for self-service (Tier 2)
    enable_self_service: true
```

## Identity Mapping for Self-Service

Identity mapping links a user's destination GitHub account to their source account(s). This enables:

1. **Self-Service Authorization**: Users can migrate repos they admin on the source
2. **Contribution Attribution**: Commits and history can be attributed correctly
3. **Team Mapping**: Team memberships can be migrated accurately

### How It Works

1. User authenticates with GitHub (destination)
2. User completes identity mapping (links source account)
3. System verifies:
   - User has a valid identity mapping in the database
   - The mapped source identity has admin rights on the source repository
4. If verified, migration is allowed

### Identity Mapping Status

Users can check their identity mapping status via the `/api/v1/auth/authorization-status` endpoint:

```json
{
  "auth_enabled": true,
  "tier": "SelfService",
  "reason": "User has self-service access. Identity mapping is required.",
  "permissions": {
    "can_migrate_all_repos": false,
    "can_migrate_own_repos": true,
    "has_completed_identity_mapping": true
  },
  "upgrade_path": null
}
```

## API Endpoints

### Public Endpoints

- `GET /api/v1/auth/config` - Get authentication configuration
- `GET /api/v1/auth/login` - Initiate OAuth login
- `GET /api/v1/auth/callback` - OAuth callback handler
- `GET /api/v1/auth/sources` - List sources with OAuth (deprecated, returns empty)

### Protected Endpoints (require authentication)

- `POST /api/v1/auth/logout` - Log out user
- `GET /api/v1/auth/user` - Get current user info
- `POST /api/v1/auth/refresh` - Refresh session token
- `GET /api/v1/auth/authorization-status` - Get user's authorization tier and permissions

## Package Components

### JWT Manager (`jwt.go`)

Handles JWT token generation, validation, and encryption:
- AES-256-GCM encryption for tokens
- Configurable expiration
- Token refresh functionality

### OAuth Handler (`oauth.go`)

Manages GitHub OAuth flow:
- Login initiation with state cookies (CSRF protection)
- Callback handling and token exchange
- Support for GitHub.com and GitHub Enterprise Server

### Authorizer (`authorizer.go`)

Performs authorization checks:
- Organization membership verification
- Team membership verification
- Enterprise admin checks
- Tier determination logic

### Middleware (`middleware.go`)

HTTP middleware for authentication:
- Token extraction from Authorization header
- User context injection
- Protected endpoint enforcement

## Test Coverage

The package includes comprehensive tests for all components. See the test files for examples:

- `jwt_test.go` - Token management tests
- `oauth_test.go` - OAuth flow tests
- `authorizer_test.go` - Authorization logic tests
- `middleware_test.go` - Middleware tests

Run tests:
```bash
go test ./internal/auth/... -v
go test ./internal/auth/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Security Considerations

1. **Session Secret**: Use a strong, random string (at least 32 characters)
2. **HTTPS**: Always use HTTPS in production
3. **Cookie Security**: Tokens are stored in HttpOnly cookies
4. **State Tokens**: CSRF protection via state parameter in OAuth flow
5. **Token Encryption**: All tokens are encrypted with AES-256-GCM
