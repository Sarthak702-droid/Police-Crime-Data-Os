package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"

	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	service *services.ChatService
}

func NewChatHandler(service *services.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

type QueryRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Language  string `json:"language"`
}

type TranslateRequest struct {
	Input              string `json:"input" binding:"required"`
	SourceLanguageCode string `json:"source_language_code"`
	TargetLanguageCode string `json:"target_language_code" binding:"required"`
}

func (h *ChatHandler) Translate(c *gin.Context) {
	var req TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid translation request", err)
		return
	}
	result, err := h.service.Translate(c.Request.Context(), req.Input, req.SourceLanguageCode, req.TargetLanguageCode)
	if err != nil {
		utils.SendError(c, http.StatusBadGateway, "Translation service unavailable", "")
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Text translated", result)
}

func (h *ChatHandler) SpeechToText(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Audio file is required", "")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > 10<<20 {
		utils.SendError(c, http.StatusRequestEntityTooLarge, "Audio file must be between 1 byte and 10 MB", "")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Unable to read audio file", "")
		return
	}
	defer file.Close()
	audio, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil || len(audio) > 10<<20 {
		utils.SendError(c, http.StatusRequestEntityTooLarge, "Audio file exceeds 10 MB", "")
		return
	}
	result, err := h.service.Transcribe(c.Request.Context(), fileHeader.Filename, audio, c.PostForm("language_code"), c.PostForm("mode"))
	if err != nil {
		utils.SendError(c, http.StatusBadGateway, "Speech service unavailable", "")
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Speech transcribed", result)
}

func (h *ChatHandler) Query(c *gin.Context) {
	// Retrieve employee claims context
	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid input data", err)
		return
	}

	lang := req.Language
	if lang == "" {
		lang = "en-IN"
	}

	response, err := h.service.ProcessQuery(c.Request.Context(), req.SessionID, claims.EmployeeID, claims.UnitID, claims.RankHierarchy, req.Message, lang)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to process chat query", err)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Query processed successfully", response)
}

func (h *ChatHandler) GetSessions(c *gin.Context) {
	// Retrieve employee claims context
	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	sessions, err := h.service.GetSessions(claims.EmployeeID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve sessions", err)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Chat sessions retrieved", sessions)
}

func (h *ChatHandler) GetTurns(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.SendError(c, http.StatusBadRequest, "session_id parameter is required", "")
		return
	}

	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	turns, err := h.service.GetHistoryForUser(sessionID, claims.EmployeeID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve conversation history", err)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Conversation turns retrieved", turns)
}

func (h *ChatHandler) ExportPDF(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.SendError(c, http.StatusBadRequest, "session_id parameter is required", "")
		return
	}

	rawClaims, exists := c.Get("claims")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
		return
	}
	claims := rawClaims.(*services.AuthClaims)

	turns, err := h.service.GetHistoryForUser(sessionID, claims.EmployeeID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to load history for export", err)
		return
	}

	pdfName := "conversation_export_" + sessionID + ".pdf"
	lines := []string{
		"Session: " + sessionID,
		"Exported at: " + time.Now().UTC().Format(time.RFC3339),
		"Officer: " + claims.KGID,
		"",
	}
	for _, turn := range turns {
		lines = append(lines, fmt.Sprintf("[%s] %s", turn.Speaker, turn.CreatedAt.UTC().Format(time.RFC3339)), turn.Content, "")
	}
	pdf := utils.BuildTextPDF("Crime Analytics Conversation Export", lines)
	digest := fmt.Sprintf("%x", sha256.Sum256(pdf))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, pdfName))
	c.Header("X-Content-SHA256", digest)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func (h *ChatHandler) GetEvidenceTrails(c *gin.Context) {
	sessionID := c.Param("session_id")
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	rows, err := h.service.GetEvidenceTrailsForUser(sessionID, claims.EmployeeID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve evidence trails", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Evidence trails retrieved", rows)
}

func (h *ChatHandler) GetToolCatalog(c *gin.Context) {
	utils.SendSuccess(c, http.StatusOK, "AI tool allowlist retrieved", services.AvailableAITools())
}
