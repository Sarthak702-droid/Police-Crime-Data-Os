package police.authz

default allow := false

allow if {
	input.subject.active == true
	"admin" in input.subject.roles
}

allow if {
	input.subject.active == true
	input.resource.unit_id == input.subject.unit_id
	input.action in {"case.read", "case.write", "chat.query", "analytics.read", "evidence.read", "evidence.write"}
}

allow if {
	input.subject.active == true
	"supervisor" in input.subject.roles
	input.resource.district_id == input.subject.district_id
	input.action in {"case.read", "analytics.read", "evidence.read", "supervisor.review"}
}

redact_person_names if {
	input.subject.rank_hierarchy > 5
}
