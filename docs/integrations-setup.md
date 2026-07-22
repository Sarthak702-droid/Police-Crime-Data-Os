# AI, Search, Storage, Graph, Identity and Policy Setup

## Important credential action

The Gemini and Sarvam credentials pasted into chat must be revoked and replaced. Do not reuse them and do not commit replacements. The repository ignores `.env` files and contains placeholders only.

## Provider selection

The requested `gemini-3.6-flash` identifier is not an official stable Gemini API model code. The implemented default is the current stable `gemini-3.5-flash`.

The AI flow is:

1. Receive the officer query and server-side unit/rank context.
2. Translate Kannada to English with Sarvam Mayura when required.
3. Ask Gemini to choose from the allowlisted function declarations.
4. Validate arguments and execute the selected repository-owned tool.
5. Return the scoped/redacted tool result to Gemini for synthesis.
6. Translate the final answer to Kannada when requested.
7. Persist the tool, result identifiers, model, confidence, response hash and redactions in `EvidenceTrail`.
8. Fall back to the deterministic router when Gemini/Sarvam is unavailable.

Gemini never receives database credentials or a general SQL function.

## Backend environment

Copy `backend/.env.example` to `backend/.env` and configure new, rotated credentials:

```dotenv
AI_ENABLED=true
AI_PROVIDER=gemini
AI_BASE_URL=https://generativelanguage.googleapis.com/v1beta
AI_MODEL=gemini-3.5-flash
AI_API_KEY=<rotated Gemini key>
TRANSLATION_BASE_URL=https://api.sarvam.ai
SARVAM_API_KEY=<rotated Sarvam key>
```

Available protected endpoints:

- `POST /api/v1/chat/query` — Gemini tool orchestration with safe fallback.
- `POST /api/v1/ai/translate` — Sarvam Mayura Kannada/English translation.
- `POST /api/v1/ai/speech-to-text` — Sarvam Saaras v3 transcription/translation for short audio.
- `GET /api/v1/ai/tools` — published allowlist/function schemas.
- `GET /api/v1/chat/sessions/:session_id/evidence-trails` — provenance records.
- `GET /api/v1/search/hybrid?q=...` — multilingual E5 plus OpenSearch retrieval.
- `POST /api/v1/search/cases/:id/index` — role-protected case indexing.
- `POST /api/v1/cases/:id/evidence/upload` — scoped multipart upload to S3/MinIO with SHA-256 registration.
- `POST /api/v1/graph/cases/:id/sync` — scoped relational-to-Neo4j synchronization.

## Local self-hosted infrastructure

From `deploy/dev`:

```bash
cp .env.infrastructure.example .env.infrastructure
# Replace every password in .env.infrastructure.
docker compose --env-file .env.infrastructure -f docker-compose.infrastructure.yml --profile ai up -d
```

Services bind only to loopback by default:

- OpenSearch: `http://127.0.0.1:9200` (loopback-only development configuration)
- MinIO S3: `http://127.0.0.1:9000`
- MinIO console: `http://127.0.0.1:9001`
- Neo4j HTTP/Bolt: `http://127.0.0.1:7474`, `bolt://127.0.0.1:7687`
- Keycloak: `http://127.0.0.1:8081`
- Keycloak health/metrics: `http://127.0.0.1:9002`
- OPA: `http://127.0.0.1:8181`
- multilingual E5/TEI: `http://127.0.0.1:8082`

This Compose file is an integration environment, not a production topology. Pin and verify every image digest before deployment.

## Keycloak/OIDC

The `police-intelligence` realm and `crime-api` client are imported automatically. For every Keycloak officer, administrators must set these user attributes:

- `employee_id`
- `kgid`
- `unit_id`
- `district_id`
- `rank`
- `rank_hierarchy`
- `designation`

Then configure the backend:

```dotenv
AUTH_MODE=oidc
OIDC_ISSUER=http://127.0.0.1:8081/realms/police-intelligence
OIDC_AUDIENCE=crime-api
OIDC_JWKS_URL=http://127.0.0.1:8081/realms/police-intelligence/protocol/openid-connect/certs
```

The backend validates RS256, issuer, audience, expiry, key ID and required employee/unit claims. Local HS256 login remains available only when `AUTH_MODE=local`.

## OPA

Enable the policy decision point only after the OPA container is healthy:

```dotenv
OPA_URL=http://127.0.0.1:8181/v1/data/police/authz/allow
```

The backend fails closed if enabled OPA becomes unavailable. The included policy covers unit-level investigator access, district supervisor review, administrative access and rank-based redaction decisions. Production policies should be distributed as signed bundles and tested against departmental role matrices.

## Integration status

- Gemini function calling: implemented and wired.
- Sarvam Mayura translation: implemented and wired.
- Sarvam Saaras v3 speech-to-text: implemented for short REST audio requests.
- Keycloak JWT/JWKS validation: implemented and optional through `AUTH_MODE=oidc`.
- OPA policy enforcement: implemented and optional through `OPA_URL`.
- OpenSearch + multilingual E5: hybrid retrieval and case-indexing adapters are implemented.
- MinIO/S3: signed evidence upload, bucket creation, object hashing and document registration are implemented.
- Neo4j: parameterized case/accused graph synchronization is implemented; relational graph queries remain the safe fallback.

Production enablement still requires TLS certificates, non-demo credentials, retention rules, backup/restore drills, object immutability, OpenSearch index lifecycle policies, Neo4j backup policy and approved data-classification controls.
