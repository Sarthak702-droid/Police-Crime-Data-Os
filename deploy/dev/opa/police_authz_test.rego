package police.authz_test

import data.police.authz

test_same_unit_investigator_allowed if {
	authz.allow with input as {
		"subject": {"active": true, "roles": ["investigator"], "unit_id": 10, "district_id": 1, "rank_hierarchy": 5},
		"resource": {"unit_id": 10, "district_id": 1},
		"action": "case.read",
	}
}

test_cross_unit_investigator_denied if {
	not authz.allow with input as {
		"subject": {"active": true, "roles": ["investigator"], "unit_id": 10, "district_id": 1, "rank_hierarchy": 5},
		"resource": {"unit_id": 11, "district_id": 1},
		"action": "case.read",
	}
}

test_district_supervisor_allowed if {
	authz.allow with input as {
		"subject": {"active": true, "roles": ["supervisor"], "unit_id": 10, "district_id": 1, "rank_hierarchy": 3},
		"resource": {"unit_id": 11, "district_id": 1},
		"action": "supervisor.review",
	}
}
