package services

import (
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCreateCaseAndTimeline(t *testing.T) {
	// 1. Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// 2. Run migrations
	err = db.AutoMigrate(
		&models.CaseMaster{},
		&models.Inv_OccuranceTime{},
		&models.ComplainantDetails{},
		&models.Victim{},
		&models.Accused{},
		&models.ArrestSurrender{},
		&models.InvArrestSurrenderAccused{},
		&models.ChargesheetDetails{},
		&models.Employee{},
		&models.Rank{},
		&models.Designation{},
		&models.Unit{},
		&models.UnitType{},
		&models.District{},
		&models.State{},
		&models.CaseCategory{},
		&models.CaseStatusMaster{},
		&models.ActSectionAssociation{},
		&models.Act{},
		&models.GravityOffence{},
		&models.CrimeHead{},
		&models.CrimeSubHead{},
		&models.Court{},
		&models.CasteMaster{},
		&models.ReligionMaster{},
		&models.OccupationMaster{},
	)
	if err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}

	// Seed basic lookup tables
	db.Create(&models.CaseCategory{CaseCategoryID: 1, LookupValue: "FIR"})
	db.Create(&models.CaseStatusMaster{CaseStatusID: 1, CaseStatusName: "Under Investigation"})

	// 3. Initialize Repositories and Services
	caseRepo := repositories.NewCaseRepository(db)
	partyRepo := repositories.NewPartyRepository(db)
	svc := NewCaseService(caseRepo, partyRepo)

	// 4. Set up mock case inputs
	complainant := models.ComplainantDetails{
		ComplainantName: "Basavaraj",
		AgeYear:         45,
	}

	occurrence := &models.Inv_OccuranceTime{
		AddressText: "MG Road, Bengaluru",
	}

	c := &models.CaseMaster{
		CrimeRegisteredDate: time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
		PolicePersonID:      1,
		PoliceStationID:     1,
		CaseCategoryID:      1, // FIR
		GravityOffenceID:    1,
		CrimeMajorHeadID:    1,
		CrimeMinorHeadID:    1,
		IncidentFromDate:   time.Date(2026, 7, 4, 22, 0, 0, 0, time.UTC),
		IncidentToDate:     time.Date(2026, 7, 4, 23, 0, 0, 0, time.UTC),
		InfoReceivedPSDate: time.Date(2026, 7, 5, 8, 0, 0, 0, time.UTC),
		BriefFacts:         "House breaking theft at night",
		OccuranceTime:      occurrence,
		Complainants:       []models.ComplainantDetails{complainant},
	}

	// 5. Test Case Creation & Serial Number Formatting
	districtID := 44 // Mock Bengaluru City District ID
	err = svc.CreateCase(c, districtID)
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	// Verify CrimeNo format: 1 Category + 4 District + 4 Station + 4 Year + 5 Serial
	// Expected: "100440001202600001"
	expectedCrimeNo := "100440001202600001"
	if c.CrimeNo != expectedCrimeNo {
		t.Errorf("expected CrimeNo %s, got %s", expectedCrimeNo, c.CrimeNo)
	}

	// Verify CaseNo format: YYYY + 5-digit Serial (last 9 digits of CrimeNo)
	// Expected: "202600001"
	expectedCaseNo := "202600001"
	if c.CaseNo != expectedCaseNo {
		t.Errorf("expected CaseNo %s, got %s", expectedCaseNo, c.CaseNo)
	}

	// 6. Test Chronological Case Timeline Generation
	timeline, err := svc.GetTimeline(c.CaseMasterID)
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	// Should contain 2 events initially: Occurrence, then Registration
	if len(timeline) != 2 {
		t.Fatalf("expected 2 timeline events, got %d", len(timeline))
	}

	if timeline[0].EventType != "Occurrence" {
		t.Errorf("expected first event to be Occurrence, got %s", timeline[0].EventType)
	}

	if timeline[1].EventType != "Registration" {
		t.Errorf("expected second event to be Registration, got %s", timeline[1].EventType)
	}

	// Verify timeline chronological order: occurrence is on July 4th, registration is on July 5th
	if !timeline[0].Date.Before(timeline[1].Date) {
		t.Errorf("timeline events are out of order")
	}
}
