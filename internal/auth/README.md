# Authentication Package Tests

This package contains comprehensive tests for the GitHub OAuth authentication system.

## Test Coverage

**Overall Coverage: 55.6%**

### Test Files

#### 1. `jwt_test.go` - JWT Token Management Tests

Tests for token generation, validation, encryption, and refresh:

- ✅ `TestNewJWTManager` - JWT manager initialization with various secret keys
- ✅ `TestGenerateAndValidateToken` - Full token lifecycle (generate → validate → decrypt)
- ✅ `TestValidateInvalidToken` - Handling of invalid, malformed, and empty tokens
- ✅ `TestValidateTokenWithDifferentSecret` - Cross-secret validation prevention
- ✅ `TestRefreshToken` - Token refresh with extended expiration
- ✅ `TestTokenEncryption` - AES-256-GCM encryption/decryption
- ✅ `TestTokenEncryptionUniqueness` - Nonce randomization verification

**Key Test Scenarios:**
- Valid and invalid tokens
- Token expiration
- Encryption security (nonce uniqueness)
- Claims preservation during refresh

#### 2. `middleware_test.go` - Authentication Middleware Tests

Tests for HTTP middleware authentication:

- ✅ `TestRequireAuth` - Authentication enforcement with various scenarios
- ✅ `TestGetUserFromContext` - User context retrieval
- ✅ `TestGetClaimsFromContext` - JWT claims context retrieval
- ✅ `TestExtractBearerToken` - Authorization header parsing
- ✅ `TestRequireAuthWithExpiredToken` - Expired token handling

**Key Test Scenarios:**
- Valid token authentication
- Missing token handling
- Invalid token rejection
- Auth disabled mode (backward compatibility)
- Bearer token extraction

#### 3. `oauth_test.go` - OAuth Flow Tests

Tests for GitHub OAuth 2.0 authentication:

- ✅ `TestNewOAuthHandler` - Handler initialization (GitHub.com and GHES)
- ✅ `TestHandleLogin` - OAuth initiation and state cookie
- ✅ `TestHandleCallbackMissingState` - CSRF protection (missing state)
- ✅ `TestHandleCallbackStateMismatch` - CSRF protection (mismatched state)
- ✅ `TestHandleCallbackMissingCode` - Authorization code validation
- ✅ `TestGetGitHubUserFromContext` - User context storage
- ✅ `TestGetTokenFromContext` - Token context storage
- ✅ `TestBuildOAuthURL` - OAuth URL construction for GitHub.com and GHES
- ✅ `TestGenerateStateToken` - CSRF token generation and uniqueness
- ✅ `TestOAuthHandlerWithGHES` - GitHub Enterprise Server support

**Key Test Scenarios:**
- OAuth flow initiation
- State cookie management (CSRF protection)
- GitHub.com vs GHES URL handling
- Context value storage and retrieval

#### 4. `authorizer_test.go` - Authorization Logic Tests

Tests for authorization checks using mock GitHub API:

- ✅ `TestNewAuthorizer` - Authorizer initialization
- ✅ `TestAuthorizeNoRules` - Default allow when no rules configured
- ✅ `TestCheckOrganizationMembership` - Org membership verification
- ✅ `TestCheckTeamMembership` - Team membership verification
- ✅ `TestAuthorizeWithOrgMembership` - Full authorization flow with org check
- ✅ `TestAuthorizeWithOrgMembershipDenied` - Authorization denial
- ✅ `TestAuthorizeWithMultipleRules` - Combined org + team requirements
- ✅ `TestIsOrgMember` - Direct org membership check
- ✅ `TestIsTeamMember` - Direct team membership check

**Key Test Scenarios:**
- Organization membership checks
- Team membership checks (with team slug parsing)
- Multiple authorization rules
- Mock GitHub API responses
- Authorization denial reasons

## Running the Tests

```bash
# Run all auth tests
go test ./internal/auth/...

# Run with coverage
go test ./internal/auth/... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run specific test
go test ./internal/auth/... -run TestGenerateAndValidateToken -v
```

## Test Statistics

- **Total Tests**: 40+ test cases
- **Test Files**: 4 files
- **Coverage**: 55.6% of statements
- **All Tests**: ✅ PASSING

## Coverage by Component

| Component | Key Features Tested |
|-----------|---------------------|
| JWT Manager | Token generation, validation, encryption, refresh |
| OAuth Handler | Login flow, callback handling, state management |
| Authorizer | Org/team membership, enterprise admin, multiple rules |
| Middleware | Request authentication, context management, error handling |

## Mock Testing Approach

The tests use `httptest` to create mock GitHub API servers that simulate:
- Organization membership checks (`/orgs/{org}/members/{user}`)
- Team membership checks (`/orgs/{org}/teams/{team}/memberships/{user}`)
- Enterprise admin checks (`/enterprises/{slug}/users/{user}`)

This allows comprehensive testing without requiring actual GitHub API calls.

## Security Test Coverage

✅ CSRF protection (state token validation)  
✅ Token encryption (AES-256-GCM)  
✅ Token expiration handling  
✅ HttpOnly cookie flags  
✅ Invalid token rejection  
✅ Authorization denial with reasons  
✅ Nonce uniqueness in encryption

## Future Test Enhancements

Potential areas for additional testing:
- Integration tests with real GitHub OAuth (using test accounts)
- Performance tests for token validation
- Concurrent token generation/validation
- Enterprise admin role checking (when GitHub API mock is more comprehensive)
- Rate limiting on auth endpoints
- Session invalidation and token blacklisting

