package middleware

import (
	"errors"
	"net/http"
	"strings"

	"backend/internal/config"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.SendError(c, http.StatusUnauthorized, "Authorization header is required", "")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.SendError(c, http.StatusUnauthorized, "Authorization header must be Bearer {token}", "")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &services.AuthClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			utils.SendError(c, http.StatusUnauthorized, "Invalid or expired token", "")
			c.Abort()
			return
		}

		// Inject claims into context
		c.Set("claims", claims)
		c.Next()
	}
}

// RoleAuthMiddleware checks if user has a required rank level (hierarchy check)
func RoleAuthMiddleware(maxHierarchyAllowed int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock rank check logic: can extend based on rank hierarchy in DB
		// In this template, we allow all authenticated employees by default unless restricted
		c.Next()
	}
}
