package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const maxEvidenceUploadBytes = 25 << 20

type EvidenceHandler struct {
	service *services.EvidenceStorageService
}

func (h *EvidenceHandler) Download(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return
	}
	documentID, err := strconv.Atoi(c.Param("document_id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid document ID", "")
		return
	}
	row, data, err := h.service.Download(c.Request.Context(), caseID, documentID, claims.EmployeeID, claims.UnitID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Evidence not available", "")
		return
	}
	contentType := row.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	filename := strings.ReplaceAll(row.OriginalName, `"`, "")
	if filename == "" {
		filename = fmt.Sprintf("evidence-%d", documentID)
	}
	disposition := "attachment"
	if strings.HasPrefix(contentType, "image/") || contentType == "application/pdf" || strings.HasPrefix(contentType, "text/") {
		disposition = "inline"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, filename))
	c.Header("X-Content-SHA256", row.SHA256)
	c.Data(http.StatusOK, contentType, data)
}

type evidenceMetadataRequest struct {
	DocumentType string `json:"document_type"`
	LanguageCode string `json:"language_code"`
	PiiLevel     string `json:"pii_level"`
	Note         string `json:"note"`
}

func (h *EvidenceHandler) UpdateMetadata(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, _ := strconv.Atoi(c.Param("id"))
	documentID, _ := strconv.Atoi(c.Param("document_id"))
	var req evidenceMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendBadRequest(c, "Invalid evidence metadata", err)
		return
	}
	if req.PiiLevel != "" && req.PiiLevel != "public" && req.PiiLevel != "internal" && req.PiiLevel != "restricted" && req.PiiLevel != "highly_restricted" {
		utils.SendError(c, http.StatusBadRequest, "Invalid pii_level", "")
		return
	}
	if err := h.service.UpdateMetadata(caseID, documentID, claims.EmployeeID, claims.UnitID, strings.TrimSpace(req.DocumentType), strings.TrimSpace(req.LanguageCode), req.PiiLevel, strings.TrimSpace(req.Note)); err != nil {
		utils.SendBadRequest(c, "Failed to classify evidence", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Evidence metadata updated", gin.H{"document_id": documentID})
}

func (h *EvidenceHandler) Custody(c *gin.Context) {
	caseID, _ := strconv.Atoi(c.Param("id"))
	documentID, _ := strconv.Atoi(c.Param("document_id"))
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	allowed, err := h.service.Authorize(caseID, claims.UnitID)
	if err != nil || !allowed {
		utils.SendError(c, http.StatusNotFound, "Evidence not found", "")
		return
	}
	rows, err := h.service.Custody(caseID, documentID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to retrieve custody history", err)
		return
	}
	utils.SendSuccess(c, http.StatusOK, "Evidence custody history retrieved", rows)
}

func NewEvidenceHandler(service *services.EvidenceStorageService) *EvidenceHandler {
	return &EvidenceHandler{service: service}
}
func (h *EvidenceHandler) Upload(c *gin.Context) {
	claims, ok := claimsFromContext(c)
	if !ok {
		return
	}
	caseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || caseID <= 0 {
		utils.SendError(c, http.StatusBadRequest, "Invalid case ID", "")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Evidence file is required", "")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxEvidenceUploadBytes {
		utils.SendError(c, http.StatusRequestEntityTooLarge, "Evidence file must be between 1 byte and 25 MB", "")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Unable to read evidence file", "")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxEvidenceUploadBytes+1))
	if err != nil || len(data) > maxEvidenceUploadBytes {
		utils.SendError(c, http.StatusRequestEntityTooLarge, "Evidence file exceeds 25 MB", "")
		return
	}
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	documentType := strings.TrimSpace(c.PostForm("document_type"))
	if documentType == "" {
		documentType = "evidence"
	}
	row, err := h.service.Upload(c.Request.Context(), caseID, claims.EmployeeID, claims.UnitID, fileHeader.Filename, documentType, c.PostForm("language_code"), c.PostForm("pii_level"), contentType, data)
	if err != nil {
		utils.SendError(c, http.StatusBadGateway, "Evidence storage failed", "")
		return
	}
	utils.SendSuccess(c, http.StatusCreated, "Evidence uploaded and registered", row)
}
