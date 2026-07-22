package handlers

import (
	"net/http"
	"strconv"

	"backend/internal/repositories"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	repo  *repositories.AnalyticsRepository
	intel *services.IntelligenceService
}

func NewAnalyticsHandler(repo *repositories.AnalyticsRepository, intel *services.IntelligenceService) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo, intel: intel}
}

func (h *AnalyticsHandler) GetCaseReadiness(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || caseID <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return
	}
	result, err := h.intel.CaseReadiness(caseID, claims.UnitID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to assess case readiness", err)
		return
	}
	if result == nil {
		utils.SendError(c, http.StatusNotFound, "Case not found", "")
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Case readiness assessed", result)
}

func (h *AnalyticsHandler) GetSimilarCases(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || caseID <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	rows, err := h.intel.SimilarCases(caseID, claims.UnitID, limit)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to find similar cases", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Similar cases ranked", gin.H{"cases": rows, "method": "explainable weighted match over crime head, legal sections, narrative and distance"})
}

func (h *AnalyticsHandler) GetPendingActions(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("minimum_age_days", "30"))
	rows, err := h.intel.PendingActions(claims.UnitID, days)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to prioritize pending cases", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Pending investigation actions prioritized", gin.H{"cases": rows, "disclaimer": "Priority is advisory; supervisors retain decision authority."})
}

func (h *AnalyticsHandler) GetHotspots(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}

	hotspots, err := h.repo.GetBurglaryHotspotsForUnit(claims.UnitID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to compute burglary hotspots", err)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Burglary hotspots retrieved", hotspots)
}

func (h *AnalyticsHandler) GetSubgraph(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}

	accusedIDStr := c.Query("accused_id")
	if accusedIDStr == "" {
		utils.SendError(c, http.StatusBadRequest, "accused_id query parameter is required", "")
		return
	}

	accusedID, err := strconv.Atoi(accusedIDStr)
	if err != nil {
		utils.SendBadRequest(c, "Invalid accused_id parameter", err)
		return
	}

	nodes, edges, err := h.repo.GetCoaccusalGraphForUnit(accusedID, claims.UnitID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to generate co-accusal graph", err)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Criminal network graph retrieved", gin.H{
		"nodes": nodes,
		"edges": edges,
	})
}

func claimsFromContext(c *gin.Context) (*services.AuthClaims, bool) {
	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return nil, false
	}
	claims, ok := rawClaims.(*services.AuthClaims)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return nil, false
	}
	return claims, true
}
