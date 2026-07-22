package services

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"backend/internal/models"
	"backend/internal/repositories"
)

type IntelligenceService struct {
	repo *repositories.AnalyticsRepository
}

func NewIntelligenceService(repo *repositories.AnalyticsRepository) *IntelligenceService {
	return &IntelligenceService{repo: repo}
}

type ReadinessCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Weight int    `json:"weight"`
	Action string `json:"action,omitempty"`
}
type CaseReadiness struct {
	CaseMasterID int              `json:"case_master_id"`
	CrimeNo      string           `json:"crime_no"`
	Score        int              `json:"score"`
	Band         string           `json:"band"`
	Checks       []ReadinessCheck `json:"checks"`
	Disclaimer   string           `json:"disclaimer"`
}

func (s *IntelligenceService) CaseReadiness(caseID, unitID int) (*CaseReadiness, error) {
	cm, err := s.repo.GetCaseIntelligenceRecord(caseID, unitID)
	if err != nil || cm == nil {
		return nil, err
	}
	docs, err := s.repo.GetCaseDocuments(caseID)
	if err != nil {
		return nil, err
	}
	checks := []ReadinessCheck{
		{Name: "incident_chronology", Passed: !cm.IncidentFromDate.IsZero() && !cm.IncidentToDate.Before(cm.IncidentFromDate) && !cm.InfoReceivedPSDate.Before(cm.IncidentFromDate), Weight: 10, Action: "Correct incident and information-received timestamps."},
		{Name: "location", Passed: (cm.Latitude != 0 && cm.Longitude != 0) || (cm.OccuranceTime != nil && strings.TrimSpace(cm.OccuranceTime.AddressText) != ""), Weight: 10, Action: "Record coordinates or a precise occurrence address."},
		{Name: "brief_facts", Passed: len(strings.Fields(cm.BriefFacts)) >= 12, Weight: 15, Action: "Expand brief facts with what, when, where, how and reported loss/injury."},
		{Name: "complainant", Passed: len(cm.Complainants) > 0, Weight: 10, Action: "Link at least one complainant record."},
		{Name: "victim", Passed: len(cm.Victims) > 0, Weight: 10, Action: "Record the victim or explicitly document why no victim record applies."},
		{Name: "accused_or_unknown", Passed: len(cm.AccusedList) > 0, Weight: 10, Action: "Add known accused or an explicit Unknown Person placeholder."},
		{Name: "legal_sections", Passed: len(cm.ActsAssociated) > 0, Weight: 15, Action: "Map at least one act and section."},
		{Name: "occurrence_record", Passed: cm.OccuranceTime != nil, Weight: 10, Action: "Create the normalized occurrence-time/location record."},
		{Name: "evidence_metadata", Passed: len(docs) > 0, Weight: 10, Action: "Register evidence/document metadata with checksum and storage URI."},
	}
	score := 0
	for i := range checks {
		if checks[i].Passed {
			score += checks[i].Weight
			checks[i].Action = ""
		}
	}
	band := "needs-attention"
	if score >= 85 {
		band = "ready-for-supervisor-review"
	} else if score >= 65 {
		band = "partially-ready"
	}
	return &CaseReadiness{CaseMasterID: cm.CaseMasterID, CrimeNo: cm.CrimeNo, Score: score, Band: band, Checks: checks, Disclaimer: "Decision support only; the investigating officer and supervisor retain final responsibility."}, nil
}

type SimilarCase struct {
	CaseMasterID   int       `json:"case_master_id"`
	CrimeNo        string    `json:"crime_no"`
	RegisteredDate time.Time `json:"registered_date"`
	Score          int       `json:"score"`
	DistanceKM     *float64  `json:"distance_km,omitempty"`
	Reasons        []string  `json:"reasons"`
}

func (s *IntelligenceService) SimilarCases(caseID, unitID, limit int) ([]SimilarCase, error) {
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	source, err := s.repo.GetCaseIntelligenceRecord(caseID, unitID)
	if err != nil || source == nil {
		return nil, err
	}
	candidates, err := s.repo.GetSimilarCaseCandidates(source, unitID)
	if err != nil {
		return nil, err
	}
	sourceSections := sectionSet(source.ActsAssociated)
	sourceWords := meaningfulWords(source.BriefFacts)
	results := make([]SimilarCase, 0)
	for _, candidate := range candidates {
		score := 0
		reasons := []string{}
		if candidate.CrimeMinorHeadID == source.CrimeMinorHeadID {
			score += 35
			reasons = append(reasons, "same crime sub-head")
		} else if candidate.CrimeMajorHeadID == source.CrimeMajorHeadID {
			score += 15
			reasons = append(reasons, "same major crime head")
		}
		overlap := setOverlap(sourceSections, sectionSet(candidate.ActsAssociated))
		if overlap > 0 {
			points := overlap * 10
			if points > 20 {
				points = 20
			}
			score += points
			reasons = append(reasons, "shared legal sections")
		}
		wordSimilarity := jaccard(sourceWords, meaningfulWords(candidate.BriefFacts))
		if wordSimilarity >= 0.15 {
			points := int(math.Round(wordSimilarity * 20))
			if points > 20 {
				points = 20
			}
			score += points
			reasons = append(reasons, "similar modus-operandi narrative")
		}
		var distance *float64
		if source.Latitude != 0 && source.Longitude != 0 && candidate.Latitude != 0 && candidate.Longitude != 0 {
			value := haversineKM(source.Latitude, source.Longitude, candidate.Latitude, candidate.Longitude)
			distance = &value
			if value <= 1 {
				score += 15
				reasons = append(reasons, "within 1 km")
			} else if value <= 5 {
				score += 8
				reasons = append(reasons, "within 5 km")
			}
		}
		if score >= 25 {
			results = append(results, SimilarCase{CaseMasterID: candidate.CaseMasterID, CrimeNo: candidate.CrimeNo, RegisteredDate: candidate.CrimeRegisteredDate, Score: score, DistanceKM: distance, Reasons: reasons})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].RegisteredDate.After(results[j].RegisteredDate)
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

type PendingCase struct {
	CaseMasterID   int      `json:"case_master_id"`
	CrimeNo        string   `json:"crime_no"`
	AgeDays        int      `json:"age_days"`
	PriorityScore  int      `json:"priority_score"`
	MissingActions []string `json:"missing_actions"`
}

func (s *IntelligenceService) PendingActions(unitID, minimumAgeDays int) ([]PendingCase, error) {
	if minimumAgeDays <= 0 {
		minimumAgeDays = 30
	}
	rows, err := s.repo.GetPendingCaseCandidates(unitID, 1)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	results := []PendingCase{}
	for _, cm := range rows {
		age := int(now.Sub(cm.CrimeRegisteredDate).Hours() / 24)
		if age < minimumAgeDays {
			continue
		}
		missing := []string{}
		score := age
		if len(cm.Arrests) == 0 {
			missing = append(missing, "no arrest/surrender event")
			score += 15
		}
		if cm.Chargesheet == nil {
			missing = append(missing, "no final report/chargesheet")
			score += 20
		}
		if cm.GravityOffenceID == 1 {
			score += 25
		}
		results = append(results, PendingCase{CaseMasterID: cm.CaseMasterID, CrimeNo: cm.CrimeNo, AgeDays: age, PriorityScore: score, MissingActions: missing})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].PriorityScore > results[j].PriorityScore })
	return results, nil
}

func sectionSet(rows []models.ActSectionAssociation) map[string]bool {
	out := map[string]bool{}
	for _, r := range rows {
		out[strings.ToUpper(r.ActID+":"+r.SectionID)] = true
	}
	return out
}
func setOverlap(a, b map[string]bool) int {
	n := 0
	for key := range a {
		if b[key] {
			n++
		}
	}
	return n
}
func meaningfulWords(value string) map[string]bool {
	stop := map[string]bool{"the": true, "and": true, "was": true, "were": true, "that": true, "with": true, "from": true, "this": true, "have": true, "has": true, "for": true}
	out := map[string]bool{}
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for _, word := range fields {
		if len(word) >= 4 && !stop[word] {
			out[word] = true
		}
	}
	return out
}
func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	union := map[string]bool{}
	for k := range a {
		union[k] = true
		if b[k] {
			intersection++
		}
	}
	for k := range b {
		union[k] = true
	}
	return float64(intersection) / float64(len(union))
}
func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const radius = 6371.0
	toRad := math.Pi / 180
	dLat := (lat2 - lat1) * toRad
	dLon := (lon2 - lon1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return radius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
