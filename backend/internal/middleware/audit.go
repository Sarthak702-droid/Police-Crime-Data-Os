package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"backend/internal/models"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var auditMutex sync.Mutex

func AuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if db == nil || !shouldAudit(c.Request.Method, c.FullPath()) {
			return
		}

		actor := "unknown"
		if rawClaims, exists := c.Get("claims"); exists {
			if claims, ok := rawClaims.(*services.AuthClaims); ok {
				actor = claims.KGID
			}
		}

		reqIDVal, _ := c.Get("request_id")
		traceIDVal, _ := c.Get("trace_id")

		reqID := fmt.Sprintf("%v", reqIDVal)
		traceID := fmt.Sprintf("%v", traceIDVal)

		event := models.AuditEvent{
			AuditID:   uuid.NewString(),
			Actor:     actor,
			Action:    c.Request.Method,
			Resource:  fmt.Sprintf("%s status=%d", c.FullPath(), c.Writer.Status()),
			CreatedAt: time.Now().UTC(),
			RequestID: reqID,
			TraceID:   traceID,
		}

		auditMutex.Lock()
		defer auditMutex.Unlock()

		var lastEvent models.AuditEvent
		err := db.Order("created_at DESC, AuditID DESC").First(&lastEvent).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				event.BeforeHash = "genesis_hash_placeholder"
			} else {
				event.BeforeHash = "error_fetching_previous_hash"
			}
		} else {
			event.BeforeHash = lastEvent.AfterHash
		}

		dataToHash := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
			event.AuditID,
			event.Actor,
			event.Action,
			event.Resource,
			event.BeforeHash,
			event.CreatedAt.Format(time.RFC3339Nano),
			event.RequestID,
			event.TraceID,
		)
		hash := sha256.Sum256([]byte(dataToHash))
		event.AfterHash = hex.EncodeToString(hash[:])

		_ = db.Create(&event).Error

		writeToAuditFile(&event)
	}
}

func writeToAuditFile(event *models.AuditEvent) {
	f, err := os.OpenFile("audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	logBytes, err := json.Marshal(event)
	if err == nil {
		_, _ = f.Write(append(logBytes, '\n'))
	}
}

func shouldAudit(method, path string) bool {
	if path == "" {
		return false
	}
	if method != "GET" {
		return true
	}
	return strings.Contains(path, "/cases") || strings.Contains(path, "/chat") || strings.Contains(path, "/analytics") || strings.Contains(path, "/graph")
}
