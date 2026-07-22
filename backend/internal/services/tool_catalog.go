package services

type AIToolDefinition struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	InputSchema    map[string]interface{} `json:"input_schema"`
	DataClass      string                 `json:"data_class"`
	HumanReview    bool                   `json:"human_review_required"`
	Implementation string                 `json:"implementation"`
}

// AvailableAITools is the allowlist exposed to an LLM planner. The model must
// never receive a general SQL executor or unrestricted database credentials.
func AvailableAITools() []AIToolDefinition {
	integer := func(name string) map[string]interface{} {
		return map[string]interface{}{"type": "object", "properties": map[string]interface{}{name: map[string]interface{}{"type": "integer"}}, "required": []string{name}, "additionalProperties": false}
	}
	return []AIToolDefinition{
		{Name: "lookup_case", Description: "Retrieve one role-scoped FIR/case and its governed relations.", InputSchema: integer("case_id"), DataClass: "restricted", HumanReview: false, Implementation: "available"},
		{Name: "search_cases", Description: "Search cases inside the officer's authorised organisational scope.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}}, "required": []string{"query"}, "additionalProperties": false}, DataClass: "restricted", HumanReview: false, Implementation: "available"},
		{Name: "get_case_timeline", Description: "Build an occurrence-to-disposal timeline with role redaction.", InputSchema: integer("case_id"), DataClass: "restricted", HumanReview: false, Implementation: "available"},
		{Name: "get_hotspots", Description: "Return descriptive station-scoped hotspot counts; never autonomous targeting.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"days": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 365}}, "additionalProperties": false}, DataClass: "aggregate", HumanReview: true, Implementation: "available"},
		{Name: "get_repeat_offenders", Description: "Return explainable repeat-offender candidates inside the authorised unit scope.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"minimum_cases": map[string]interface{}{"type": "integer", "minimum": 2, "maximum": 20}}, "additionalProperties": false}, DataClass: "restricted", HumanReview: true, Implementation: "available"},
		{Name: "get_coaccusal_network", Description: "Return a scoped co-accused graph with provenance identifiers.", InputSchema: integer("accused_id"), DataClass: "restricted", HumanReview: true, Implementation: "available"},
		{Name: "assess_case_readiness", Description: "Identify missing FIR/evidence fields before supervisory or court review.", InputSchema: integer("case_id"), DataClass: "restricted", HumanReview: true, Implementation: "available"},
		{Name: "find_similar_cases", Description: "Rank explainable similar cases using crime type, sections, narrative and distance.", InputSchema: integer("case_id"), DataClass: "restricted", HumanReview: true, Implementation: "available"},
		{Name: "get_pending_actions", Description: "Prioritise aged investigations with missing custody or final-report actions.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"minimum_age_days": map[string]interface{}{"type": "integer", "minimum": 1}}, "additionalProperties": false}, DataClass: "restricted", HumanReview: true, Implementation: "available"},
		{Name: "translate_text", Description: "Translate Kannada and English without changing named entities or legal citations.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"text": map[string]interface{}{"type": "string"}, "target_language": map[string]interface{}{"type": "string", "enum": []string{"kn-IN", "en-IN"}}}, "required": []string{"text", "target_language"}, "additionalProperties": false}, DataClass: "restricted", HumanReview: false, Implementation: "requires translation model endpoint"},
	}
}
