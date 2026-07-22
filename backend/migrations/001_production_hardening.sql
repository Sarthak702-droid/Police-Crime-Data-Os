-- PostgreSQL production hardening for the Karnataka Police FIR ER model.
-- Apply through the deployment migration runner; do not run against SQLite.

CREATE UNIQUE INDEX IF NOT EXISTS uidx_case_number_scope
    ON "CaseMaster" ("PoliceStationID", "CaseCategoryID", "CaseNo");
CREATE INDEX IF NOT EXISTS idx_case_station_registered
    ON "CaseMaster" ("PoliceStationID", "CrimeRegisteredDate" DESC);
CREATE INDEX IF NOT EXISTS idx_case_status ON "CaseMaster" ("CaseStatusID");
CREATE INDEX IF NOT EXISTS idx_complainant_case ON "ComplainantDetails" ("CaseMasterID");
CREATE INDEX IF NOT EXISTS idx_victim_case ON "Victim" ("CaseMasterID");
CREATE INDEX IF NOT EXISTS idx_accused_case ON "Accused" ("CaseMasterID");
CREATE INDEX IF NOT EXISTS idx_accused_name ON "Accused" ("AccusedName");
CREATE INDEX IF NOT EXISTS idx_arrest_case ON "ArrestSurrender" ("CaseMasterID");
CREATE INDEX IF NOT EXISTS idx_document_case ON "CaseDocument" ("CaseMasterID");
CREATE INDEX IF NOT EXISTS idx_turn_session ON "ConversationTurn" ("SessionID");
CREATE INDEX IF NOT EXISTS idx_evidence_session ON "EvidenceTrail" ("SessionID", "CreatedAt");
CREATE INDEX IF NOT EXISTS idx_audit_created ON "AuditEvent" (created_at DESC);

DO $$ BEGIN
    ALTER TABLE "CaseMaster" ADD CONSTRAINT chk_case_incident_dates
        CHECK ("IncidentToDate" >= "IncidentFromDate");
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE "CaseMaster" ADD CONSTRAINT chk_case_coordinates
        CHECK (latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE "ChargesheetDetails" ADD CONSTRAINT chk_chargesheet_type
        CHECK (cstype IN ('A', 'B', 'C'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE "Victim" ADD CONSTRAINT chk_victim_police
        CHECK ("VictimPolice" IN ('0', '1'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE "ComplainantDetails" ADD CONSTRAINT chk_complainant_age
        CHECK ("AgeYear" BETWEEN 0 AND 125);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE "Victim" ADD CONSTRAINT chk_victim_age
        CHECK ("AgeYear" BETWEEN 0 AND 125);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE "Accused" ADD CONSTRAINT chk_accused_age
        CHECK ("AgeYear" BETWEEN 0 AND 125);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- PostgreSQL full-text baseline. OpenSearch can supersede this for statewide
-- hybrid retrieval, but this index provides safe lexical search immediately.
CREATE INDEX IF NOT EXISTS idx_case_brief_facts_fts
    ON "CaseMaster" USING GIN (to_tsvector('simple', coalesce("BriefFacts", '')));
