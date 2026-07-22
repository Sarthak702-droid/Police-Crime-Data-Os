package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"
)

type EmbeddingClient struct {
	baseURL string
	http    *http.Client
}

func NewEmbeddingClient(baseURL string) *EmbeddingClient {
	return &EmbeddingClient{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *EmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]interface{}{"inputs": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding service returned HTTP %d", resp.StatusCode)
	}
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return embeddingVector(raw)
}

func embeddingVector(raw interface{}) ([]float64, error) {
	values, ok := raw.([]interface{})
	if !ok || len(values) == 0 {
		return nil, errors.New("invalid embedding response")
	}
	if nested, ok := values[0].([]interface{}); ok {
		values = nested
	}
	vector := make([]float64, len(values))
	for i, value := range values {
		number, ok := value.(float64)
		if !ok {
			return nil, errors.New("embedding contains non-numeric value")
		}
		vector[i] = number
	}
	if len(vector) == 0 {
		return nil, errors.New("empty embedding")
	}
	return vector, nil
}

type OpenSearchClient struct {
	baseURL, index, username, password string
	http                               *http.Client
}

func NewOpenSearchClient(baseURL, index, username, password string) *OpenSearchClient {
	return &OpenSearchClient{baseURL: strings.TrimRight(baseURL, "/"), index: index, username: username, password: password, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *OpenSearchClient) request(ctx context.Context, method, path string, payload interface{}) ([]byte, int, error) {
	var reader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes))
	return data, resp.StatusCode, err
}

func (c *OpenSearchClient) EnsureIndex(ctx context.Context, dimension int) error {
	_, status, err := c.request(ctx, http.MethodHead, "/"+url.PathEscape(c.index), nil)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		return nil
	}
	if status != http.StatusNotFound {
		return fmt.Errorf("OpenSearch index check returned HTTP %d", status)
	}
	mapping := map[string]interface{}{"settings": map[string]interface{}{"index": map[string]interface{}{"knn": true}}, "mappings": map[string]interface{}{"properties": map[string]interface{}{"case_master_id": map[string]interface{}{"type": "integer"}, "crime_no": map[string]interface{}{"type": "keyword"}, "police_station_id": map[string]interface{}{"type": "integer"}, "brief_facts": map[string]interface{}{"type": "text"}, "embedding": map[string]interface{}{"type": "knn_vector", "dimension": dimension}}}}
	_, status, err = c.request(ctx, http.MethodPut, "/"+url.PathEscape(c.index), mapping)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("OpenSearch index creation returned HTTP %d", status)
	}
	return nil
}

func (c *OpenSearchClient) IndexCase(ctx context.Context, cm *models.CaseMaster, embedding []float64) error {
	document := map[string]interface{}{"case_master_id": cm.CaseMasterID, "crime_no": cm.CrimeNo, "case_no": cm.CaseNo, "police_station_id": cm.PoliceStationID, "crime_major_head_id": cm.CrimeMajorHeadID, "crime_minor_head_id": cm.CrimeMinorHeadID, "registered_date": cm.CrimeRegisteredDate, "brief_facts": cm.BriefFacts, "embedding": embedding}
	path := "/" + url.PathEscape(c.index) + "/_doc/" + strconv.Itoa(cm.CaseMasterID) + "?refresh=true"
	_, status, err := c.request(ctx, http.MethodPut, path, document)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("OpenSearch indexing returned HTTP %d", status)
	}
	return nil
}

type HybridSearchHit struct {
	CaseMasterID int     `json:"case_master_id"`
	CrimeNo      string  `json:"crime_no"`
	BriefFacts   string  `json:"brief_facts"`
	Score        float64 `json:"score"`
}

func (c *OpenSearchClient) HybridSearch(ctx context.Context, query string, embedding []float64, unitID, limit int) ([]HybridSearchHit, error) {
	payload := map[string]interface{}{"size": limit, "_source": []string{"case_master_id", "crime_no", "brief_facts"}, "query": map[string]interface{}{"bool": map[string]interface{}{"filter": []interface{}{map[string]interface{}{"term": map[string]interface{}{"police_station_id": unitID}}}, "should": []interface{}{map[string]interface{}{"multi_match": map[string]interface{}{"query": query, "fields": []string{"brief_facts^2", "crime_no", "case_no"}}}, map[string]interface{}{"knn": map[string]interface{}{"embedding": map[string]interface{}{"vector": embedding, "k": limit}}}}, "minimum_should_match": 1}}}
	data, status, err := c.request(ctx, http.MethodPost, "/"+url.PathEscape(c.index)+"/_search", payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("OpenSearch search returned HTTP %d", status)
	}
	var decoded struct {
		Hits struct {
			Hits []struct {
				Score  float64 `json:"_score"`
				Source struct {
					CaseMasterID int    `json:"case_master_id"`
					CrimeNo      string `json:"crime_no"`
					BriefFacts   string `json:"brief_facts"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	rows := make([]HybridSearchHit, 0, len(decoded.Hits.Hits))
	for _, hit := range decoded.Hits.Hits {
		rows = append(rows, HybridSearchHit{CaseMasterID: hit.Source.CaseMasterID, CrimeNo: hit.Source.CrimeNo, BriefFacts: hit.Source.BriefFacts, Score: hit.Score})
	}
	return rows, nil
}

type RetrievalService struct {
	embeddings *EmbeddingClient
	search     *OpenSearchClient
	cases      *repositories.CaseRepository
}

func NewRetrievalService(embeddings *EmbeddingClient, search *OpenSearchClient, cases *repositories.CaseRepository) *RetrievalService {
	return &RetrievalService{embeddings: embeddings, search: search, cases: cases}
}
func (s *RetrievalService) IndexCase(ctx context.Context, caseID, unitID int) error {
	cm, err := s.cases.GetByIDForUnit(caseID, unitID)
	if err != nil {
		return err
	}
	if cm == nil {
		return errors.New("case not found")
	}
	vector, err := s.embeddings.Embed(ctx, "passage: "+cm.BriefFacts)
	if err != nil {
		return err
	}
	if err := s.search.EnsureIndex(ctx, len(vector)); err != nil {
		return err
	}
	return s.search.IndexCase(ctx, cm, vector)
}
func (s *RetrievalService) Search(ctx context.Context, query string, unitID, limit int) ([]HybridSearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	vector, err := s.embeddings.Embed(ctx, "query: "+query)
	if err != nil {
		return nil, err
	}
	return s.search.HybridSearch(ctx, query, vector, unitID, limit)
}
