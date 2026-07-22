package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type OPAClient struct {
	url  string
	http *http.Client
}

func NewOPAClient(url string) *OPAClient {
	return &OPAClient{url: url, http: &http.Client{Timeout: 3 * time.Second}}
}

func (c *OPAClient) Allow(ctx context.Context, input map[string]interface{}) (bool, error) {
	body, err := json.Marshal(map[string]interface{}{"input": input})
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, errors.New("OPA decision endpoint unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, err
	}
	var decoded struct {
		Result bool `json:"result"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return false, err
	}
	return decoded.Result, nil
}

func OPAAuthorizationMiddleware(client *OPAClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get("claims")
		claims, ok := raw.(*services.AuthClaims)
		if !exists || !ok {
			utils.SendError(c, http.StatusUnauthorized, "Unauthorized access", "")
			c.Abort()
			return
		}
		roles := append([]string{}, claims.RealmAccess.Roles...)
		if len(roles) == 0 {
			roles = append(roles, "investigator")
			if claims.RankHierarchy > 0 && claims.RankHierarchy <= 3 {
				roles = append(roles, "supervisor")
			}
			if strings.Contains(strings.ToLower(claims.Designation), "admin") {
				roles = append(roles, "admin")
			}
		}
		input := map[string]interface{}{"subject": map[string]interface{}{"active": true, "employee_id": claims.EmployeeID, "unit_id": claims.UnitID, "district_id": claims.DistrictID, "rank_hierarchy": claims.RankHierarchy, "roles": roles}, "resource": map[string]interface{}{"unit_id": claims.UnitID, "district_id": claims.DistrictID, "route": c.FullPath()}, "action": policyAction(c.Request.Method, c.FullPath())}
		allowed, err := client.Allow(c.Request.Context(), input)
		if err != nil {
			utils.SendError(c, http.StatusServiceUnavailable, "Authorization policy service unavailable", "")
			c.Abort()
			return
		}
		if !allowed {
			utils.SendError(c, http.StatusForbidden, "Access denied by policy", "")
			c.Abort()
			return
		}
		c.Next()
	}
}

func policyAction(method, path string) string {
	if strings.Contains(path, "/auth/register") {
		return "admin.manage"
	}
	if strings.Contains(path, "/documents") {
		if method == http.MethodGet {
			return "evidence.read"
		}
		return "evidence.write"
	}
	if strings.Contains(path, "/chat") || strings.Contains(path, "/ai/") {
		return "chat.query"
	}
	if strings.Contains(path, "/analytics") || strings.Contains(path, "/graph") {
		return "analytics.read"
	}
	if method == http.MethodGet {
		return "case.read"
	}
	return "case.write"
}
