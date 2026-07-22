package services_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/config"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPasswordValidation(t *testing.T) {
	if err := services.ValidatePassword("weak"); err == nil {
		t.Error("expected error for short password, got nil")
	}
	if err := services.ValidatePassword("weakpass123"); err == nil {
		t.Error("expected error for password without uppercase, got nil")
	}
	if err := services.ValidatePassword("WEAKPASS123"); err == nil {
		t.Error("expected error for password without lowercase, got nil")
	}
	if err := services.ValidatePassword("WeakPassNoDigit"); err == nil {
		t.Error("expected error for password without digit, got nil")
	}
	if err := services.ValidatePassword("WeakPass123"); err == nil {
		t.Error("expected error for password without special character, got nil")
	}

	if err := services.ValidatePassword("StrongPass123!"); err != nil {
		t.Errorf("expected no error for strong password, got: %v", err)
	}
}

func TestAccountLockout(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_ = db.AutoMigrate(&models.Employee{}, &models.UserCredentials{}, &models.Rank{}, &models.Designation{}, &models.Unit{}, &models.District{}, &models.State{})

	state := models.State{StateID: 1, StateName: "Karnataka"}
	db.Create(&state)
	dist := models.District{DistrictID: 1, DistrictName: "Bengaluru", StateID: 1}
	db.Create(&dist)
	ut := models.UnitType{UnitTypeID: 1, UnitTypeName: "Police Station"}
	db.Create(&ut)
	unit := models.Unit{UnitID: 1, UnitName: "Koramangala", TypeID: 1, StateID: 1, DistrictID: 1}
	db.Create(&unit)
	rank := models.Rank{RankID: 1, RankName: "Inspector", Hierarchy: 5}
	db.Create(&rank)
	desig := models.Designation{DesignationID: 1, DesignationName: "SHO"}
	db.Create(&desig)

	emp := models.Employee{
		EmployeeID:    1,
		KGID:          "KG99999",
		DistrictID:    1,
		UnitID:        1,
		RankID:        1,
		DesignationID: 1,
	}

	authRepo := repositories.NewAuthRepository(db)
	cfg := &config.Config{JWTSecret: "testsecretkeytestsecretkeytestsecretkey", JWTExpiryHours: 1}
	authSvc := services.NewAuthService(authRepo, cfg)

	err = authSvc.Register(&emp, "ValidPass123!")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	for i := 0; i < 4; i++ {
		_, _, _, err := authSvc.Login("KG99999", "WrongPass123!")
		if err == nil || err.Error() != "invalid KGID or password" {
			t.Errorf("expected login error 'invalid KGID or password', got: %v", err)
		}
	}

	_, _, _, err = authSvc.Login("KG99999", "WrongPass123!")
	if err == nil || err.Error() != "account locked due to multiple failed login attempts, please try again in 15 minutes" {
		t.Errorf("expected lockout message on 5th attempt, got: %v", err)
	}

	_, _, _, err = authSvc.Login("KG99999", "ValidPass123!")
	if err == nil || err.Error() == "invalid KGID or password" {
		t.Errorf("expected lockout message on correct password attempt while locked, got: %v", err)
	}
}

func TestRefreshTokenFlow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_ = db.AutoMigrate(&models.Employee{}, &models.UserCredentials{}, &models.RefreshToken{}, &models.Rank{}, &models.Designation{}, &models.Unit{}, &models.District{}, &models.State{})

	state := models.State{StateID: 1, StateName: "Karnataka"}
	db.Create(&state)
	dist := models.District{DistrictID: 1, DistrictName: "Bengaluru", StateID: 1}
	db.Create(&dist)
	ut := models.UnitType{UnitTypeID: 1, UnitTypeName: "Police Station"}
	db.Create(&ut)
	unit := models.Unit{UnitID: 1, UnitName: "Koramangala", TypeID: 1, StateID: 1, DistrictID: 1}
	db.Create(&unit)
	rank := models.Rank{RankID: 1, RankName: "Inspector", Hierarchy: 5}
	db.Create(&rank)
	desig := models.Designation{DesignationID: 1, DesignationName: "SHO"}
	db.Create(&desig)

	emp := models.Employee{
		EmployeeID:    1,
		KGID:          "KG88888",
		DistrictID:    1,
		UnitID:        1,
		RankID:        1,
		DesignationID: 1,
	}

	authRepo := repositories.NewAuthRepository(db)
	cfg := &config.Config{JWTSecret: "testsecretkeytestsecretkeytestsecretkey", JWTExpiryHours: 1}
	authSvc := services.NewAuthService(authRepo, cfg)

	err = authSvc.Register(&emp, "ValidPass123!")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	accessToken, refreshToken, _, err := authSvc.Login("KG88888", "ValidPass123!")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("tokens should not be empty")
	}

	var rt models.RefreshToken
	refreshDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(refreshToken)))
	err = db.Where("token = ?", refreshDigest).First(&rt).Error
	if err != nil {
		t.Fatalf("refresh token not found in database: %v", err)
	}

	newAccessToken, newRefreshToken, err := authSvc.Refresh(refreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if newAccessToken == "" || newRefreshToken == "" {
		t.Fatal("refreshed tokens should not be empty")
	}

	var checkRt models.RefreshToken
	err = db.Where("token = ?", refreshDigest).First(&checkRt).Error
	if err == nil {
		t.Fatal("old refresh token should have been deleted")
	}

	var checkNewRt models.RefreshToken
	newRefreshDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(newRefreshToken)))
	err = db.Where("token = ?", newRefreshDigest).First(&checkNewRt).Error
	if err != nil {
		t.Fatalf("new refresh token not found in database: %v", err)
	}
}

func TestCrossUnitAccessAndRedaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	_ = db.AutoMigrate(
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

	state := models.State{StateID: 1, StateName: "Karnataka"}
	db.Create(&state)
	dist := models.District{DistrictID: 1, DistrictName: "Bengaluru", StateID: 1}
	db.Create(&dist)
	ut := models.UnitType{UnitTypeID: 1, UnitTypeName: "Police Station"}
	db.Create(&ut)
	unit1 := models.Unit{UnitID: 1, UnitName: "Station 1", TypeID: 1, StateID: 1, DistrictID: 1}
	db.Create(&unit1)
	unit2 := models.Unit{UnitID: 2, UnitName: "Station 2", TypeID: 1, StateID: 1, DistrictID: 1}
	db.Create(&unit2)

	case1 := models.CaseMaster{
		CaseMasterID:    101,
		CrimeNo:         "100010001202600001",
		PoliceStationID: 1,
	}
	db.Create(&case1)

	caseRepo := repositories.NewCaseRepository(db)
	partyRepo := repositories.NewPartyRepository(db)
	caseSvc := services.NewCaseService(caseRepo, partyRepo)

	retrievedCase, err := caseSvc.GetCaseByIDForUnit(101, 1)
	if err != nil {
		t.Fatalf("failed to retrieve case: %v", err)
	}
	if retrievedCase == nil {
		t.Fatal("case should be retrieved")
	}

	retrievedCase2, err := caseSvc.GetCaseByIDForUnit(101, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrievedCase2 != nil {
		t.Fatal("expected nil case when accessed from out-of-scope Unit 2")
	}

	filtersUnit2 := repositories.SearchFilters{ScopeUnitID: 2}
	results2, total2, _ := caseSvc.SearchCases(filtersUnit2)
	if len(results2) > 0 || total2 > 0 {
		t.Errorf("expected 0 search results from out-of-scope Unit 2, got %d", total2)
	}

	filtersUnit1 := repositories.SearchFilters{ScopeUnitID: 1}
	_, total1, _ := caseSvc.SearchCases(filtersUnit1)
	if total1 != 1 {
		t.Errorf("expected 1 search result from Unit 1, got %d", total1)
	}
}

func TestTimelineRedaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	_ = db.AutoMigrate(
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

	state := models.State{StateID: 1, StateName: "Karnataka"}
	db.Create(&state)
	dist := models.District{DistrictID: 1, DistrictName: "Bengaluru", StateID: 1}
	db.Create(&dist)
	ut := models.UnitType{UnitTypeID: 1, UnitTypeName: "Police Station"}
	db.Create(&ut)
	unit := models.Unit{UnitID: 1, UnitName: "Koramangala", TypeID: 1, StateID: 1, DistrictID: 1}
	db.Create(&unit)

	case1 := models.CaseMaster{CaseMasterID: 301, PoliceStationID: 1}
	db.Create(&case1)

	accused := models.Accused{AccusedMasterID: 10, CaseMasterID: 301, AccusedName: "Sensitive Name"}
	db.Create(&accused)

	arrest := models.ArrestSurrender{
		ArrestSurrenderID:   1,
		CaseMasterID:        301,
		AccusedMasterID:     10,
		ArrestSurrenderDate: time.Now(),
		IOID:                1,
		CourtID:             1,
		Accused:             &accused,
	}
	db.Create(&arrest)

	caseRepo := repositories.NewCaseRepository(db)
	partyRepo := repositories.NewPartyRepository(db)
	caseSvc := services.NewCaseService(caseRepo, partyRepo)

	timelineInspector, err := caseSvc.GetTimelineForUnit(301, 1, 5)
	if err != nil {
		t.Fatalf("failed to get timeline: %v", err)
	}
	foundSensitiveInspector := false
	for _, event := range timelineInspector {
		if event.EventType == "Arrest" && len(event.Description) > 0 {
			if testingContains(event.Description, "Sensitive Name") {
				foundSensitiveInspector = true
			}
		}
	}
	if !foundSensitiveInspector {
		t.Error("expected timeline description to contain accused name for Inspector")
	}

	timelineConstable, err := caseSvc.GetTimelineForUnit(301, 1, 9)
	if err != nil {
		t.Fatalf("failed to get timeline: %v", err)
	}
	foundRedactedConstable := false
	for _, event := range timelineConstable {
		if event.EventType == "Arrest" && len(event.Description) > 0 {
			if testingContains(event.Description, "REDACTED") && !testingContains(event.Description, "Sensitive Name") {
				foundRedactedConstable = true
			}
		}
	}
	if !foundRedactedConstable {
		t.Error("expected timeline description to be redacted for Constable")
	}
}

func testingContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr || checkSubstr(s, substr))
}

func checkSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntrusionWaf(t *testing.T) {
	t.Chdir(t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_ = db.AutoMigrate(&models.AuditEvent{})

	gin.SetMode(gin.TestMode)
	r := gin.New()

	middleware.ClearBlacklists()

	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.IntrusionWafMiddleware(db))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 OK for safe request, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test?q=1'%20OR%201=1", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for malicious request, got %d", w2.Code)
	}

	if !middleware.IsIPBlocked("1.2.3.4") {
		t.Error("expected IP 1.2.3.4 to be blacklisted, but it was not")
	}

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for blacklisted IP, got %d", w3.Code)
	}

	var count int64
	db.Model(&models.AuditEvent{}).Where("action = ?", "INTRUSION_BLOCKED").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 emergency audit event log, got %d", count)
	}
}
