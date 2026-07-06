package handlers

import (
	"net/http"
	"time"

	"backend/internal/models"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type LoginRequest struct {
	KGID     string `json:"kgid" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid input data", err)
		return
	}

	token, refreshToken, emp, err := h.service.Login(req.KGID, req.Password)
	if err != nil {
		utils.SendError(c, http.StatusUnauthorized, err.Error(), "")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Login successful", gin.H{
		"token":         token,
		"refresh_token": refreshToken,
		"employee":      emp,
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid input data", err)
		return
	}

	token, refreshToken, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		utils.SendError(c, http.StatusUnauthorized, err.Error(), "")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Token refreshed successfully", gin.H{
		"token":         token,
		"refresh_token": refreshToken,
	})
}

type RegisterRequest struct {
	KGID                 string `json:"kgid" binding:"required"`
	Password             string `json:"password" binding:"required"`
	FirstName            string `json:"first_name" binding:"required"`
	EmployeeDOB          string `json:"employee_dob" binding:"required"` // Format: YYYY-MM-DD
	GenderID             int    `json:"gender_id" binding:"required"`
	BloodGroupID         int    `json:"blood_group_id" binding:"required"`
	PhysicallyChallenged bool   `json:"physically_challenged"`
	AppointmentDate      string `json:"appointment_date" binding:"required"` // Format: YYYY-MM-DD
	DistrictID           int    `json:"district_id" binding:"required"`
	UnitID               int    `json:"unit_id" binding:"required"`
	RankID               int    `json:"rank_id" binding:"required"`
	DesignationID        int    `json:"designation_id" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid input data", err)
		return
	}

	dob, err := time.Parse("2006-01-02", req.EmployeeDOB)
	if err != nil {
		utils.SendBadRequest(c, "Invalid employee_dob format. Use YYYY-MM-DD", err)
		return
	}

	appDate, err := time.Parse("2006-01-02", req.AppointmentDate)
	if err != nil {
		utils.SendBadRequest(c, "Invalid appointment_date format. Use YYYY-MM-DD", err)
		return
	}

	emp := &models.Employee{
		KGID:                 req.KGID,
		FirstName:            req.FirstName,
		EmployeeDOB:          dob,
		GenderID:             req.GenderID,
		BloodGroupID:         req.BloodGroupID,
		PhysicallyChallenged: req.PhysicallyChallenged,
		AppointmentDate:      appDate,
		DistrictID:           req.DistrictID,
		UnitID:               req.UnitID,
		RankID:               req.RankID,
		DesignationID:        req.DesignationID,
	}

	if err := h.service.Register(emp, req.Password); err != nil {
		utils.SendBadRequest(c, err.Error(), err)
		return
	}

	utils.SendSuccess(c, http.StatusCreated, "Employee registered successfully", emp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	// Retrieve employee claims from context (populated by JWT middleware)
	claims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "User context not found", "")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "User profile retrieved", claims)
}
