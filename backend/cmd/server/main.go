package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handlers"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/routes"
	"backend/internal/services"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting Crime Intelligence Platform Backend...")

	// 1. Load Configurations
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid security configuration: %v", err)
	}

	// 2. Connect to Database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// 3. Run Auto-migrations
	log.Println("Running database schema auto-migrations...")
	err = db.AutoMigrate(
		&models.State{},
		&models.District{},
		&models.UnitType{},
		&models.Unit{},
		&models.Rank{},
		&models.Designation{},
		&models.Employee{},
		&models.CaseCategory{},
		&models.GravityOffence{},
		&models.CrimeHead{},
		&models.CrimeSubHead{},
		&models.Act{},
		&models.Section{},
		&models.CrimeHeadActSection{},
		&models.CasteMaster{},
		&models.ReligionMaster{},
		&models.OccupationMaster{},
		&models.CaseStatusMaster{},
		&models.Court{},
		&models.CaseMaster{},
		&models.ComplainantDetails{},
		&models.Victim{},
		&models.Accused{},
		&models.ArrestSurrender{},
		&models.InvArrestSurrenderAccused{},
		&models.ChargesheetDetails{},
		&models.Inv_OccuranceTime{},
		&models.UserCredentials{},
		&models.CaseDocument{},
		&models.EvidenceCustodyEvent{},
		&models.InvestigationTask{},
		&models.InvestigationTaskEvent{},
		&models.ConversationSession{},
		&models.ConversationTurn{},
		&models.AuditEvent{},
		&models.RefreshToken{},
		&models.FIRSequence{},
		&models.EvidenceTrail{},
	)
	if err != nil {
		log.Fatalf("Auto-migration failed: %v", err)
	}
	log.Println("Schema auto-migrations completed successfully")

	// 4. Seed Database
	seedDatabase(db, cfg)

	// 5. Initialize Repositories
	authRepo := repositories.NewAuthRepository(db)
	caseRepo := repositories.NewCaseRepository(db)
	partyRepo := repositories.NewPartyRepository(db)
	chatRepo := repositories.NewChatRepository(db)
	analyticsRepo := repositories.NewAnalyticsRepository(db)
	domainRepo := repositories.NewDomainRepository(db)

	// 6. Initialize Services
	authSvc := services.NewAuthService(authRepo, cfg)
	caseSvc := services.NewCaseService(caseRepo, partyRepo)
	chatSvc := services.NewChatService(chatRepo, caseRepo, analyticsRepo)
	intelligenceSvc := services.NewIntelligenceService(analyticsRepo)
	if cfg.AIEnabled {
		geminiClient := services.NewGeminiClient(cfg.AIBaseURL, cfg.AIModel, cfg.AIAPIKey)
		sarvamClient := services.NewSarvamClient(cfg.TranslationBaseURL, cfg.SarvamAPIKey)
		chatSvc.SetOrchestrator(services.NewGeminiOrchestrator(geminiClient, sarvamClient, caseRepo, caseSvc, analyticsRepo, intelligenceSvc, cfg.AIModel))
		log.Printf("Governed AI orchestration enabled with model %s", cfg.AIModel)
	}

	// 7. Initialize Handlers
	authHandler := handlers.NewAuthHandler(authSvc)
	caseHandler := handlers.NewCaseHandler(caseSvc)
	chatHandler := handlers.NewChatHandler(chatSvc)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsRepo, intelligenceSvc)
	domainHandler := handlers.NewDomainHandler(domainRepo)
	var retrievalHandler *handlers.RetrievalHandler
	if cfg.EmbeddingBaseURL != "" && cfg.SearchBaseURL != "" {
		retrievalSvc := services.NewRetrievalService(services.NewEmbeddingClient(cfg.EmbeddingBaseURL), services.NewOpenSearchClient(cfg.SearchBaseURL, cfg.SearchIndex, cfg.SearchUsername, cfg.SearchPassword), caseRepo)
		retrievalHandler = handlers.NewRetrievalHandler(retrievalSvc)
	}
	var evidenceHandler *handlers.EvidenceHandler
	if cfg.ObjectStoreEndpoint != "" && cfg.ObjectStoreAccessKey != "" && cfg.ObjectStoreSecretKey != "" {
		store := services.NewS3ObjectStore(cfg.ObjectStoreEndpoint, cfg.ObjectStoreAccessKey, cfg.ObjectStoreSecretKey, cfg.ObjectStoreBucket, cfg.ObjectStoreRegion)
		evidenceHandler = handlers.NewEvidenceHandler(services.NewEvidenceStorageService(store, domainRepo))
	}
	var graphSyncHandler *handlers.GraphSyncHandler
	if cfg.GraphBaseURL != "" && cfg.GraphPassword != "" {
		graphSyncHandler = handlers.NewGraphSyncHandler(services.NewGraphSyncService(services.NewNeo4jClient(cfg.GraphBaseURL, cfg.GraphUsername, cfg.GraphPassword), caseRepo))
	}

	// 8. Setup Routing
	router := routes.SetupRouter(cfg, db, authHandler, caseHandler, chatHandler, analyticsHandler, domainHandler, retrievalHandler, evidenceHandler, graphSyncHandler)

	// 9. Start Server
	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server listening on %s", serverAddr)
	server := &http.Server{
		Addr:              serverAddr,
		Handler:           router,
		ReadHeaderTimeout: time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeoutSeconds) * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func seedDatabase(db *gorm.DB, cfg *config.Config) {
	if cfg.Env == "production" {
		log.Println("Production environment detected, skipping demo seed phase")
		return
	}
	// We will count states to determine if seeding is required
	var stateCount int64
	db.Model(&models.State{}).Count(&stateCount)
	if stateCount > 0 {
		log.Println("Database already has records, skipping seed phase")
		return
	}

	log.Println("Seeding default database records...")

	// 1. Seed State
	state := models.State{StateID: 1, StateName: "Karnataka", NationalityID: 1, Active: true}
	db.Create(&state)

	// 2. Seed District
	district := models.District{DistrictID: 1, DistrictName: "Bengaluru City", StateID: 1, Active: true}
	db.Create(&district)

	// 3. Seed UnitType
	ut := models.UnitType{UnitTypeID: 1, UnitTypeName: "Police Station", CityDistState: "District", Hierarchy: 3, Active: true}
	db.Create(&ut)

	// 4. Seed Unit (Police Station)
	unit := models.Unit{UnitID: 1, UnitName: "Koramangala PS", TypeID: 1, StateID: 1, DistrictID: 1, Active: true}
	db.Create(&unit)

	// 5. Seed Ranks
	rank := models.Rank{RankID: 1, RankName: "Inspector", Hierarchy: 5, Active: true}
	db.Create(&rank)
	db.Create(&models.Rank{RankID: 2, RankName: "Constable", Hierarchy: 9, Active: true})

	// 6. Seed Designations
	desig := models.Designation{DesignationID: 1, DesignationName: "Station House Officer", Active: true, SortOrder: 1}
	db.Create(&desig)
	db.Create(&models.Designation{DesignationID: 2, DesignationName: "Investigating Officer", Active: true, SortOrder: 2})

	// 7. Seed Employee (KGID: KG12345, Password: password)
	emp := models.Employee{
		EmployeeID:           1,
		DistrictID:           1,
		UnitID:               1,
		RankID:               1,
		DesignationID:        1,
		KGID:                 "KG12345",
		FirstName:            "Ramesh Kumar",
		EmployeeDOB:          time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		GenderID:             1,
		BloodGroupID:         1,
		PhysicallyChallenged: false,
		AppointmentDate:      time.Date(2005, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(&emp)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	creds := models.UserCredentials{
		EmployeeID:   1,
		PasswordHash: string(hashedPassword),
	}
	db.Create(&creds)

	// 8. Seed Case Categories
	db.Create(&models.CaseCategory{CaseCategoryID: 1, LookupValue: "FIR"})
	db.Create(&models.CaseCategory{CaseCategoryID: 3, LookupValue: "UDR"})
	db.Create(&models.CaseCategory{CaseCategoryID: 4, LookupValue: "PAR"})
	db.Create(&models.CaseCategory{CaseCategoryID: 8, LookupValue: "Zero FIR"})

	// 9. Seed Gravity Offence
	db.Create(&models.GravityOffence{GravityOffenceID: 1, LookupValue: "Heinous"})
	db.Create(&models.GravityOffence{GravityOffenceID: 2, LookupValue: "Non-Heinous"})

	// 10. Seed Crime Heads
	db.Create(&models.CrimeHead{CrimeHeadID: 1, CrimeGroupName: "Crimes Against Property", Active: true})
	db.Create(&models.CrimeHead{CrimeHeadID: 2, CrimeGroupName: "Crimes Against Body", Active: true})

	// 11. Seed Crime Sub Heads
	db.Create(&models.CrimeSubHead{CrimeSubHeadID: 1, CrimeHeadID: 1, CrimeHeadName: "Burglary - Night House Breaking", SeqID: 1})
	db.Create(&models.CrimeSubHead{CrimeSubHeadID: 2, CrimeHeadID: 2, CrimeHeadName: "Murder for Gain", SeqID: 2})

	// 12. Seed Case Status
	db.Create(&models.CaseStatusMaster{CaseStatusID: 1, CaseStatusName: "Under Investigation"})
	db.Create(&models.CaseStatusMaster{CaseStatusID: 2, CaseStatusName: "Charge Sheeted"})
	db.Create(&models.CaseStatusMaster{CaseStatusID: 3, CaseStatusName: "Closed"})

	// 13. Seed Court
	db.Create(&models.Court{CourtID: 1, CourtName: "1st ACMM Bengaluru", DistrictID: 1, StateID: 1, Active: true})

	// 14. Seed Demographics
	db.Create(&models.CasteMaster{CasteMasterID: 1, CasteMasterName: "General"})
	db.Create(&models.ReligionMaster{ReligionID: 1, ReligionName: "Hindu"})
	db.Create(&models.OccupationMaster{OccupationID: 1, OccupationName: "Business Officer"})

	// 15. Seed Acts
	db.Create(&models.Act{ActCode: "IPC", ActDescription: "Indian Penal Code", ShortName: "IPC", Active: true})

	// 16. Seed Sections
	db.Create(&models.Section{ActCode: "IPC", SectionCode: "380", SectionDescription: "Theft in dwelling house", Active: true})
	db.Create(&models.Section{ActCode: "IPC", SectionCode: "457", SectionDescription: "Lurking house trespass or house breaking by night", Active: true})

	log.Println("Database seeding completed successfully")
}
