# Drishti Police Intelligence Console

Responsive React frontend for the FIR, crime analytics and grounded AI backend.

## Run locally

Start the Go API on port `8002`, then:

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open `http://localhost:5173`. Vite proxies `/api` requests to the Go API, so browser credentials never need to be placed in the frontend configuration.

## Production build

```bash
npm run lint
npm run build
```

Serve `dist/` behind an HTTPS reverse proxy and route `/api/v1` to the Go backend. Never put Gemini, Sarvam, storage, graph or database secrets in `VITE_*` variables because Vite embeds those variables into public browser assets.

## Implemented workflows

- Officer login, refresh-token renewal and secure sign-out
- Optional Keycloak Authorization Code + PKCE department SSO
- Jurisdiction command dashboard
- FIR search, filtering and structured registration
- Case details, readiness checks, timeline, evidence upload and status changes
- Editable complainant, victim and accused demographic records
- Arrest/surrender records, legal sections and chargesheet management
- Evidence preview/download, classification and custody-event history
- Assignable investigation tasks with priorities, deadlines and audit history
- Similar-case intelligence, direct hybrid semantic search and indexing
- Consolidated case-connections workspace for similar FIRs and co-accused links
- AI copilot with English/Kannada, microphone and audio transcription
- Conversation history and audited PDF export
- Hotspot activity, pending-action priorities and co-accused graph explorer

The visual system uses responsive glass surfaces, restrained navy/teal colors, semantic warning colors, reduced-motion support and mobile navigation.
