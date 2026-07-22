package handlers

import (
	"net/http"
	"strconv"

	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type RetrievalHandler struct{ service *services.RetrievalService }

func NewRetrievalHandler(service *services.RetrievalService) *RetrievalHandler {
	return &RetrievalHandler{service: service}
}

func (h *RetrievalHandler) Search(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	rows, err := h.service.Search(c.Request.Context(), c.Query("q"), claims.UnitID, limit)
	if err != nil {
		utils.SendError(c, http.StatusBadGateway, "Hybrid search service unavailable", "")
		return
	}
	if claims.RankHierarchy > 5 {
		for i := range rows {
			rows[i].BriefFacts = "REDACTED FOR ROLE"
		}
	}
	utils.SendSuccess(c, http.StatusOK, "Hybrid search completed", rows)
}
func (h *RetrievalHandler) IndexCase(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || caseID <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return
	}
	if err := h.service.IndexCase(c.Request.Context(), caseID, claims.UnitID); err != nil {
		utils.SendError(c, http.StatusBadGateway, "Case indexing failed", "")
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Case indexed", gin.H{"case_master_id": caseID})
}
