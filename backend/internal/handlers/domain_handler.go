package handlers

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DomainHandler struct {
	repo *repositories.DomainRepository
}

func NewDomainHandler(repo *repositories.DomainRepository) *DomainHandler {
	return &DomainHandler{repo: repo}
}

func caseIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return 0, false
	}
	return id, true
}

func (h *DomainHandler) authorizeCase(c *gin.Context) (int, int, int, bool) {
	caseID, ok := caseIDParam(c)
	if !ok {
		return 0, 0, 0, false
	}
	claims, ok := claimsFromContext(c)
	if !ok {
		return 0, 0, 0, false
	}
	exists, err := h.repo.CaseInUnit(caseID, claims.UnitID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to authorize case access", err)
		return 0, 0, 0, false
	}
	if !exists {
		utils.SendError(c, http.StatusNotFound, "Case not found", "")
		return 0, 0, 0, false
	}
	return caseID, claims.EmployeeID, claims.RankHierarchy, true
}

func (h *DomainHandler) ListComplainants(c *gin.Context) {
	caseID, _, hierarchy, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListComplainants(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve complainants", err)
		return
	}
	if hierarchy > 5 {
		for i := range rows {
			rows[i].ComplainantName = "REDACTED"
			rows[i].CasteID = 0
			rows[i].ReligionID = 0
			rows[i].Caste = nil
			rows[i].Religion = nil
		}
	}
	utils.SendSuccess(c, http.StatusOK, "Complainants retrieved", rows)
}

func (h *DomainHandler) AddComplainant(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var row models.ComplainantDetails
	if err := c.ShouldBindJSON(&row); err != nil {
		utils.SendBadRequest(c, "Invalid complainant", err)
		return
	}
	row.ComplainantID = 0
	row.CaseMasterID = caseID
	if strings.TrimSpace(row.ComplainantName) == "" {
		utils.SendError(c, http.StatusBadRequest, "Complainant name is required", "")
		return
	}
	if err := h.repo.AddComplainant(&row); err != nil {
		utils.SendInternalServerError(c, "Failed to add complainant", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Complainant added", row)
}

type partyUpdateRequest struct {
	Name         *string `json:"name"`
	AgeYear      *int    `json:"age_year"`
	GenderID     *int    `json:"gender_id"`
	OccupationID *int    `json:"occupation_id"`
	ReligionID   *int    `json:"religion_id"`
	CasteID      *int    `json:"caste_id"`
	PersonCode   *string `json:"person_code"`
	VictimPolice *string `json:"victim_police"`
}

func parsePositiveParam(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid "+name, "")
		return 0, false
	}
	return id, true
}

func (h *DomainHandler) UpdateComplainant(c *gin.Context) { h.updateParty(c, "complainant") }
func (h *DomainHandler) UpdateVictim(c *gin.Context)      { h.updateParty(c, "victim") }
func (h *DomainHandler) UpdateAccused(c *gin.Context)     { h.updateParty(c, "accused") }
func (h *DomainHandler) updateParty(c *gin.Context, kind string) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	id, ok := parsePositiveParam(c, "party_id")
	if !ok {
		return
	}
	var req partyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid party update", err)
		return
	}
	values := map[string]interface{}{}
	if req.AgeYear != nil {
		if *req.AgeYear < 0 || *req.AgeYear > 125 {
			utils.SendError(c, http.StatusBadRequest, "age_year must be between 0 and 125", "")
			return
		}
		values["AgeYear"] = *req.AgeYear
	}
	if req.GenderID != nil {
		values["GenderID"] = *req.GenderID
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			utils.SendError(c, http.StatusBadRequest, "name cannot be empty", "")
			return
		}
		switch kind {
		case "complainant":
			values["ComplainantName"] = name
		case "victim":
			values["VictimName"] = name
		default:
			values["AccusedName"] = name
		}
	}
	var err error
	switch kind {
	case "complainant":
		if req.OccupationID != nil {
			values["OccupationID"] = *req.OccupationID
		}
		if req.ReligionID != nil {
			values["ReligionID"] = *req.ReligionID
		}
		if req.CasteID != nil {
			values["CasteID"] = *req.CasteID
		}
		err = h.repo.UpdateComplainant(caseID, id, values)
	case "victim":
		if req.VictimPolice != nil {
			values["VictimPolice"] = *req.VictimPolice
		}
		err = h.repo.UpdateVictim(caseID, id, values)
	default:
		if req.PersonCode != nil {
			values["PersonID"] = strings.TrimSpace(*req.PersonCode)
		}
		err = h.repo.UpdateAccused(caseID, id, values)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.SendError(c, http.StatusNotFound, "Party record not found", "")
		} else {
			utils.SendInternalServerError(c, "Failed to update party", err)
		}
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Party record updated", gin.H{"id": id})
}

func (h *DomainHandler) ListVictims(c *gin.Context) {
	caseID, _, hierarchy, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListVictims(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve victims", err)
		return
	}
	if hierarchy > 5 {
		for i := range rows {
			rows[i].VictimName = "REDACTED"
		}
	}
	utils.SendSuccess(c, http.StatusOK, "Victims retrieved", rows)
}

func (h *DomainHandler) AddVictim(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var row models.Victim
	if err := c.ShouldBindJSON(&row); err != nil {
		utils.SendBadRequest(c, "Invalid victim", err)
		return
	}
	row.VictimMasterID = 0
	row.CaseMasterID = caseID
	if strings.TrimSpace(row.VictimName) == "" {
		utils.SendError(c, http.StatusBadRequest, "Victim name is required", "")
		return
	}
	if err := h.repo.AddVictim(&row); err != nil {
		utils.SendInternalServerError(c, "Failed to add victim", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Victim added", row)
}

func (h *DomainHandler) ListAccused(c *gin.Context) {
	caseID, _, hierarchy, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListAccused(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve accused", err)
		return
	}
	if hierarchy > 5 {
		for i := range rows {
			rows[i].AccusedName = "REDACTED"
		}
	}
	utils.SendSuccess(c, http.StatusOK, "Accused retrieved", rows)
}

func (h *DomainHandler) AddAccused(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var row models.Accused
	if err := c.ShouldBindJSON(&row); err != nil {
		utils.SendBadRequest(c, "Invalid accused", err)
		return
	}
	row.AccusedMasterID = 0
	row.CaseMasterID = caseID
	if strings.TrimSpace(row.AccusedName) == "" {
		utils.SendError(c, http.StatusBadRequest, "Accused name is required", "")
		return
	}
	if err := h.repo.AddAccused(&row); err != nil {
		utils.SendInternalServerError(c, "Failed to add accused", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Accused added", row)
}

type arrestRequest struct {
	ArrestSurrenderTypeID     int    `json:"arrest_surrender_type_id" binding:"required"`
	ArrestSurrenderDate       string `json:"arrest_surrender_date" binding:"required"`
	ArrestSurrenderStateID    int    `json:"arrest_surrender_state_id" binding:"required"`
	ArrestSurrenderDistrictID int    `json:"arrest_surrender_district_id" binding:"required"`
	CourtID                   int    `json:"court_id" binding:"required"`
	AccusedMasterID           int    `json:"accused_master_id" binding:"required"`
	AccusedIDs                []int  `json:"accused_ids"`
	IsComplainantAccused      bool   `json:"is_complainant_accused"`
}

func (h *DomainHandler) AddArrest(c *gin.Context) {
	caseID, employeeID, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	claims, _ := claimsFromContext(c)
	var req arrestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid arrest/surrender event", err)
		return
	}
	date, err := time.Parse("2006-01-02", req.ArrestSurrenderDate)
	if err != nil {
		utils.SendBadRequest(c, "Use YYYY-MM-DD for arrest_surrender_date", err)
		return
	}
	row := models.ArrestSurrender{CaseMasterID: caseID, ArrestSurrenderTypeID: req.ArrestSurrenderTypeID, ArrestSurrenderDate: date, ArrestSurrenderStateId: req.ArrestSurrenderStateID, ArrestSurrenderDistrictId: req.ArrestSurrenderDistrictID, PoliceStationID: claims.UnitID, IOID: employeeID, CourtID: req.CourtID, AccusedMasterID: req.AccusedMasterID, IsAccused: true, IsComplainantAccused: req.IsComplainantAccused}
	ids := append(req.AccusedIDs, req.AccusedMasterID)
	if err := h.repo.AddArrest(&row, ids); err != nil {
		utils.SendBadRequest(c, "Failed to add arrest/surrender event", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Arrest/surrender event added", row)
}

func (h *DomainHandler) ListArrests(c *gin.Context) {
	caseID, _, hierarchy, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListArrests(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve arrests", err)
		return
	}
	if hierarchy > 5 {
		for i := range rows {
			if rows[i].Accused != nil {
				rows[i].Accused.AccusedName = "REDACTED"
			}
			for j := range rows[i].AccusedLinks {
				rows[i].AccusedLinks[j].AccusedName = "REDACTED"
			}
		}
	}
	utils.SendSuccess(c, http.StatusOK, "Arrest/surrender events retrieved", rows)
}

type chargesheetRequest struct {
	Date string `json:"cs_date" binding:"required"`
	Type string `json:"cs_type" binding:"required"`
}

func (h *DomainHandler) PutChargesheet(c *gin.Context) {
	caseID, employeeID, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var req chargesheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid chargesheet", err)
		return
	}
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		utils.SendBadRequest(c, "Use YYYY-MM-DD for cs_date", err)
		return
	}
	req.Type = strings.ToUpper(strings.TrimSpace(req.Type))
	if req.Type != "A" && req.Type != "B" && req.Type != "C" {
		utils.SendError(c, http.StatusBadRequest, "cs_type must be A, B or C", "")
		return
	}
	row := models.ChargesheetDetails{CaseMasterID: caseID, CsDate: date, CsType: req.Type, PolicePersonID: employeeID}
	if err := h.repo.UpsertChargesheet(&row); err != nil {
		utils.SendInternalServerError(c, "Failed to save chargesheet", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Chargesheet saved", row)
}

func (h *DomainHandler) GetChargesheet(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	row, err := h.repo.GetChargesheet(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve chargesheet", err)
		return
	}
	if row == nil {
		utils.SendError(c, http.StatusNotFound, "Chargesheet not found", "")
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Chargesheet retrieved", row)
}

func (h *DomainHandler) AddDocument(c *gin.Context) {
	caseID, employeeID, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var row models.CaseDocument
	if err := c.ShouldBindJSON(&row); err != nil {
		utils.SendBadRequest(c, "Invalid evidence document", err)
		return
	}
	row.DocumentID = 0
	row.CaseMasterID = caseID
	row.CreatedBy = employeeID
	row.CreatedAt = time.Now().UTC()
	if strings.TrimSpace(row.DocumentType) == "" || strings.TrimSpace(row.StorageURI) == "" {
		utils.SendError(c, http.StatusBadRequest, "document_type and storage_uri are required", "")
		return
	}
	if row.SHA256 != "" {
		decoded, err := hex.DecodeString(row.SHA256)
		if err != nil || len(decoded) != 32 {
			utils.SendError(c, http.StatusBadRequest, "sha256 must be a 64-character hexadecimal digest", "")
			return
		}
	}
	if err := h.repo.AddDocument(&row); err != nil {
		utils.SendInternalServerError(c, "Failed to add evidence document", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Evidence document registered", row)
}

func (h *DomainHandler) ListDocuments(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListDocuments(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve evidence documents", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Evidence documents retrieved", rows)
}

type statusRequest struct {
	CaseStatusID int `json:"case_status_id" binding:"required"`
}

func (h *DomainHandler) UpdateStatus(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	claims, _ := claimsFromContext(c)
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid status", err)
		return
	}
	if err := h.repo.UpdateCaseStatus(caseID, claims.UnitID, req.CaseStatusID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.SendError(c, http.StatusNotFound, "Case not found", "")
		} else {
			utils.SendInternalServerError(c, "Failed to update case status", err)
		}
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Case status updated", gin.H{"case_master_id": caseID, "case_status_id": req.CaseStatusID})
}

func (h *DomainHandler) ListActs(c *gin.Context) {
	rows, err := h.repo.ListActs(c.Query("keyword"))
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve acts", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Acts retrieved", rows)
}

func (h *DomainHandler) ListSections(c *gin.Context) {
	rows, err := h.repo.ListSections(c.Query("act_code"), c.Query("keyword"))
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve sections", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Sections retrieved", rows)
}

func (h *DomainHandler) CaseSections(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.CaseSections(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve case sections", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Case sections retrieved", rows)
}

type caseSectionRequest struct {
	ActID     string `json:"act_id" binding:"required"`
	SectionID string `json:"section_code" binding:"required"`
}

func (h *DomainHandler) AddCaseSection(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var req caseSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid legal section", err)
		return
	}
	row := &models.ActSectionAssociation{CaseMasterID: caseID, ActID: strings.TrimSpace(req.ActID), SectionID: strings.TrimSpace(req.SectionID)}
	if err := h.repo.AddCaseSection(row); err != nil {
		utils.SendBadRequest(c, "Failed to add legal section", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Legal section added", row)
}
func (h *DomainHandler) RemoveCaseSection(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	if err := h.repo.RemoveCaseSection(caseID, c.Query("act_id"), c.Query("section_code")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.SendError(c, http.StatusNotFound, "Legal section not found", "")
		} else {
			utils.SendInternalServerError(c, "Failed to remove legal section", err)
		}
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Legal section removed", gin.H{"case_master_id": caseID})
}

func (h *DomainHandler) ListUnitEmployees(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListUnitEmployees(claims.UnitID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to list unit employees", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Unit employees retrieved", rows)
}

type taskRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Priority       string `json:"priority"`
	Status         string `json:"status"`
	AssignedTo     int    `json:"assigned_to"`
	DueAt          string `json:"due_at"`
	CompletionNote string `json:"completion_note"`
}

func validChoice(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
func (h *DomainHandler) CreateTask(c *gin.Context) {
	caseID, actorID, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid investigation task", err)
		return
	}
	due, err := time.Parse(time.RFC3339, req.DueAt)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "due_at must be an RFC3339 timestamp", "")
		return
	}
	req.Priority = strings.ToLower(req.Priority)
	if !validChoice(req.Priority, "low", "medium", "high", "critical") {
		utils.SendError(c, http.StatusBadRequest, "Invalid priority", "")
		return
	}
	if strings.TrimSpace(req.Title) == "" || req.AssignedTo <= 0 {
		utils.SendError(c, http.StatusBadRequest, "title and assigned_to are required", "")
		return
	}
	row := &models.InvestigationTask{CaseMasterID: caseID, Title: strings.TrimSpace(req.Title), Description: strings.TrimSpace(req.Description), Priority: req.Priority, Status: "open", AssignedTo: req.AssignedTo, CreatedBy: actorID, DueAt: due, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := h.repo.CreateTask(row); err != nil {
		utils.SendBadRequest(c, "Failed to create investigation task", err)
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Investigation task created", row)
}
func (h *DomainHandler) ListCaseTasks(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListCaseTasks(caseID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to list tasks", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Investigation tasks retrieved", rows)
}
func (h *DomainHandler) ListUnitTasks(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	rows, err := h.repo.ListUnitTasks(claims.UnitID, c.Query("status"))
	if err != nil {
		utils.SendInternalServerError(c, "Failed to list unit tasks", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Unit investigation tasks retrieved", rows)
}
func (h *DomainHandler) UpdateTask(c *gin.Context) {
	caseID, actorID, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	taskID, ok := parsePositiveParam(c, "task_id")
	if !ok {
		return
	}
	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid task update", err)
		return
	}
	values := map[string]interface{}{"updated_at": time.Now().UTC()}
	if req.Title != "" {
		values["title"] = strings.TrimSpace(req.Title)
	}
	if req.Description != "" {
		values["description"] = strings.TrimSpace(req.Description)
	}
	if req.Priority != "" {
		req.Priority = strings.ToLower(req.Priority)
		if !validChoice(req.Priority, "low", "medium", "high", "critical") {
			utils.SendError(c, http.StatusBadRequest, "Invalid priority", "")
			return
		}
		values["priority"] = req.Priority
	}
	if req.Status != "" {
		req.Status = strings.ToLower(req.Status)
		if !validChoice(req.Status, "open", "in_progress", "blocked", "completed", "cancelled") {
			utils.SendError(c, http.StatusBadRequest, "Invalid status", "")
			return
		}
		values["status"] = req.Status
		if req.Status == "completed" {
			now := time.Now().UTC()
			values["completed_at"] = &now
			values["completion_note"] = req.CompletionNote
		}
	}
	if req.AssignedTo > 0 {
		values["assigned_to"] = req.AssignedTo
	}
	if req.DueAt != "" {
		due, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			utils.SendError(c, http.StatusBadRequest, "Invalid due_at", "")
			return
		}
		values["due_at"] = due
	}
	if err := h.repo.UpdateTask(caseID, taskID, actorID, values, req.CompletionNote); err != nil {
		utils.SendBadRequest(c, "Failed to update task", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Investigation task updated", gin.H{"task_id": taskID})
}
func (h *DomainHandler) ListTaskEvents(c *gin.Context) {
	caseID, _, _, ok := h.authorizeCase(c)
	if !ok {
		return
	}
	taskID, ok := parsePositiveParam(c, "task_id")
	if !ok {
		return
	}
	rows, err := h.repo.ListTaskEvents(caseID, taskID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to list task history", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Task history retrieved", rows)
}
