package repositories

import (
	"errors"

	"backend/internal/models"

	"gorm.io/gorm"
)

type PartyRepository struct {
	db *gorm.DB
}

func NewPartyRepository(db *gorm.DB) *PartyRepository {
	return &PartyRepository{db: db}
}

func (r *PartyRepository) AddVictim(v *models.Victim) error {
	return r.db.Create(v).Error
}

func (r *PartyRepository) AddAccused(a *models.Accused) error {
	return r.db.Create(a).Error
}

func (r *PartyRepository) AddComplainant(c *models.ComplainantDetails) error {
	return r.db.Create(c).Error
}

func (r *PartyRepository) AddArrest(arr *models.ArrestSurrender) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create main ArrestSurrender record
		if err := tx.Create(arr).Error; err != nil {
			return err
		}

		// 2. Create the bridging records in inv_arrestsurrenderaccused
		// for any linked accused
		for _, linkedAcc := range arr.AccusedLinks {
			ja := models.InvArrestSurrenderAccused{
				ArrestSurrenderID: arr.ArrestSurrenderID,
				AccusedMasterID:   linkedAcc.AccusedMasterID,
				IsPrimary:         linkedAcc.AccusedMasterID == arr.AccusedMasterID,
			}
			if err := tx.Create(&ja).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *PartyRepository) AddChargesheet(cs *models.ChargesheetDetails) error {
	return r.db.Create(cs).Error
}

func (r *PartyRepository) GetArrestsByCaseID(caseID int) ([]models.ArrestSurrender, error) {
	var arrests []models.ArrestSurrender
	err := r.db.Preload("InvestigatingOfficer").
		Preload("Accused").
		Preload("Court").
		Preload("PoliceStation").
		Preload("District").
		Preload("State").
		Where("CaseMasterID = ?", caseID).
		Find(&arrests).Error
	return arrests, err
}

func (r *PartyRepository) GetChargesheetByCaseID(caseID int) (*models.ChargesheetDetails, error) {
	var cs models.ChargesheetDetails
	err := r.db.Preload("PolicePerson").Where("CaseMasterID = ?", caseID).First(&cs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cs, nil
}
