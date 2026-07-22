# Security Work Summary

Date: 2026-07-06
Project: Intelligent Conversational AI and Crime Analytics Platform
Scope: Backend API and database-facing security hardening based on `Intelligent Conversational AI and Crime Analytics security.pdf`.

## Security PDF Requirements Addressed

The PDF recommends a zero-trust, API-first, auditable backend for police/FIR data. The implemented work focused on these directly actionable controls:

- Every protected API should enforce user scope.
- Sensitive case, chat, graph, and analytics access should be audit logged.
- APIs should defend against OWASP API risks, especially BOLA/IDOR and excessive data exposure.
- Tokens should be short-lived and production secrets should not use defaults.
- CORS should not allow wildcard credentialed access.
- Backend should have safer HTTP defaults and security headers.
- Demo credentials should not be seeded in production.

## Implemented Changes

### 1. Configuration Security

File: `backend/internal/config/config.go`

Changes:

- Added `Validate()` for security-sensitive configuration.
- Production now rejects default or weak `JWT_SECRET` values.
- JWT expiry is constrained to 1-12 hours.
- Default JWT expiry changed from 24 hours to 8 hours.
- Production rejects wildcard CORS origins.
- Default CORS origins changed from `*` to local development origins only:
  - `http://localhost:3000`
  - `http://127.0.0.1:3000`
- Added configurable HTTP timeout settings:
  - `READ_TIMEOUT_SECONDS`
  - `WRITE_TIMEOUT_SECONDS`
  - `IDLE_TIMEOUT_SECONDS`

Security impact:

- Reduces token replay window.
- Prevents accidental production deployment with weak secrets.
- Avoids unsafe wildcard credentialed CORS.
- Adds operational protection against slow-client/resource exhaustion attacks.

### 2. HTTP Server Hardening

File: `backend/cmd/server/main.go`

Changes:

- Calls `cfg.Validate()` during startup.
- Replaced plain `http.ListenAndServe` with explicit `http.Server`.
- Added read-header, read, write, and idle timeouts.
- Passes database handle into router setup so protected route access can be audit logged.
- Production now skips demo database seeding.

Security impact:

- Safer production startup behavior.
- Less exposure to slowloris-style attacks.
- Prevents default demo officer credentials from being created in production.

### 3. CORS and Security Headers

File: `backend/internal/middleware/cors.go`

Changes:

- Replaced permissive `Access-Control-Allow-Origin: *` behavior.
- CORS now reflects only explicitly configured allowed origins.
- Credentialed CORS is only enabled for matched origins.
- Preflight requests from non-allowed origins return `403`.
- Added `SecurityHeadersMiddleware()` with:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: no-referrer`
  - `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`
  - `Cache-Control: no-store`
  - `Strict-Transport-Security` when TLS is active

Security impact:

- Prevents browser-based credential leakage through broad CORS.
- Adds baseline response hardening for API responses containing sensitive policing data.

### 4. Auth Error Hardening and JWT Claims

Files:

- `backend/internal/middleware/auth.go`
- `backend/internal/handlers/auth_handler.go`
- `backend/internal/services/auth_service.go`

Changes:

- Authentication failures no longer return raw internal error details to clients.
- Invalid token errors no longer echo parser errors.
- Authorization header parsing now uses `SplitN`.
- JWT claims now include `rank_hierarchy` for future role policy checks.

Security impact:

- Reduces information leakage during login/token attacks.
- Adds foundation for role-based authorization without extra database lookups.

### 5. Case-Level Object Authorization

Files:

- `backend/internal/repositories/case_repo.go`
- `backend/internal/services/case_service.go`
- `backend/internal/handlers/case_handler.go`

Changes:

- Added `GetByIDForUnit(caseID, unitID)`.
- Added `ScopeUnitID` to search filters.
- Case search now enforces authenticated user station/unit scope.
- Case detail reads now enforce authenticated user station/unit scope.
- Case timeline reads now enforce authenticated user station/unit scope.
- Search result limit is capped at 100.
- Case creation continues to stamp `PoliceStationID` from authenticated claims, not from client input.

Security impact:

- Fixes IDOR/BOLA risk where any authenticated user could read another station's case by guessing `case_id`.
- Prevents broad data dumping through high search limits.
- Keeps FIR/case ownership tied to the authenticated officer's unit.

### 6. Chat Session Ownership Controls

Files:

- `backend/internal/repositories/chat_repo.go`
- `backend/internal/services/chat_service.go`
- `backend/internal/handlers/chat_handler.go`

Changes:

- Added `GetSessionByIDForUser`.
- Added `GetTurnsBySessionIDForUser`.
- Chat query processing now rejects use of another user's existing `session_id`.
- Chat turn history endpoint now checks session ownership.
- Chat PDF export endpoint now checks session ownership.
- Chat's internal case lookup is scoped to the user's unit.
- Chat's internal graph, hotspot, and repeat-offender analytics are scoped to the user's unit.

Security impact:

- Fixes IDOR/BOLA risk where one user could read or export another user's conversation by knowing the session ID.
- Prevents chat from becoming a bypass around case and analytics authorization.

### 7. Analytics and Graph Scope Controls

Files:

- `backend/internal/repositories/analytics_repo.go`
- `backend/internal/handlers/analytics_handler.go`

Changes:

- Added scoped analytics methods:
  - `GetBurglaryHotspotsForUnit(unitID)`
  - `GetRepeatOffendersForUnit(minCases, unitID)`
  - `GetCoaccusalGraphForUnit(accusedID, unitID)`
- Protected analytics handlers now read authenticated claims and pass `UnitID`.
- Co-accusal graph seed accused lookup is restricted to the caller's unit.
- Graph expansion is restricted to cases in the caller's unit.

Security impact:

- Prevents graph traversal into out-of-scope police station data.
- Reduces accidental exposure of accused/person names outside the user's unit scope.

### 8. Audit Logging

Files:

- `backend/internal/middleware/audit.go`
- `backend/internal/routes/routes.go`
- Existing model used: `models.AuditEvent`

Changes:

- Added `AuditMiddleware(db)` for protected routes.
- Audit logs record:
  - Actor KGID
  - HTTP method/action
  - Route path/resource
  - Response status
  - UTC timestamp
- Audit middleware is applied after JWT authentication to protected APIs.
- Audited surfaces include:
  - Case APIs
  - Chat APIs
  - Analytics APIs
  - Graph APIs
  - Non-GET protected mutations

Security impact:

- Supports the PDF's requirement for attributable sensitive actions.
- Creates a database-backed trail for case access, chat export, graph access, and analytics usage.

### 9. Internal Error Exposure Reduction

File: `backend/internal/utils/response.go`

Changes:

- `SendInternalServerError` no longer exposes raw internal error messages to API clients.

Security impact:

- Prevents database, query, filesystem, or implementation errors from leaking through API responses.

## Files Added

- `backend/internal/middleware/audit.go`

## Files Modified

- `backend/cmd/server/main.go`
- `backend/internal/config/config.go`
- `backend/internal/middleware/auth.go`
- `backend/internal/middleware/cors.go`
- `backend/internal/middleware/audit.go`
- `backend/internal/routes/routes.go`
- `backend/internal/handlers/auth_handler.go`
- `backend/internal/handlers/case_handler.go`
- `backend/internal/handlers/chat_handler.go`
- `backend/internal/handlers/analytics_handler.go`
- `backend/internal/services/auth_service.go`
- `backend/internal/services/case_service.go`
- `backend/internal/services/chat_service.go`
- `backend/internal/repositories/case_repo.go`
- `backend/internal/repositories/chat_repo.go`
- `backend/internal/repositories/analytics_repo.go`
- `backend/internal/utils/response.go`

## Verification Status

Completed:

- `gofmt` was run successfully on the edited Go files.

Not completed yet:

- Full `go test ./...` was started with workspace-local Go cache settings, but the run was interrupted before completion.
- Earlier test attempt without local Go cache failed because the sandbox blocked writes to:
  - `C:\Users\91891\AppData\Local\go-build`
  - `C:\Users\91891\AppData\Roaming\go\telemetry`

Recommended verification command:

```powershell
$env:GOCACHE='C:\Users\91891\Downloads\datathon\.gocache'
$env:GOTELEMETRYDIR='C:\Users\91891\Downloads\datathon\.gotelemetry'
go test ./...
```

Run from:

```text
C:\Users\91891\Downloads\datathon\backend
```

## Remaining Recommended Security Work

These are not yet implemented and should be next:

1. Add automated tests for cross-unit access denial.
2. Add rate limiting for login and chat endpoints.
3. Add refresh-token or OIDC/Keycloak integration instead of local-only JWT login for production.
4. Add field-level redaction for sensitive victim/complainant/accused data based on role.
5. Add immutable/append-only audit log protection outside the primary application database.
6. Add request IDs and trace IDs to audit events.
7. Add audit hash chaining using `BeforeHash` and `AfterHash` fields.
8. Add password policy validation during registration.
9. Add account lockout or throttling after repeated failed login attempts.
10. Add formal database indexes for scoped lookups, especially `CaseMaster.PoliceStationID`, `ConversationSession.UserID`, and audit timestamps.

## Important Note

The current implementation uses station/unit-level authorization because the existing backend schema and JWT claims expose `UnitID` directly. A richer production system should add district/state/supervisor scope through a formal RBAC/ABAC policy engine, as recommended by the PDF.
