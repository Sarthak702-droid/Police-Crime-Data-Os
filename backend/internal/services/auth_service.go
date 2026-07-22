package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo            *repositories.AuthRepository
	cfg             *config.Config
	lockoutAttempts map[string]int
	lockoutTime     map[string]time.Time
	lockoutMutex    sync.Mutex
}

type AuthClaims struct {
	EmployeeID    int    `json:"employee_id"`
	KGID          string `json:"kgid"`
	RankName      string `json:"rank"`
	RankHierarchy int    `json:"rank_hierarchy"`
	Designation   string `json:"designation"`
	UnitID        int    `json:"unit_id"`
	DistrictID    int    `json:"district_id"`
	RealmAccess   struct {
		Roles []string `json:"roles"`
	} `json:"realm_access,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthService(repo *repositories.AuthRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:            repo,
		cfg:             cfg,
		lockoutAttempts: make(map[string]int),
		lockoutTime:     make(map[string]time.Time),
	}
}

func generateSecureToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *AuthService) Login(kgid, password string) (string, string, *models.Employee, error) {
	s.lockoutMutex.Lock()
	if unlockTime, locked := s.lockoutTime[kgid]; locked {
		if time.Now().Before(unlockTime) {
			s.lockoutMutex.Unlock()
			timeLeft := time.Until(unlockTime).Round(time.Second)
			return "", "", nil, fmt.Errorf("account locked due to multiple failed login attempts, please try again in %v", timeLeft)
		}
		delete(s.lockoutTime, kgid)
		delete(s.lockoutAttempts, kgid)
	}
	s.lockoutMutex.Unlock()

	emp, err := s.repo.GetByKGID(kgid)
	if err != nil {
		return "", "", nil, err
	}
	if emp == nil {
		return "", "", nil, errors.New("invalid KGID or password")
	}

	creds, err := s.repo.GetCredentials(emp.EmployeeID)
	if err != nil {
		return "", "", nil, err
	}
	if creds == nil {
		return "", "", nil, errors.New("credentials not configured for this user")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(creds.PasswordHash), []byte(password)); err != nil {
		s.lockoutMutex.Lock()
		s.lockoutAttempts[kgid]++
		if s.lockoutAttempts[kgid] >= 5 {
			s.lockoutTime[kgid] = time.Now().Add(15 * time.Minute)
			s.lockoutMutex.Unlock()
			return "", "", nil, errors.New("account locked due to multiple failed login attempts, please try again in 15 minutes")
		}
		s.lockoutMutex.Unlock()
		return "", "", nil, errors.New("invalid KGID or password")
	}

	// Reset lockout attempts
	s.lockoutMutex.Lock()
	delete(s.lockoutAttempts, kgid)
	delete(s.lockoutTime, kgid)
	s.lockoutMutex.Unlock()

	// Generate JWT token
	expirationTime := time.Now().Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)
	claims := &AuthClaims{
		EmployeeID:    emp.EmployeeID,
		KGID:          emp.KGID,
		RankName:      emp.Rank.RankName,
		RankHierarchy: emp.Rank.Hierarchy,
		Designation:   emp.Designation.DesignationName,
		UnitID:        emp.UnitID,
		DistrictID:    emp.DistrictID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", nil, err
	}

	// Generate Refresh Token
	rfTokenStr := generateSecureToken()
	rt := &models.RefreshToken{
		EmployeeID: emp.EmployeeID,
		Token:      rfTokenStr,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateRefreshToken(rt); err != nil {
		return "", "", nil, err
	}

	return tokenString, rfTokenStr, emp, nil
}

func (s *AuthService) Refresh(refreshTokenStr string) (string, string, error) {
	if refreshTokenStr == "" {
		return "", "", errors.New("refresh token is required")
	}

	rt, err := s.repo.GetRefreshToken(refreshTokenStr)
	if err != nil {
		return "", "", err
	}
	if rt == nil {
		return "", "", errors.New("invalid refresh token")
	}

	if time.Now().After(rt.ExpiresAt) {
		_ = s.repo.DeleteRefreshToken(refreshTokenStr)
		return "", "", errors.New("refresh token expired")
	}

	emp, err := s.repo.GetEmployeeByID(rt.EmployeeID)
	if err != nil || emp == nil {
		return "", "", errors.New("employee not found")
	}

	_ = s.repo.DeleteRefreshToken(refreshTokenStr)

	expirationTime := time.Now().Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)
	claims := &AuthClaims{
		EmployeeID:    emp.EmployeeID,
		KGID:          emp.KGID,
		RankName:      emp.Rank.RankName,
		RankHierarchy: emp.Rank.Hierarchy,
		Designation:   emp.Designation.DesignationName,
		UnitID:        emp.UnitID,
		DistrictID:    emp.DistrictID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	newRfTokenStr := generateSecureToken()
	newRt := &models.RefreshToken{
		EmployeeID: emp.EmployeeID,
		Token:      newRfTokenStr,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateRefreshToken(newRt); err != nil {
		return "", "", err
	}

	return accessToken, newRfTokenStr, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)
	for _, char := range password {
		if char >= 'A' && char <= 'Z' {
			hasUpper = true
		} else if char >= 'a' && char <= 'z' {
			hasLower = true
		} else if char >= '0' && char <= '9' {
			hasDigit = true
		} else {
			hasSpecial = true
		}
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	return nil
}

func (s *AuthService) Register(emp *models.Employee, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}

	// Check if already exists
	existing, err := s.repo.GetByKGID(emp.KGID)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("employee with this KGID already exists")
	}

	// Create Employee record
	if err := s.repo.CreateEmployee(emp); err != nil {
		return err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Save credentials
	creds := &models.UserCredentials{
		EmployeeID:   emp.EmployeeID,
		PasswordHash: string(hashedPassword),
	}

	return s.repo.CreateCredentials(creds)
}
