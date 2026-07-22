package routes

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/handlers"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(
	cfg *config.Config,
	db *gorm.DB,
	authH *handlers.AuthHandler,
	caseH *handlers.CaseHandler,
	chatH *handlers.ChatHandler,
	analyticsH *handlers.AnalyticsHandler,
	domainH *handlers.DomainHandler,
	retrievalH *handlers.RetrievalHandler,
	evidenceH *handlers.EvidenceHandler,
	graphSyncH *handlers.GraphSyncHandler,
) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.IntrusionWafMiddleware(db))
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.CORSMiddleware(cfg))

	// Serve the production React bundle from the same origin as the API when it
	// is available. This makes http://localhost:<port> the complete application
	// while Vite on :5173 remains available for hot-reload development.
	if frontendDir := findFrontendDist(); frontendDir != "" {
		r.Static("/assets", filepath.Join(frontendDir, "assets"))
		r.GET("/", func(c *gin.Context) { c.File(filepath.Join(frontendDir, "index.html")) })
		r.NoRoute(func(c *gin.Context) {
			if c.Request.Method == http.MethodGet && !strings.HasPrefix(c.Request.URL.Path, "/api/") && filepath.Ext(c.Request.URL.Path) == "" {
				c.File(filepath.Join(frontendDir, "index.html"))
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Route not found"})
		})
	} else {
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Crime Intelligence API is running", "api": "/api/v1", "frontend": "Run npm run build in frontend or use http://localhost:5173"})
		})
	}

	api := r.Group("/api")
	v1 := api.Group("/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", middleware.RateLimitMiddleware(5, 60*time.Second, "Too many login attempts. Please try again later."), authH.Login)
			authGroup.POST("/refresh", authH.Refresh)
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		if cfg.OPAURL != "" {
			protected.Use(middleware.OPAAuthorizationMiddleware(middleware.NewOPAClient(cfg.OPAURL)))
		}
		protected.Use(middleware.AuditMiddleware(db))
		{
			protected.GET("/auth/me", authH.Me)
			protected.POST("/auth/register", middleware.RoleAuthMiddleware(5), authH.Register)

			protected.POST("/cases", caseH.Create)
			protected.GET("/cases/search", caseH.Search)
			protected.GET("/cases/:id", caseH.GetByID)
			protected.GET("/cases/:id/timeline", caseH.GetTimeline)
			protected.PATCH("/cases/:id/status", domainH.UpdateStatus)
			protected.GET("/cases/:id/complainants", domainH.ListComplainants)
			protected.POST("/cases/:id/complainants", domainH.AddComplainant)
			protected.PATCH("/cases/:id/complainants/:party_id", domainH.UpdateComplainant)
			protected.GET("/cases/:id/victims", domainH.ListVictims)
			protected.POST("/cases/:id/victims", domainH.AddVictim)
			protected.PATCH("/cases/:id/victims/:party_id", domainH.UpdateVictim)
			protected.GET("/cases/:id/accused", domainH.ListAccused)
			protected.POST("/cases/:id/accused", domainH.AddAccused)
			protected.PATCH("/cases/:id/accused/:party_id", domainH.UpdateAccused)
			protected.GET("/cases/:id/arrests", domainH.ListArrests)
			protected.POST("/cases/:id/arrests", domainH.AddArrest)
			protected.GET("/cases/:id/chargesheet", domainH.GetChargesheet)
			protected.PUT("/cases/:id/chargesheet", domainH.PutChargesheet)
			protected.GET("/cases/:id/documents", domainH.ListDocuments)
			protected.POST("/cases/:id/documents", domainH.AddDocument)
			if evidenceH != nil {
				protected.POST("/cases/:id/evidence/upload", evidenceH.Upload)
				protected.GET("/cases/:id/evidence/:document_id/content", evidenceH.Download)
				protected.PATCH("/cases/:id/evidence/:document_id", evidenceH.UpdateMetadata)
				protected.GET("/cases/:id/evidence/:document_id/custody", evidenceH.Custody)
			}
			protected.GET("/cases/:id/sections", domainH.CaseSections)
			protected.POST("/cases/:id/sections", domainH.AddCaseSection)
			protected.DELETE("/cases/:id/sections", domainH.RemoveCaseSection)
			protected.GET("/cases/:id/tasks", domainH.ListCaseTasks)
			protected.POST("/cases/:id/tasks", domainH.CreateTask)
			protected.PATCH("/cases/:id/tasks/:task_id", domainH.UpdateTask)
			protected.GET("/cases/:id/tasks/:task_id/events", domainH.ListTaskEvents)
			protected.GET("/investigation/tasks", domainH.ListUnitTasks)
			protected.GET("/unit/employees", domainH.ListUnitEmployees)
			protected.GET("/acts", domainH.ListActs)
			protected.GET("/sections", domainH.ListSections)
			if retrievalH != nil {
				protected.GET("/search/hybrid", retrievalH.Search)
				protected.POST("/search/cases/:id/index", middleware.RoleAuthMiddleware(5), retrievalH.IndexCase)
			}

			protected.GET("/analytics/hotspots", analyticsH.GetHotspots)
			protected.GET("/analytics/cases/:id/readiness", analyticsH.GetCaseReadiness)
			protected.GET("/analytics/cases/:id/similar", analyticsH.GetSimilarCases)
			protected.GET("/analytics/pending-actions", analyticsH.GetPendingActions)
			protected.GET("/graph/subgraph", analyticsH.GetSubgraph)
			if graphSyncH != nil {
				protected.POST("/graph/cases/:id/sync", middleware.RoleAuthMiddleware(5), graphSyncH.SyncCase)
			}

			protected.POST("/chat/query", middleware.RateLimitMiddleware(20, 60*time.Second, "Too many chat requests. Please try again later."), chatH.Query)
			protected.GET("/ai/tools", chatH.GetToolCatalog)
			protected.POST("/ai/translate", middleware.RateLimitMiddleware(30, 60*time.Second, "Too many translation requests. Please try again later."), chatH.Translate)
			protected.POST("/ai/speech-to-text", middleware.RateLimitMiddleware(10, 60*time.Second, "Too many speech requests. Please try again later."), chatH.SpeechToText)
			protected.GET("/chat/sessions", chatH.GetSessions)
			protected.GET("/chat/sessions/:session_id/turns", chatH.GetTurns)
			protected.GET("/chat/sessions/:session_id/evidence-trails", chatH.GetEvidenceTrails)
			protected.POST("/chat/sessions/:session_id/export/pdf", chatH.ExportPDF)
		}
	}

	return r
}

func findFrontendDist() string {
	candidates := []string{"../frontend/dist", "frontend/dist"}
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(filepath.Join(absolute, "index.html")); err == nil && !info.IsDir() {
			return absolute
		}
	}
	return ""
}
