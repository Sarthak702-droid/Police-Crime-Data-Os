package services

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEmbeddingClient(t *testing.T) {
	client := NewEmbeddingClient("http://embedding.test")
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/embed" {
			t.Errorf("unexpected path %s", request.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[0.1,0.2,0.3]`)), Header: http.Header{}}, nil
	})
	vector, err := client.Embed(context.Background(), "query: burglary")
	if err != nil {
		t.Fatal(err)
	}
	if len(vector) != 3 || vector[1] != 0.2 {
		t.Fatalf("unexpected vector %#v", vector)
	}
}

func TestS3ObjectStoreSigning(t *testing.T) {
	store := NewS3ObjectStore("http://minio.test", "access-key", "secret-key", "evidence", "us-east-1")
	requests := 0
	store.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if !strings.Contains(request.Header.Get("Authorization"), "Credential=access-key/") {
			t.Error("missing SigV4 credential")
		}
		if strings.Contains(request.Header.Get("Authorization"), "secret-key") {
			t.Error("secret leaked in authorization header")
		}
		status := http.StatusOK
		if request.Method == http.MethodHead {
			status = http.StatusNotFound
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	})
	if err := store.Put(context.Background(), "cases/1/file.txt", "text/plain", []byte("evidence")); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("expected bucket check, bucket create and object put; got %d", requests)
	}
}

func TestNeo4jClient(t *testing.T) {
	client := NewNeo4jClient("http://neo4j.test", "neo4j", "password")
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		user, password, ok := request.BasicAuth()
		if !ok || user != "neo4j" || password != "password" {
			t.Error("missing Neo4j basic auth")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"results":[],"errors":[]}`)), Header: http.Header{}}, nil
	})
	if err := client.Execute(context.Background(), "RETURN $value", map[string]interface{}{"value": 1}); err != nil {
		t.Fatal(err)
	}
}

func TestSafeFilename(t *testing.T) {
	if got := safeFilename("../../unsafe report?.pdf"); got != "unsafe_report_.pdf" {
		t.Fatalf("unexpected safe filename %q", got)
	}
}
