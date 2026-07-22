package handlers

import (
	"net/http"
	"strconv"

	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type GraphSyncHandler struct{ service *services.GraphSyncService }

func NewGraphSyncHandler(service *services.GraphSyncService) *GraphSyncHandler {
	return &GraphSyncHandler{service: service}
}
func (h *GraphSyncHandler) SyncCase(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || caseID <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return
	}
	if err := h.service.SyncCase(c.Request.Context(), caseID, claims.UnitID); err != nil {
		utils.SendError(c, http.StatusBadGateway, "Graph synchronization failed", "")
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Case graph synchronized", gin.H{"case_master_id": caseID})
}
