package repositories

import (
	"errors"

	"backend/internal/models"

	"gorm.io/gorm"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) GetByKGID(kgid string) (*models.Employee, error) {
	var emp models.Employee
	err := r.db.Preload("Rank").Preload("Designation").Preload("Unit").Preload("District").
		Where("KGID = ?", kgid).First(&emp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &emp, nil
}

func (r *AuthRepository) GetCredentials(empID int) (*models.UserCredentials, error) {
	var creds models.UserCredentials
	err := r.db.Where("EmployeeID = ?", empID).First(&creds).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &creds, nil
}

func (r *AuthRepository) CreateCredentials(creds *models.UserCredentials) error {
	return r.db.Create(creds).Error
}

func (r *AuthRepository) CreateEmployee(emp *models.Employee) error {
	return r.db.Create(emp).Error
}

func (r *AuthRepository) CreateRefreshToken(rt *models.RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *AuthRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.Where("token = ?", token).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

func (r *AuthRepository) DeleteRefreshToken(token string) error {
	return r.db.Where("token = ?", token).Delete(&models.RefreshToken{}).Error
}

func (r *AuthRepository) GetEmployeeByID(empID int) (*models.Employee, error) {
	var emp models.Employee
	err := r.db.Preload("Rank").Preload("Designation").Preload("Unit").Preload("District").
		Where("EmployeeID = ?", empID).First(&emp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &emp, nil
}
