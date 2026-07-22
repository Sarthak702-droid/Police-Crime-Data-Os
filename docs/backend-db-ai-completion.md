# Backend, Database and AI Completion Notes

Date: 2026-07-22

This pass intentionally excludes frontend work. It completes the repository-owned ER/database operations and adds safe backend foundations for governed AI. Infrastructure that needs deployed services or credentials is listed separately instead of being simulated.

## Police FIR ER coverage

All tables visible in `Police_FIR_ER_Diagram.pdf` are represented by Go models and migrations:

- `CaseMaster`, `ComplainantDetails`, `Victim`, `Accused`
- `ArrestSurrender` and `inv_arrestsurrenderaccused`
- `Act`, `Section`, `ActSectionAssociation`, `CrimeHeadActSection`
- `CrimeHead`, `CrimeSubHead`, `CaseCategory`, `GravityOffence`, `CaseStatusMaster`
- `State`, `District`, `Unit`, `UnitType`, `Court`
- `Employee`, `Rank`, `Designation`
- `CasteMaster`, `ReligionMaster`, `OccupationMaster`
- `ChargesheetDetails`
- The inferred one-to-one `Inv_OccuranceTime` table

Production additions required by the architecture document are also present: `CaseDocument`, conversations, audit events, refresh tokens, atomic `FIRSequence`, and machine-readable `EvidenceTrail`.

The completed investigation workspace additionally includes `InvestigationTask`,
`InvestigationTaskEvent`, and `EvidenceCustodyEvent`. These provide unit-scoped
assignment, deadlines, completion notes, append-only task history, evidence
classification, authenticated retrieval, and custody-event history. Production
DDL is versioned in `backend/migrations/002_investigation_workspace.sql`.

Database hardening added:

- Atomic station/category/year FIR serial allocation.
- Scoped unique case-number index.
- Case, person, custody, document, conversation, audit and evidence-trail indexes.
- PostgreSQL checks for chronology, coordinates, age, victim-police flag and chargesheet type.
- PostgreSQL full-text index for case narratives.
- Versioned production SQL in `backend/migrations/001_production_hardening.sql`.

## New backend API coverage

The protected `/api/v1` surface now includes:

- `PATCH /cases/:id/status`
- `GET|POST /cases/:id/complainants`
- `GET|POST /cases/:id/victims`
- `GET|POST /cases/:id/accused`
- `GET|POST /cases/:id/arrests`
- `GET|PUT /cases/:id/chargesheet`
- `GET|POST /cases/:id/documents`
- `GET /cases/:id/sections`
- `POST|DELETE /cases/:id/sections`
- `PATCH /cases/:id/{complainants|victims|accused}/:party_id`
- `GET|POST /cases/:id/tasks`
- `PATCH /cases/:id/tasks/:task_id` and task-event history
- `GET /investigation/tasks` and `GET /unit/employees`
- Evidence content, metadata and custody-history endpoints
- `GET /acts` and `GET /sections`
- `GET /chat/sessions/:session_id/evidence-trails`
- Real binary `POST /chat/sessions/:session_id/export/pdf`
- `GET /ai/tools`

Employee registration moved behind authenticated rank authorization. Refresh tokens are stored as SHA-256 digests rather than reusable plaintext.

## High-impact police features (USP)

### Case Readiness Copilot

`GET /analytics/cases/:id/readiness`

Checks incident chronology, location, narrative quality, parties, legal sections, normalized occurrence data and evidence metadata. It returns explicit corrective actions before a file reaches supervisor/court review. It is decision support, not an automated legal decision.

### Explainable Similar-Case Linker

`GET /analytics/cases/:id/similar`

Ranks cases using crime head/sub-head, common legal sections, narrative/modus-operandi similarity and geographic distance. Every match includes reasons, avoiding an opaque “AI similarity score.” This helps officers notice serial patterns that are easy to miss across individual FIR files.

### Pending Investigation Triage

`GET /analytics/pending-actions?minimum_age_days=30`

Prioritizes aged open cases and highlights missing arrest/surrender and final-report actions, with additional weight for grave offences. This gives supervisors an actionable backlog instead of a raw case list.

### Evidence-first conversation

Every chat answer now records the chosen tool, arguments, result identifiers, response hash, confidence, prompt/model label and redaction decisions. Low-rank chat/graph/repeat-offender results redact person data consistently.

## AI/tool orchestration: tools required

The backend currently has a deterministic, allowlisted tool router. This works without sending police data to an external model. `GET /api/v1/ai/tools` publishes the exact tool schemas.

To enable full natural-language LLM planning and Kannada/English synthesis, provision:

1. **Approved Gemini reasoning endpoint** — the backend now uses the official Gemini REST function-calling flow. The stable configured model is `gemini-3.5-flash`; there is no stable `gemini-3.6-flash` model identifier. Configure `AI_BASE_URL`, `AI_MODEL`, and `AI_API_KEY` and set `AI_ENABLED=true` only after policy approval.
2. **Kannada/English translation service** — IndicTrans2 is the recommended self-hosted baseline. Configure `TRANSLATION_BASE_URL`.
3. **Embedding service** — multilingual E5 is a suitable retrieval embedding model. Configure `EMBEDDING_BASE_URL`.
4. **Hybrid search store** — OpenSearch/Elasticsearch for BM25 plus vector retrieval. Configure `SEARCH_BASE_URL`. PostgreSQL full-text remains the safe baseline until this exists.
5. **Object storage** — MinIO/S3-compatible storage for PDFs, photographs, audio and evidence objects. Configure `OBJECT_STORAGE_ENDPOINT`. The API currently registers governed metadata and SHA-256 checksums; binary upload requires this store.
6. **Optional graph store** — Neo4j for statewide multi-hop graph traversal. The current relational co-accused graph remains usable for the MVP.
7. **Optional speech services** — Whisper/Riva for on-prem ASR and an approved TTS service when voice is scheduled.
8. **Identity and policy** — Keycloak/OIDC and a policy engine for production district/state/supervisor and service-to-service authorization.

Never expose a general SQL tool to the model. Only the allowlisted, parameter-validated tools should be callable, and user/unit/rank scope must be injected by the server rather than accepted from model arguments.

## External deployment work still required

The following cannot be completed purely in this source tree: production PostgreSQL provisioning, Keycloak, OpenSearch, Neo4j, object storage, model/translation endpoints, CDC/event bus, Kubernetes, KMS/HSM, SIEM, government-approved NTP, backups/DR, CERT-In operational procedures, and legal/policy approval. The repository now contains the schema, configuration contract and integration boundaries needed for those workstreams.
