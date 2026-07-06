package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"backend/internal/models"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	blacklistedIPs   = make(map[string]bool)
	blacklistedKGIDs = make(map[string]bool)
	blacklistMutex   sync.RWMutex
)

// Regex patterns for SQL injection and XSS
var (
	sqliRegex = regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|delete\s+from|drop\s+table|'\s*or\s*'\d+'\s*=\s*'\d+|--|' OR '1'='1|' OR 1=1)`)
	xssRegex  = regexp.MustCompile(`(?i)(<script|javascript:|onerror\s*=|onload\s*=)`)
)

func BlockIP(ip string) {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()
	blacklistedIPs[ip] = true
}

func BlockKGID(kgid string) {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()
	blacklistedKGIDs[kgid] = true
}

func IsIPBlocked(ip string) bool {
	blacklistMutex.RLock()
	defer blacklistMutex.RUnlock()
	return blacklistedIPs[ip]
}

func IsKGIDBlocked(kgid string) bool {
	blacklistMutex.RLock()
	defer blacklistMutex.RUnlock()
	return blacklistedKGIDs[kgid]
}

func ClearBlacklists() {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()
	blacklistedIPs = make(map[string]bool)
	blacklistedKGIDs = make(map[string]bool)
}

func DetectMaliciousPayload(input string) bool {
	if input == "" {
		return false
	}
	decoded, err := url.QueryUnescape(input)
	if err != nil {
		decoded = input
	}
	return sqliRegex.MatchString(decoded) || xssRegex.MatchString(decoded)
}

func IntrusionWafMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// 1. Check IP Blacklist
		if IsIPBlocked(clientIP) {
			utils.SendError(c, http.StatusForbidden, "Access permanently blocked due to security violations.", "")
			c.Abort()
			return
		}

		// 2. Check KGID Blacklist (if claims exist)
		var actor string = "unknown"
		if rawClaims, exists := c.Get("claims"); exists {
			if claims, ok := rawClaims.(*services.AuthClaims); ok {
				actor = claims.KGID
				if IsKGIDBlocked(claims.KGID) {
					utils.SendError(c, http.StatusForbidden, "Your account has been locked due to security violations.", "")
					c.Abort()
					return
				}
			}
		}

		// 3. Scan URL and Query Parameters
		if DetectMaliciousPayload(c.Request.URL.String()) {
			handleIntrusion(c, db, clientIP, actor, "Malicious URL/Query pattern detected")
			return
		}

		// Scan Headers
		for k, v := range c.Request.Header {
			headerVal := strings.Join(v, " ")
			if DetectMaliciousPayload(headerVal) {
				handleIntrusion(c, db, clientIP, actor, fmt.Sprintf("Malicious header payload in %s", k))
				return
			}
		}

		// Scan Body
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// Restore request body so handlers can read it
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				bodyStr := string(bodyBytes)
				if DetectMaliciousPayload(bodyStr) {
					handleIntrusion(c, db, clientIP, actor, "Malicious body payload detected")
					return
				}
			}
		}

		c.Next()
	}
}

func handleIntrusion(c *gin.Context, db *gorm.DB, ip string, actor string, reason string) {
	BlockIP(ip)

	if actor != "unknown" && actor != "" {
		BlockKGID(actor)
	}

	reqIDVal, _ := c.Get("request_id")
	traceIDVal, _ := c.Get("trace_id")
	reqID := fmt.Sprintf("%v", reqIDVal)
	traceID := fmt.Sprintf("%v", traceIDVal)

	event := models.AuditEvent{
		AuditID:   uuid.NewString(),
		Actor:     actor,
		Action:    "INTRUSION_BLOCKED",
		Resource:  fmt.Sprintf("IP=%s Reason=%s Status=403", ip, reason),
		CreatedAt: time.Now().UTC(),
		RequestID: reqID,
		TraceID:   traceID,
	}

	if db != nil {
		auditMutex.Lock()
		var lastEvent models.AuditEvent
		err := db.Order("created_at DESC, AuditID DESC").First(&lastEvent).Error
		if err == nil {
			event.BeforeHash = lastEvent.AfterHash
		} else {
			event.BeforeHash = "genesis_hash_placeholder"
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
		auditMutex.Unlock()
	}

	writeToAuditFile(&event)

	utils.SendError(c, http.StatusForbidden, "Request blocked due to security violation. Your IP and account have been locked.", "")
	c.Abort()
}
