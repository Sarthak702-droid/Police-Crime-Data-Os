package repositories

import (
	"errors"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CaseRepository struct {
	db *gorm.DB
}

func NewCaseRepository(db *gorm.DB) *CaseRepository {
	return &CaseRepository{db: db}
}

func (r *CaseRepository) Create(c *models.CaseMaster) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("OccuranceTime", "Complainants", "Victims", "AccusedList", "ActsAssociated").Create(c).Error; err != nil {
			return err
		}

		if c.OccuranceTime != nil {
			c.OccuranceTime.CaseMasterID = c.CaseMasterID
			if err := tx.Create(c.OccuranceTime).Error; err != nil {
				return err
			}
		}

		for i := range c.Complainants {
			c.Complainants[i].CaseMasterID = c.CaseMasterID
			if err := tx.Create(&c.Complainants[i]).Error; err != nil {
				return err
			}
		}

		for i := range c.Victims {
			c.Victims[i].CaseMasterID = c.CaseMasterID
			if err := tx.Create(&c.Victims[i]).Error; err != nil {
				return err
			}
		}

		for i := range c.AccusedList {
			c.AccusedList[i].CaseMasterID = c.CaseMasterID
			if err := tx.Create(&c.AccusedList[i]).Error; err != nil {
				return err
			}
		}

		for i := range c.ActsAssociated {
			c.ActsAssociated[i].CaseMasterID = c.CaseMasterID
			if err := tx.Create(&c.ActsAssociated[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *CaseRepository) GetByID(id int) (*models.CaseMaster, error) {
	return r.getByID(id, 0)
}

func (r *CaseRepository) GetByIDForUnit(id int, unitID int) (*models.CaseMaster, error) {
	return r.getByID(id, unitID)
}

func (r *CaseRepository) getByID(id int, unitID int) (*models.CaseMaster, error) {
	var cm models.CaseMaster
	query := r.db.Preload("PolicePerson.Rank").
		Preload("PoliceStation").
		Preload("CaseCategory").
		Preload("GravityOffence").
		Preload("CrimeHead").
		Preload("CrimeSubHead").
		Preload("CaseStatus").
		Preload("Court").
		Preload("Complainants.Occupation").
		Preload("Complainants.Religion").
		Preload("Complainants.Caste").
		Preload("Victims").
		Preload("AccusedList").
		Preload("Arrests.InvestigatingOfficer").
		Preload("Arrests.Accused").
		Preload("ActsAssociated.Act").
		Preload("Chargesheet").
		Preload("OccuranceTime").
		Where("CaseMasterID = ?", id)

	if unitID > 0 {
		query = query.Where("PoliceStationID = ?", unitID)
	}

	err := query.First(&cm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cm, nil
}

type SearchFilters struct {
	CrimeHeadID     *int
	PoliceStationID *int
	FromDate        *time.Time
	ToDate          *time.Time
	StatusID        *int
	GravityID       *int
	CasteID         *int
	ReligionID      *int
	Keyword         string
	Limit           int
	Offset          int
	ScopeUnitID     int
}

func (r *CaseRepository) Search(f SearchFilters) ([]models.CaseMaster, int64, error) {
	var cases []models.CaseMaster
	var total int64

	query := r.db.Model(&models.CaseMaster{})

	if f.ScopeUnitID > 0 {
		query = query.Where("PoliceStationID = ?", f.ScopeUnitID)
	}

	if f.CasteID != nil || f.ReligionID != nil {
		query = query.Joins("JOIN ComplainantDetails ON ComplainantDetails.CaseMasterID = CaseMaster.CaseMasterID")
		if f.CasteID != nil {
			query = query.Where("ComplainantDetails.CasteID = ?", *f.CasteID)
		}
		if f.ReligionID != nil {
			query = query.Where("ComplainantDetails.ReligionID = ?", *f.ReligionID)
		}
	}

	if f.CrimeHeadID != nil {
		query = query.Where("CrimeMajorHeadID = ?", *f.CrimeHeadID)
	}
	if f.PoliceStationID != nil {
		query = query.Where("PoliceStationID = ?", *f.PoliceStationID)
	}
	if f.StatusID != nil {
		query = query.Where("CaseStatusID = ?", *f.StatusID)
	}
	if f.GravityID != nil {
		query = query.Where("GravityOffenceID = ?", *f.GravityID)
	}
	if f.FromDate != nil {
		query = query.Where("CrimeRegisteredDate >= ?", *f.FromDate)
	}
	if f.ToDate != nil {
		query = query.Where("CrimeRegisteredDate <= ?", *f.ToDate)
	}
	if f.Keyword != "" {
		query = query.Where("CrimeNo LIKE ? OR CaseNo LIKE ? OR BriefFacts LIKE ?", "%"+f.Keyword+"%", "%"+f.Keyword+"%", "%"+f.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("PoliceStation").
		Preload("CrimeHead").
		Preload("CrimeSubHead").
		Preload("CaseStatus").
		Preload("GravityOffence")

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	err := query.Order("CrimeRegisteredDate DESC").
		Limit(limit).
		Offset(offset).
		Find(&cases).Error

	return cases, total, err
}

func (r *CaseRepository) GetLastSerialNo(policeStationID int, caseCategoryID int, year int) (int, error) {
	var count int64
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	err := r.db.Model(&models.CaseMaster{}).
		Where("PoliceStationID = ? AND CaseCategoryID = ? AND CrimeRegisteredDate >= ? AND CrimeRegisteredDate < ?",
			policeStationID, caseCategoryID, startDate, endDate).
		Count(&count).Error

	return int(count), err
}

// AllocateSerial increments and returns the station/category/year sequence in
// one database statement. This prevents concurrent FIR registrations from
// receiving the same number, which a COUNT-based allocator cannot guarantee.
func (r *CaseRepository) AllocateSerial(policeStationID int, caseCategoryID int, year int) (int, error) {
	sequence := models.FIRSequence{
		PoliceStationID: policeStationID,
		CaseCategoryID:  caseCategoryID,
		Year:            year,
		CurrentSerial:   1,
		UpdatedAt:       time.Now().UTC(),
	}

	err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "PoliceStationID"},
			{Name: "CaseCategoryID"},
			{Name: "Year"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"CurrentSerial": gorm.Expr("CurrentSerial + 1"),
			"UpdatedAt":     time.Now().UTC(),
		}),
	}).Create(&sequence).Error
	if err != nil {
		return 0, err
	}

	err = r.db.Where("PoliceStationID = ? AND CaseCategoryID = ? AND Year = ?", policeStationID, caseCategoryID, year).
		First(&sequence).Error
	if err != nil {
		return 0, err
	}
	return sequence.CurrentSerial, nil
}
