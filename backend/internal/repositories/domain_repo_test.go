package repositories

import (
	"testing"
	"time"

	"backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInvestigationTaskLifecycleAndUnitScoping(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Employee{}, &models.CaseMaster{}, &models.InvestigationTask{}, &models.InvestigationTaskEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Employee{EmployeeID: 1, UnitID: 7, KGID: "IO1", FirstName: "IO One"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Employee{EmployeeID: 2, UnitID: 8, KGID: "IO2", FirstName: "Wrong Unit"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CaseMaster{CaseMasterID: 11, PoliceStationID: 7, CrimeNo: "7/2026", CaseNo: "7", CaseCategoryID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewDomainRepository(db)
	task := &models.InvestigationTask{CaseMasterID: 11, Title: "Collect CCTV", Priority: "high", Status: "open", AssignedTo: 1, CreatedBy: 1, DueAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateTask(task); err != nil {
		t.Fatal(err)
	}
	if task.TaskID == 0 {
		t.Fatal("expected task id")
	}
	if err := repo.UpdateTask(11, task.TaskID, 1, map[string]interface{}{"status": "completed"}, "CCTV collected"); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListTaskEvents(11, task.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 history events, got %d", len(events))
	}
	wrong := &models.InvestigationTask{CaseMasterID: 11, Title: "Invalid", Priority: "low", Status: "open", AssignedTo: 2, CreatedBy: 1, DueAt: time.Now().Add(time.Hour)}
	if err := repo.CreateTask(wrong); err == nil {
		t.Fatal("expected cross-unit assignment to fail")
	}
}

func TestPartyUpdatesAreCaseScoped(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&models.ComplainantDetails{})
	repo := NewDomainRepository(db)
	row := &models.ComplainantDetails{CaseMasterID: 9, ComplainantName: "Original"}
	_ = db.Create(row).Error
	if err := repo.UpdateComplainant(10, row.ComplainantID, map[string]interface{}{"ComplainantName": "Wrong"}); err == nil {
		t.Fatal("expected cross-case update to fail")
	}
	if err := repo.UpdateComplainant(9, row.ComplainantID, map[string]interface{}{"ComplainantName": "Updated"}); err != nil {
		t.Fatal(err)
	}
}
