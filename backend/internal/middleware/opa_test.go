package middleware

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOPAClientDecision(t *testing.T) {
	client := NewOPAClient("http://opa.test/v1/data/police/authz/allow")
	client.http.Transport = middlewareRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Errorf("unexpected method %s", request.Method)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"result":true}`)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})
	allowed, err := client.Allow(context.Background(), map[string]interface{}{"action": "case.read"})
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected allow decision")
	}
}

func TestPolicyActionMapping(t *testing.T) {
	if got := policyAction(http.MethodPost, "/api/v1/cases/:id/documents"); got != "evidence.write" {
		t.Fatalf("unexpected action %s", got)
	}
	if got := policyAction(http.MethodGet, "/api/v1/analytics/hotspots"); got != "analytics.read" {
		t.Fatalf("unexpected action %s", got)
	}
}

type middlewareRoundTrip func(*http.Request) (*http.Response, error)

func (fn middlewareRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
