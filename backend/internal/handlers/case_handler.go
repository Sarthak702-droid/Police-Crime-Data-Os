package handlers

import (
	"net/http"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type CaseHandler struct {
	service *services.CaseService
}

func NewCaseHandler(service *services.CaseService) *CaseHandler {
	return &CaseHandler{service: service}
}

type CreateCaseRequest struct {
	CaseCategoryID     int                            `json:"case_category_id" binding:"required"`
	GravityOffenceID   int                            `json:"gravity_offence_id" binding:"required"`
	CrimeMajorHeadID   int                            `json:"crime_major_head_id" binding:"required"`
	CrimeMinorHeadID   int                            `json:"crime_minor_head_id" binding:"required"`
	CourtID            int                            `json:"court_id" binding:"required"`
	IncidentFromDate   string                         `json:"incident_from_date" binding:"required"`    // Format: YYYY-MM-DD HH:MM:SS
	IncidentToDate     string                         `json:"incident_to_date" binding:"required"`      // Format: YYYY-MM-DD HH:MM:SS
	InfoReceivedPSDate string                         `json:"info_received_ps_date" binding:"required"` // Format: YYYY-MM-DD HH:MM:SS
	Latitude           float64                        `json:"latitude"`
	Longitude          float64                        `json:"longitude"`
	BriefFacts         string                         `json:"brief_facts" binding:"required"`
	OccuranceTime      *models.Inv_OccuranceTime      `json:"occurance_time,omitempty"`
	Complainants       []models.ComplainantDetails    `json:"complainants,omitempty"`
	Victims            []models.Victim                `json:"victims,omitempty"`
	AccusedList        []models.Accused               `json:"accused_list,omitempty"`
	ActsAssociated     []models.ActSectionAssociation `json:"acts_associated,omitempty"`
}

func (h *CaseHandler) Create(c *gin.Context) {
	// Retrieve employee claims context
	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	var req CreateCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid input data", err)
		return
	}

	fromTime, err := time.Parse("2006-01-02 15:04:05", req.IncidentFromDate)
	if err != nil {
		utils.SendBadRequest(c, "Invalid incident_from_date format. Use YYYY-MM-DD HH:MM:SS", err)
		return
	}
	toTime, err := time.Parse("2006-01-02 15:04:05", req.IncidentToDate)
	if err != nil {
		utils.SendBadRequest(c, "Invalid incident_to_date format. Use YYYY-MM-DD HH:MM:SS", err)
		return
	}
	receivedTime, err := time.Parse("2006-01-02 15:04:05", req.InfoReceivedPSDate)
	if err != nil {
		utils.SendBadRequest(c, "Invalid info_received_ps_date format. Use YYYY-MM-DD HH:MM:SS", err)
		return
	}

	caseModel := &models.CaseMaster{
		CrimeRegisteredDate: time.Now(),
		PolicePersonID:      claims.EmployeeID,
		PoliceStationID:     claims.UnitID, // Set to registering officer's unit posting
		CaseCategoryID:      req.CaseCategoryID,
		GravityOffenceID:    req.GravityOffenceID,
		CrimeMajorHeadID:    req.CrimeMajorHeadID,
		CrimeMinorHeadID:    req.CrimeMinorHeadID,
		CourtID:             req.CourtID,
		IncidentFromDate:    fromTime,
		IncidentToDate:      toTime,
		InfoReceivedPSDate:  receivedTime,
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
		BriefFacts:          req.BriefFacts,
		OccuranceTime:       req.OccuranceTime,
		Complainants:        req.Complainants,
		Victims:             req.Victims,
		AccusedList:         req.AccusedList,
		ActsAssociated:      req.ActsAssociated,
	}

	// Propagate times to spatio-temporal occurrence if present
	if caseModel.OccuranceTime != nil {
		caseModel.OccuranceTime.IncidentFromTs = fromTime
		caseModel.OccuranceTime.IncidentToTs = toTime
		caseModel.OccuranceTime.InfoReceivedPSTs = receivedTime
		if caseModel.OccuranceTime.Latitude == 0 {
			caseModel.OccuranceTime.Latitude = req.Latitude
		}
		if caseModel.OccuranceTime.Longitude == 0 {
			caseModel.OccuranceTime.Longitude = req.Longitude
		}
	}

	if err := h.service.CreateCase(caseModel, claims.DistrictID); err != nil {
		utils.SendInternalServerError(c, "Failed to create case", err)
		return
	}

	utils.SendSuccess(c, http.StatusCreated, "Case created successfully", caseModel)
}

func (h *CaseHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendBadRequest(c, "Invalid case ID", err)
		return
	}

	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	cm, err := h.service.GetCaseByIDForUnit(id, claims.UnitID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve case details", err)
		return
	}
	if cm == nil {
		utils.SendError(c, http.StatusNotFound, "Case not found", "")
		return
	}

	redactCase(cm, claims.RankHierarchy)

	utils.SendSuccess(c, http.StatusOK, "Case details retrieved", cm)
}

func (h *CaseHandler) Search(c *gin.Context) {
	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	var filters repositories.SearchFilters
	filters.ScopeUnitID = claims.UnitID

	if val, ok := c.GetQuery("crime_head_id"); ok {
		if id, err := strconv.Atoi(val); err == nil {
			filters.CrimeHeadID = &id
		}
	}
	if val, ok := c.GetQuery("police_station_id"); ok {
		if id, err := strconv.Atoi(val); err == nil {
			filters.PoliceStationID = &id
		}
	}
	if val, ok := c.GetQuery("status_id"); ok {
		if id, err := strconv.Atoi(val); err == nil {
			filters.StatusID = &id
		}
	}
	if val, ok := c.GetQuery("gravity_id"); ok {
		if id, err := strconv.Atoi(val); err == nil {
			filters.GravityID = &id
		}
	}
	if val, ok := c.GetQuery("caste_id"); ok {
		if id, err := strconv.Atoi(val); err == nil {
			filters.CasteID = &id
		}
	}
	if val, ok := c.GetQuery("religion_id"); ok {
		if id, err := strconv.Atoi(val); err == nil {
			filters.ReligionID = &id
		}
	}
	if val, ok := c.GetQuery("from_date"); ok {
		if t, err := time.Parse("2006-01-02", val); err == nil {
			filters.FromDate = &t
		}
	}
	if val, ok := c.GetQuery("to_date"); ok {
		if t, err := time.Parse("2006-01-02", val); err == nil {
			filters.ToDate = &t
		}
	}

	filters.Keyword = c.Query("keyword")

	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit > 100 {
		limit = 100
	}
	filters.Limit = limit

	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	filters.Offset = (page - 1) * limit

	cases, total, err := h.service.SearchCases(filters)
	if err != nil {
		utils.SendInternalServerError(c, "Search query failed", err)
		return
	}

	for i := range cases {
		redactCase(&cases[i], claims.RankHierarchy)
	}

	utils.SendSuccess(c, http.StatusOK, "Search completed", gin.H{
		"cases": cases,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *CaseHandler) GetTimeline(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendBadRequest(c, "Invalid case ID", err)
		return
	}

	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	timeline, err := h.service.GetTimelineForUnit(id, claims.UnitID, claims.RankHierarchy)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to compile timeline", err)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Case timeline retrieved", timeline)
}

func redactCase(cm *models.CaseMaster, rankHierarchy int) {
	if cm == nil {
		return
	}
	if rankHierarchy > 5 { // e.g. Constable (9) vs Inspector (5)
		for i := range cm.Complainants {
			cm.Complainants[i].ComplainantName = "REDACTED"
			cm.Complainants[i].CasteID = 0
			cm.Complainants[i].ReligionID = 0
			if cm.Complainants[i].Caste != nil {
				cm.Complainants[i].Caste.CasteMasterName = "REDACTED"
			}
			if cm.Complainants[i].Religion != nil {
				cm.Complainants[i].Religion.ReligionName = "REDACTED"
			}
		}
		for i := range cm.Victims {
			cm.Victims[i].VictimName = "REDACTED"
		}
		for i := range cm.AccusedList {
			cm.AccusedList[i].AccusedName = "REDACTED"
		}
		for i := range cm.Arrests {
			if cm.Arrests[i].Accused != nil {
				cm.Arrests[i].Accused.AccusedName = "REDACTED"
			}
			for j := range cm.Arrests[i].AccusedLinks {
				cm.Arrests[i].AccusedLinks[j].AccusedName = "REDACTED"
			}
		}
	}
}
