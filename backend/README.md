# Karnataka Police FIR - Crime Intelligence Platform Backend API

A production-grade REST API backend built in Go, implementing a secure, governed, and scalable crime analytics platform based on the Karnataka Police FIR database schema.

> The current ER completion, new domain endpoints, police-focused intelligence features, and external AI provisioning requirements are documented in [`../docs/backend-db-ai-completion.md`](../docs/backend-db-ai-completion.md).

---

## Technical Stack

- **Language:** Go (1.26+)
- **API Framework:** Gin Web Framework
- **ORM / Query Layer:** GORM
- **Database:** SQLite (default for development), PostgreSQL, or MySQL (via configuration)
- **Authentication:** JWT (JSON Web Tokens) with Bcrypt password hashing
- **Architecture:** Clean Bounded-Context Microservice Architecture (Controller-Service-Repository pattern)

---

## Project Structure

```text
backend/
  cmd/
    server/
      main.go         # Application Entry Point
  internal/
    config/
      config.go       # Configuration Loader (.env)
    database/
      database.go     # GORM Database Connections & Pooling
    models/
      models.go       # Struct mappings for all 26 core tables & extensions
    repositories/
      auth_repo.go
      case_repo.go
      party_repo.go
      chat_repo.go
      analytics_repo.go
    services/
      auth_service.go
      case_service.go
      chat_service.go
    handlers/
      auth_handler.go
      case_handler.go
      chat_handler.go
      analytics_handler.go
    middleware/
      auth.go         # JWT Claims Auth Middleware
      cors.go         # CORS Middleware
      logger.go       # Custom Request JSON Logger
    utils/
      response.go     # Consistent JSON response utilities
  .env                # Local Config file (gitignored)
  .env.example        # Configuration template
  go.mod
  go.sum
```

---

## Setup & Installation

### Prerequisite
Ensure Go is installed on your Windows machine:
```powershell
go version
```

### Installation Steps

1. **Navigate to the Backend directory:**
   ```powershell
   cd C:\Users\91891\Downloads\datathon\backend
   ```

2. **Install dependencies:**
   ```powershell
   go mod tidy
   ```

3. **Configure Environment Variables:**
   Create a `.env` file from the example:
   ```powershell
   copy .env.example .env
   ```
   *(By default, `DB_DIALECT=sqlite` is used with a local database named `police_fir.db`, requiring no database installation or external configurations to run.)*

4. **Run the server:**
   ```powershell
   go run cmd/server/main.go
   ```
   *The server will boot up, perform schema auto-migrations for all 26 tables plus extensions, auto-seed default master records, and listen on `http://localhost:8002`.*

---

## Default Seed Credentials

For verification and testing, the database automatically seeds a default Station House Officer (SHO) with the following credentials:
- **KGID (Karnataka Government ID):** `KG12345`
- **Password:** `password`
- **Default Postings:** Koramangala Police Station (Unit ID `1`), Rank `Inspector` (Rank ID `1`).

---

## API Surface Specification (`/api/v1`)

All API responses return consistent JSON formats.

### Successful Response Format
```json
{
  "success": true,
  "message": "Request successful",
  "data": {}
}
```

### Error Response Format
```json
{
  "success": false,
  "message": "Error message here",
  "error": "Detailed error message"
}
```

### 1. Authentication Endpoints

#### Login
- **Endpoint:** `POST /api/v1/auth/login`
- **Request Body:**
  ```json
  {
    "kgid": "KG12345",
    "password": "password"
  }
  ```
- **Response:**
  ```json
  {
    "success": true,
    "message": "Login successful",
    "data": {
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "employee": {
        "employee_id": 1,
        "first_name": "Ramesh Kumar",
        "kgid": "KG12345",
        "rank": { "rank_name": "Inspector" }
      }
    }
  }
  ```

#### Register
- **Endpoint:** `POST /api/v1/auth/register`
- **Request Body:**
  ```json
  {
    "kgid": "KG99999",
    "password": "securepassword",
    "first_name": "Shivappa",
    "employee_dob": "1985-05-15",
    "gender_id": 1,
    "blood_group_id": 2,
    "physically_challenged": false,
    "appointment_date": "2010-01-10",
    "district_id": 1,
    "unit_id": 1,
    "rank_id": 2,
    "designation_id": 2
  }
  ```

#### Current User
- **Endpoint:** `GET /api/v1/auth/me`
- **Headers:** `Authorization: Bearer <token>`
- **Response:** Returns JWT claims context containing the logged-in officer profile.

---

### 2. Case Management Endpoints

#### Create Case (Register FIR)
- **Endpoint:** `POST /api/v1/cases`
- **Headers:** `Authorization: Bearer <token>`
- **Request Body:**
  ```json
  {
    "case_category_id": 1,
    "gravity_offence_id": 1,
    "crime_major_head_id": 1,
    "crime_minor_head_id": 1,
    "court_id": 1,
    "incident_from_date": "2026-07-04 22:30:00",
    "incident_to_date": "2026-07-04 23:30:00",
    "info_received_ps_date": "2026-07-05 08:15:00",
    "latitude": 12.9348,
    "longitude": 77.6189,
    "brief_facts": "The complainant reported that when they returned home, they found the lock of the front door broken and gold ornaments missing.",
    "occurance_time": {
      "address_text": "No. 45, 4th Block, Koramangala, Bengaluru",
      "h3_cell": "88618c2685fffff"
    },
    "complainants": [
      {
        "complainant_name": "Anil Gowda",
        "age_year": 42,
        "occupation_id": 1,
        "religion_id": 1,
        "caste_id": 1,
        "gender_id": 1
      }
    ],
    "victims": [
      {
        "victim_name": "Anil Gowda",
        "age_year": 42,
        "gender_id": 1,
        "victim_police": "0"
      }
    ],
    "accused_list": [
      {
        "accused_name": "Unknown Person",
        "age_year": 30,
        "gender_id": 1,
        "person_code": "A1"
      }
    ],
    "acts_associated": [
      {
        "act_id": "IPC",
        "section_code": "457",
        "act_order_id": 1,
        "section_order_id": 1
      },
      {
        "act_id": "IPC",
        "section_code": "380",
        "act_order_id": 1,
        "section_order_id": 2
      }
    ]
  }
  ```
- **Response:** Returns the newly registered Case details with auto-generated CrimeNo and CaseNo keys conforming to standard formats.

#### Case Details
- **Endpoint:** `GET /api/v1/cases/:id`
- **Headers:** `Authorization: Bearer <token>`
- **Response:** Returns complete case details including relations (victims, accused, acts, spatio-temporal occurrences, chargesheets).

#### Structured Search
- **Endpoint:** `GET /api/v1/cases/search`
- **Headers:** `Authorization: Bearer <token>`
- **Query Params:**
  - `crime_head_id=1` (Optional)
  - `police_station_id=1` (Optional)
  - `from_date=2026-01-01` (Optional)
  - `to_date=2026-12-31` (Optional)
  - `keyword=burglary` (Optional)
  - `limit=10`
  - `page=1`
- **Response:** Paginated list of matching cases.

#### Case Timeline
- **Endpoint:** `GET /api/v1/cases/:id/timeline`
- **Headers:** `Authorization: Bearer <token>`
- **Response:** Returns chronological timeline events of occurrences, registrations, arrests, and chargesheets.

---

### 3. Conversational / Chat Search AI Endpoints

#### Process NL Query
- **Endpoint:** `POST /api/v1/chat/query`
- **Headers:** `Authorization: Bearer <token>`
- **Request Body:**
  ```json
  {
    "session_id": "session-uuid-12345",
    "message": "show burglary hotspots",
    "language": "en-IN"
  }
  ```
- **Response:**
  ```json
  {
    "success": true,
    "message": "Query processed successfully",
    "data": {
      "answer": "Retrieved burglary hotspot counts by police station over the last 90 days. Found 1 active stations.",
      "language": "en-IN",
      "citations": [
        {
          "police_station_id": 1,
          "week": "2026-07-05T00:00:00Z",
          "case_count": 1
        }
      ]
    }
  }
  ```

#### Export PDF history
- **Endpoint:** `POST /api/v1/chat/sessions/:session_id/export/pdf`
- **Headers:** `Authorization: Bearer <token>`
- **Response:**
  ```json
  {
    "success": true,
    "message": "PDF export initiated successfully",
    "data": {
      "export_filename": "conversation_export_session-uuid-12345.pdf",
      "total_turns": 2,
      "status": "generated",
      "download_uri": "/static/exports/conversation_export_session-uuid-12345.pdf"
    }
  }
  ```

---

### 4. Graph Network & Analytics Endpoints

#### Burglary Hotspots
- **Endpoint:** `GET /api/v1/analytics/hotspots`
- **Headers:** `Authorization: Bearer <token>`
- **Response:** Returns counts of burglary cases aggregated by week and police station.

#### Network Subgraph
- **Endpoint:** `GET /api/v1/graph/subgraph?accused_id=1`
- **Headers:** `Authorization: Bearer <token>`
- **Response:** Returns Nodes and Edges representing co-accusal relationships around suspects.
