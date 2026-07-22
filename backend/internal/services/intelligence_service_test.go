package services

import (
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCaseReadinessIdentifiesEvidenceGaps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CaseMaster{}, &models.ComplainantDetails{}, &models.Victim{}, &models.Accused{}, &models.ActSectionAssociation{}, &models.ArrestSurrender{}, &models.ChargesheetDetails{}, &models.Inv_OccuranceTime{}, &models.CaseDocument{}, &models.Employee{}, &models.Unit{}, &models.UnitType{}, &models.District{}, &models.State{}, &models.Rank{}, &models.Designation{}, &models.CaseCategory{}, &models.GravityOffence{}, &models.CrimeHead{}, &models.CrimeSubHead{}, &models.CaseStatusMaster{}, &models.Court{}, &models.Act{}); err != nil {
		t.Fatal(err)
	}

	from := time.Now().Add(-48 * time.Hour)
	cm := models.CaseMaster{CrimeNo: "100010001202600001", CaseNo: "202600001", CrimeRegisteredDate: time.Now().Add(-24 * time.Hour), PoliceStationID: 1, CaseCategoryID: 1, CaseStatusID: 1, IncidentFromDate: from, IncidentToDate: from.Add(time.Hour), InfoReceivedPSDate: from.Add(2 * time.Hour), BriefFacts: "Short facts"}
	if err := db.Create(&cm).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ComplainantDetails{CaseMasterID: cm.CaseMasterID, ComplainantName: "Test"}).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewIntelligenceService(repositories.NewAnalyticsRepository(db))
	result, err := svc.CaseReadiness(cm.CaseMasterID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected readiness result")
	}
	if result.Score >= 65 {
		t.Fatalf("expected incomplete case score below 65, got %d", result.Score)
	}
	foundEvidenceGap := false
	for _, check := range result.Checks {
		if check.Name == "evidence_metadata" && !check.Passed {
			foundEvidenceGap = true
		}
	}
	if !foundEvidenceGap {
		t.Fatal("expected evidence metadata gap")
	}
}

func TestSimilarCasesUsesExplainableSignals(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CaseMaster{}, &models.ActSectionAssociation{}, &models.ComplainantDetails{}, &models.Victim{}, &models.Accused{}, &models.ArrestSurrender{}, &models.ChargesheetDetails{}, &models.Inv_OccuranceTime{}, &models.CaseDocument{}, &models.Employee{}, &models.Unit{}, &models.UnitType{}, &models.District{}, &models.State{}, &models.Rank{}, &models.Designation{}, &models.CaseCategory{}, &models.GravityOffence{}, &models.CrimeHead{}, &models.CrimeSubHead{}, &models.CaseStatusMaster{}, &models.Court{}, &models.Act{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	source := models.CaseMaster{CrimeNo: "A", CaseNo: "202600001", CrimeRegisteredDate: now, PoliceStationID: 1, CaseCategoryID: 1, CrimeMajorHeadID: 1, CrimeMinorHeadID: 2, CaseStatusID: 1, Latitude: 12.9716, Longitude: 77.5946, BriefFacts: "night house breaking gold jewellery stolen through rear window"}
	candidate := models.CaseMaster{CrimeNo: "B", CaseNo: "202600002", CrimeRegisteredDate: now.Add(-24 * time.Hour), PoliceStationID: 1, CaseCategoryID: 1, CrimeMajorHeadID: 1, CrimeMinorHeadID: 2, CaseStatusID: 1, Latitude: 12.9720, Longitude: 77.5950, BriefFacts: "night house breaking jewellery stolen through window"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&candidate).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ActSectionAssociation{CaseMasterID: source.CaseMasterID, ActID: "IPC", SectionID: "457"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ActSectionAssociation{CaseMasterID: candidate.CaseMasterID, ActID: "IPC", SectionID: "457"}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIntelligenceService(repositories.NewAnalyticsRepository(db))
	rows, err := svc.SimilarCases(source.CaseMasterID, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one similar case, got %d", len(rows))
	}
	if rows[0].Score < 60 {
		t.Fatalf("expected strong explainable match, got %d", rows[0].Score)
	}
}
