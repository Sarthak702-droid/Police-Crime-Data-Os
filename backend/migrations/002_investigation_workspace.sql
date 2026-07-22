-- Complete investigation workspace: evidence lifecycle and assigned actions.

ALTER TABLE "CaseDocument" ADD COLUMN IF NOT EXISTS original_name varchar(255);
ALTER TABLE "CaseDocument" ADD COLUMN IF NOT EXISTS content_type varchar(120);
ALTER TABLE "CaseDocument" ADD COLUMN IF NOT EXISTS size_bytes bigint NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS "EvidenceCustodyEvent" (
    "EventID" bigserial PRIMARY KEY,
    "DocumentID" bigint NOT NULL REFERENCES "CaseDocument"("DocumentID"),
    "CaseMasterID" bigint NOT NULL REFERENCES "CaseMaster"("CaseMasterID"),
    event_type varchar(40) NOT NULL,
    actor_id bigint NOT NULL REFERENCES "Employee"("EmployeeID"),
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "InvestigationTask" (
    "TaskID" bigserial PRIMARY KEY,
    "CaseMasterID" bigint NOT NULL REFERENCES "CaseMaster"("CaseMasterID"),
    title varchar(180) NOT NULL,
    description text NOT NULL DEFAULT '',
    priority varchar(20) NOT NULL,
    status varchar(20) NOT NULL,
    assigned_to bigint NOT NULL REFERENCES "Employee"("EmployeeID"),
    created_by bigint NOT NULL REFERENCES "Employee"("EmployeeID"),
    due_at timestamptz NOT NULL,
    completed_at timestamptz,
    completion_note text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_task_priority CHECK (priority IN ('low','medium','high','critical')),
    CONSTRAINT chk_task_status CHECK (status IN ('open','in_progress','blocked','completed','cancelled'))
);

CREATE TABLE IF NOT EXISTS "InvestigationTaskEvent" (
    "TaskEventID" bigserial PRIMARY KEY,
    "TaskID" bigint NOT NULL REFERENCES "InvestigationTask"("TaskID"),
    actor_id bigint NOT NULL REFERENCES "Employee"("EmployeeID"),
    action varchar(40) NOT NULL,
    from_status varchar(20) NOT NULL DEFAULT '',
    to_status varchar(20) NOT NULL DEFAULT '',
    note text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_custody_document ON "EvidenceCustodyEvent" ("DocumentID", created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_case_status ON "InvestigationTask" ("CaseMasterID", status);
CREATE INDEX IF NOT EXISTS idx_task_assignee_due ON "InvestigationTask" (assigned_to, due_at);
CREATE INDEX IF NOT EXISTS idx_task_event_task ON "InvestigationTaskEvent" ("TaskID", created_at DESC);
