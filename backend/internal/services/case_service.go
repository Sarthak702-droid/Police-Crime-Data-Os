package services

import (
	"fmt"
	"sort"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"
)

type CaseService struct {
	repo      *repositories.CaseRepository
	partyRepo *repositories.PartyRepository
}

func NewCaseService(repo *repositories.CaseRepository, partyRepo *repositories.PartyRepository) *CaseService {
	return &CaseService{
		repo:      repo,
		partyRepo: partyRepo,
	}
}

func (s *CaseService) CreateCase(c *models.CaseMaster, districtID int) error {
	year := c.CrimeRegisteredDate.Year()
	if year == 0 {
		year = time.Now().Year()
		c.CrimeRegisteredDate = time.Now()
	}

	// 1. Generate running serial number
	serialNo, err := s.repo.AllocateSerial(c.PoliceStationID, c.CaseCategoryID, year)
	if err != nil {
		return fmt.Errorf("failed to allocate running serial: %w", err)
	}

	// 2. Resolve Category Code
	categoryCode := 1 // Default to FIR (1)
	if c.CaseCategoryID == 3 {
		categoryCode = 3 // UDR
	} else if c.CaseCategoryID == 4 {
		categoryCode = 4 // PAR
	} else if c.CaseCategoryID == 8 {
		categoryCode = 8 // Zero FIR
	}

	// 3. Format serial values
	// CrimeNo format: 1 digit Category + 4 digit District + 4 digit PoliceStation (Unit) + 4 digit Year + 5 digit Serial
	c.CrimeNo = fmt.Sprintf("%d%04d%04d%04d%05d", categoryCode, districtID, c.PoliceStationID, year, serialNo)

	// CaseNo format: YYYY + 5-digit running serial (last 9 digits of CrimeNo)
	c.CaseNo = fmt.Sprintf("%04d%05d", year, serialNo)

	// Set initial status to Under Investigation (default 1) if not set
	if c.CaseStatusID == 0 {
		c.CaseStatusID = 1
	}

	return s.repo.Create(c)
}

func (s *CaseService) GetCaseByID(id int) (*models.CaseMaster, error) {
	return s.repo.GetByID(id)
}

func (s *CaseService) GetCaseByIDForUnit(id int, unitID int) (*models.CaseMaster, error) {
	return s.repo.GetByIDForUnit(id, unitID)
}

func (s *CaseService) SearchCases(filters repositories.SearchFilters) ([]models.CaseMaster, int64, error) {
	return s.repo.Search(filters)
}

type TimelineEvent struct {
	Date        time.Time `json:"date"`
	EventType   string    `json:"event_type"` // Occurrence, Registration, Arrest, Chargesheet
	Description string    `json:"description"`
}

func (s *CaseService) GetTimeline(caseID int) ([]TimelineEvent, error) {
	return s.GetTimelineForUnit(caseID, 0, 0)
}

func (s *CaseService) GetTimelineForUnit(caseID int, unitID int, rankHierarchy int) ([]TimelineEvent, error) {
	var timeline []TimelineEvent

	// 1. Load case details
	cm, err := s.repo.GetByIDForUnit(caseID, unitID)
	if err != nil {
		return nil, err
	}
	if cm == nil {
		return nil, fmt.Errorf("case not found")
	}

	// 2. Add Occurrence Event
	if cm.OccuranceTime != nil {
		timeline = append(timeline, TimelineEvent{
			Date:        cm.OccuranceTime.IncidentFromTs,
			EventType:   "Occurrence",
			Description: fmt.Sprintf("Incident occurred at %s. Brief facts: %s", cm.OccuranceTime.AddressText, cm.BriefFacts),
		})
	} else {
		timeline = append(timeline, TimelineEvent{
			Date:        cm.IncidentFromDate,
			EventType:   "Occurrence",
			Description: fmt.Sprintf("Incident occurred. Brief facts: %s", cm.BriefFacts),
		})
	}

	// 3. Add Registration Event
	timeline = append(timeline, TimelineEvent{
		Date:        cm.CrimeRegisteredDate,
		EventType:   "Registration",
		Description: fmt.Sprintf("FIR Registered with Crime No: %s (Case No: %s) by Officer ID: %d", cm.CrimeNo, cm.CaseNo, cm.PolicePersonID),
	})

	// 4. Add Arrest Events
	arrests, err := s.partyRepo.GetArrestsByCaseID(caseID)
	if err == nil {
		for _, a := range arrests {
			var accusedName string
			if a.Accused != nil {
				accusedName = a.Accused.AccusedName
			} else {
				accusedName = fmt.Sprintf("Accused ID %d", a.AccusedMasterID)
			}

			if rankHierarchy > 5 {
				accusedName = "REDACTED"
			}

			actionStr := "Arrested"
			if a.ArrestSurrenderTypeID == 2 {
				actionStr = "Surrendered"
			}

			timeline = append(timeline, TimelineEvent{
				Date:        a.ArrestSurrenderDate,
				EventType:   "Arrest",
				Description: fmt.Sprintf("%s accused %s by IO ID %d. Produced before Court ID %d", actionStr, accusedName, a.IOID, a.CourtID),
			})
		}
	}

	// 5. Add Chargesheet Event
	cs, err := s.partyRepo.GetChargesheetByCaseID(caseID)
	if err == nil && cs != nil {
		timeline = append(timeline, TimelineEvent{
			Date:        cs.CsDate,
			EventType:   "Chargesheet",
			Description: fmt.Sprintf("Chargesheet filed. Final report type: %s by Officer ID %d", cs.CsType, cs.PolicePersonID),
		})
	}

	// Sort timeline chronologically
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Date.Before(timeline[j].Date)
	})

	return timeline, nil
}
