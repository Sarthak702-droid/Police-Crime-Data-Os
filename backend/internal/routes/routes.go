package routes

import (
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
) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.IntrusionWafMiddleware(db))
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.CORSMiddleware(cfg))

	api := r.Group("/api")
	v1 := api.Group("/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", middleware.RateLimitMiddleware(5, 60*time.Second, "Too many login attempts. Please try again later."), authH.Login)
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/refresh", authH.Refresh)
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg), middleware.AuditMiddleware(db))
		{
			protected.GET("/auth/me", authH.Me)

			protected.POST("/cases", caseH.Create)
			protected.GET("/cases/search", caseH.Search)
			protected.GET("/cases/:id", caseH.GetByID)
			protected.GET("/cases/:id/timeline", caseH.GetTimeline)

			protected.GET("/analytics/hotspots", analyticsH.GetHotspots)
			protected.GET("/graph/subgraph", analyticsH.GetSubgraph)

			protected.POST("/chat/query", middleware.RateLimitMiddleware(20, 60*time.Second, "Too many chat requests. Please try again later."), chatH.Query)
			protected.GET("/chat/sessions", chatH.GetSessions)
			protected.GET("/chat/sessions/:session_id/turns", chatH.GetTurns)
			protected.POST("/chat/sessions/:session_id/export/pdf", chatH.ExportPDF)
		}
	}

	return r
}
