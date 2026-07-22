package repositories

import (
	"errors"
	"strings"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

// DomainRepository exposes the ER-backed operations that do not belong to the
// case aggregate writer: parties, legal classification, custody and evidence.
type DomainRepository struct {
	db *gorm.DB
}

func NewDomainRepository(db *gorm.DB) *DomainRepository { return &DomainRepository{db: db} }

func (r *DomainRepository) CaseInUnit(caseID, unitID int) (bool, error) {
	var count int64
	err := r.db.Model(&models.CaseMaster{}).
		Where("CaseMasterID = ? AND PoliceStationID = ?", caseID, unitID).Count(&count).Error
	return count == 1, err
}

func (r *DomainRepository) UpdateCaseStatus(caseID, unitID, statusID int) error {
	result := r.db.Model(&models.CaseMaster{}).
		Where("CaseMasterID = ? AND PoliceStationID = ?", caseID, unitID).
		Update("CaseStatusID", statusID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *DomainRepository) ListComplainants(caseID int) ([]models.ComplainantDetails, error) {
	var rows []models.ComplainantDetails
	err := r.db.Preload("Occupation").Preload("Religion").Preload("Caste").
		Where("CaseMasterID = ?", caseID).Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) ListVictims(caseID int) ([]models.Victim, error) {
	var rows []models.Victim
	err := r.db.Where("CaseMasterID = ?", caseID).Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) ListAccused(caseID int) ([]models.Accused, error) {
	var rows []models.Accused
	err := r.db.Where("CaseMasterID = ?", caseID).Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) AddComplainant(row *models.ComplainantDetails) error {
	return r.db.Create(row).Error
}

func (r *DomainRepository) AddVictim(row *models.Victim) error   { return r.db.Create(row).Error }
func (r *DomainRepository) AddAccused(row *models.Accused) error { return r.db.Create(row).Error }

func (r *DomainRepository) UpdateComplainant(caseID, id int, values map[string]interface{}) error {
	return updateScoped(r.db, &models.ComplainantDetails{}, "ComplainantID", id, caseID, values)
}
func (r *DomainRepository) UpdateVictim(caseID, id int, values map[string]interface{}) error {
	return updateScoped(r.db, &models.Victim{}, "VictimMasterID", id, caseID, values)
}
func (r *DomainRepository) UpdateAccused(caseID, id int, values map[string]interface{}) error {
	return updateScoped(r.db, &models.Accused{}, "AccusedMasterID", id, caseID, values)
}
func updateScoped(db *gorm.DB, model interface{}, idColumn string, id, caseID int, values map[string]interface{}) error {
	result := db.Model(model).Where(idColumn+" = ? AND CaseMasterID = ?", id, caseID).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *DomainRepository) AddArrest(row *models.ArrestSurrender, accusedIDs []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if row.AccusedMasterID != 0 {
			var count int64
			if err := tx.Model(&models.Accused{}).
				Where("AccusedMasterID = ? AND CaseMasterID = ?", row.AccusedMasterID, row.CaseMasterID).
				Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("primary accused does not belong to this case")
			}
		}
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		seen := map[int]bool{}
		for _, accusedID := range accusedIDs {
			if accusedID <= 0 || seen[accusedID] {
				continue
			}
			seen[accusedID] = true
			var count int64
			if err := tx.Model(&models.Accused{}).
				Where("AccusedMasterID = ? AND CaseMasterID = ?", accusedID, row.CaseMasterID).
				Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("linked accused does not belong to this case")
			}
			link := models.InvArrestSurrenderAccused{
				ArrestSurrenderID: row.ArrestSurrenderID,
				AccusedMasterID:   accusedID,
				IsPrimary:         accusedID == row.AccusedMasterID,
			}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DomainRepository) ListArrests(caseID int) ([]models.ArrestSurrender, error) {
	var rows []models.ArrestSurrender
	err := r.db.Preload("State").Preload("District").Preload("PoliceStation").
		Preload("InvestigatingOfficer").Preload("Court").Preload("Accused").Preload("AccusedLinks").
		Where("CaseMasterID = ?", caseID).Order("ArrestSurrenderDate DESC").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) UpsertChargesheet(row *models.ChargesheetDetails) error {
	var existing models.ChargesheetDetails
	err := r.db.Where("CaseMasterID = ?", row.CaseMasterID).First(&existing).Error
	if err == nil {
		row.CSID = existing.CSID
		return r.db.Save(row).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.Create(row).Error
}

func (r *DomainRepository) GetChargesheet(caseID int) (*models.ChargesheetDetails, error) {
	var row models.ChargesheetDetails
	err := r.db.Preload("PolicePerson").Where("CaseMasterID = ?", caseID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *DomainRepository) AddDocument(row *models.CaseDocument) error { return r.db.Create(row).Error }

func (r *DomainRepository) ListDocuments(caseID int) ([]models.CaseDocument, error) {
	var rows []models.CaseDocument
	err := r.db.Where("CaseMasterID = ?", caseID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) GetDocument(caseID, documentID int) (*models.CaseDocument, error) {
	var row models.CaseDocument
	err := r.db.Where("CaseMasterID = ? AND DocumentID = ?", caseID, documentID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *DomainRepository) UpdateDocument(caseID, documentID int, values map[string]interface{}) error {
	result := r.db.Model(&models.CaseDocument{}).Where("CaseMasterID = ? AND DocumentID = ?", caseID, documentID).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *DomainRepository) AddCustodyEvent(row *models.EvidenceCustodyEvent) error {
	return r.db.Create(row).Error
}
func (r *DomainRepository) ListCustodyEvents(caseID, documentID int) ([]models.EvidenceCustodyEvent, error) {
	var rows []models.EvidenceCustodyEvent
	err := r.db.Preload("Actor").Where("CaseMasterID = ? AND DocumentID = ?", caseID, documentID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) ListActs(keyword string) ([]models.Act, error) {
	var rows []models.Act
	query := r.db.Where("Active = ?", true)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("ActCode LIKE ? OR ActDescription LIKE ? OR ShortName LIKE ?", like, like, like)
	}
	err := query.Order("ActCode").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) ListSections(actCode, keyword string) ([]models.Section, error) {
	var rows []models.Section
	query := r.db.Where("Active = ?", true)
	if actCode = strings.TrimSpace(actCode); actCode != "" {
		query = query.Where("ActCode = ?", actCode)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("SectionCode LIKE ? OR SectionDescription LIKE ?", like, like)
	}
	err := query.Order("ActCode, SectionCode").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) CaseSections(caseID int) ([]models.ActSectionAssociation, error) {
	var rows []models.ActSectionAssociation
	err := r.db.Preload("Act").Where("CaseMasterID = ?", caseID).
		Order("ActOrderID, SectionOrderID").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) AddCaseSection(row *models.ActSectionAssociation) error {
	return r.db.Create(row).Error
}
func (r *DomainRepository) RemoveCaseSection(caseID int, actID, sectionID string) error {
	result := r.db.Where("CaseMasterID = ? AND ActID = ? AND SectionID = ?", caseID, actID, sectionID).Delete(&models.ActSectionAssociation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *DomainRepository) ListUnitEmployees(unitID int) ([]models.Employee, error) {
	var rows []models.Employee
	err := r.db.Preload("Rank").Preload("Designation").Where("UnitID = ?", unitID).Order("FirstName").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) CreateTask(row *models.InvestigationTask) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.Employee{}).Where("EmployeeID = ? AND UnitID = (SELECT PoliceStationID FROM CaseMaster WHERE CaseMasterID = ?)", row.AssignedTo, row.CaseMasterID).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return errors.New("assignee must belong to the case unit")
		}
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return tx.Create(&models.InvestigationTaskEvent{TaskID: row.TaskID, ActorID: row.CreatedBy, Action: "created", ToStatus: row.Status, Note: row.Description, CreatedAt: time.Now().UTC()}).Error
	})
}

func (r *DomainRepository) ListCaseTasks(caseID int) ([]models.InvestigationTask, error) {
	var rows []models.InvestigationTask
	err := r.db.Preload("Assignee").Preload("Creator").Where("CaseMasterID = ?", caseID).Order("CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END, due_at").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) ListUnitTasks(unitID int, status string) ([]models.InvestigationTask, error) {
	var rows []models.InvestigationTask
	query := r.db.Preload("Assignee").Preload("Creator").Joins("JOIN CaseMaster ON CaseMaster.CaseMasterID = InvestigationTask.CaseMasterID").Where("CaseMaster.PoliceStationID = ?", unitID)
	if status != "" {
		query = query.Where("InvestigationTask.status = ?", status)
	}
	err := query.Order("InvestigationTask.due_at").Find(&rows).Error
	return rows, err
}

func (r *DomainRepository) UpdateTask(caseID, taskID, actorID int, values map[string]interface{}, note string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var task models.InvestigationTask
		if err := tx.Where("TaskID = ? AND CaseMasterID = ?", taskID, caseID).First(&task).Error; err != nil {
			return err
		}
		from := task.Status
		if assignee, ok := values["assigned_to"]; ok {
			var count int64
			if err := tx.Model(&models.Employee{}).Where("EmployeeID = ? AND UnitID = (SELECT PoliceStationID FROM CaseMaster WHERE CaseMasterID = ?)", assignee, caseID).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("assignee must belong to the case unit")
			}
		}
		if err := tx.Model(&task).Updates(values).Error; err != nil {
			return err
		}
		to := from
		if value, ok := values["status"].(string); ok {
			to = value
		}
		return tx.Create(&models.InvestigationTaskEvent{TaskID: taskID, ActorID: actorID, Action: "updated", FromStatus: from, ToStatus: to, Note: note, CreatedAt: time.Now().UTC()}).Error
	})
}

func (r *DomainRepository) ListTaskEvents(caseID, taskID int) ([]models.InvestigationTaskEvent, error) {
	var rows []models.InvestigationTaskEvent
	err := r.db.Preload("Actor").Joins("JOIN InvestigationTask ON InvestigationTask.TaskID = InvestigationTaskEvent.TaskID").Where("InvestigationTaskEvent.TaskID = ? AND InvestigationTask.CaseMasterID = ?", taskID, caseID).Order("InvestigationTaskEvent.created_at DESC").Find(&rows).Error
	return rows, err
}
