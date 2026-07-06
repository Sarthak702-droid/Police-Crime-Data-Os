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
	repo *repositories.AnalyticsRepository
}

func NewAnalyticsHandler(repo *repositories.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo}
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
