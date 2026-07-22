package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/repositories"
)

type Neo4jClient struct {
	baseURL, username, password string
	http                        *http.Client
}

func NewNeo4jClient(baseURL, username, password string) *Neo4jClient {
	return &Neo4jClient{baseURL: strings.TrimRight(baseURL, "/"), username: username, password: password, http: &http.Client{Timeout: 20 * time.Second}}
}
func (c *Neo4jClient) Execute(ctx context.Context, statement string, parameters map[string]interface{}) error {
	payload := map[string]interface{}{"statements": []interface{}{map[string]interface{}{"statement": statement, "parameters": parameters}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/db/neo4j/tx/commit", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Neo4j returned HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if len(decoded.Errors) > 0 {
		return fmt.Errorf("Neo4j query failed: %s", decoded.Errors[0].Code)
	}
	return nil
}

type GraphSyncService struct {
	client *Neo4jClient
	cases  *repositories.CaseRepository
}

func NewGraphSyncService(client *Neo4jClient, cases *repositories.CaseRepository) *GraphSyncService {
	return &GraphSyncService{client: client, cases: cases}
}
func (s *GraphSyncService) SyncCase(ctx context.Context, caseID, unitID int) error {
	cm, err := s.cases.GetByIDForUnit(caseID, unitID)
	if err != nil {
		return err
	}
	if cm == nil {
		return errors.New("case not found")
	}
	accused := make([]map[string]interface{}, 0, len(cm.AccusedList))
	for _, person := range cm.AccusedList {
		accused = append(accused, map[string]interface{}{"id": person.AccusedMasterID, "name": person.AccusedName, "gender_id": person.GenderID, "person_code": person.PersonID})
	}
	parameters := map[string]interface{}{"case_id": cm.CaseMasterID, "crime_no": cm.CrimeNo, "unit_id": cm.PoliceStationID, "major_head_id": cm.CrimeMajorHeadID, "minor_head_id": cm.CrimeMinorHeadID, "registered_date": cm.CrimeRegisteredDate.Format("2006-01-02"), "accused": accused}
	statement := `MERGE (c:Case {case_master_id: $case_id}) SET c.crime_no=$crime_no, c.unit_id=$unit_id, c.major_head_id=$major_head_id, c.minor_head_id=$minor_head_id, c.registered_date=date($registered_date) WITH c UNWIND $accused AS a MERGE (p:Accused {accused_master_id: a.id}) SET p.name=a.name, p.gender_id=a.gender_id, p.person_code=a.person_code MERGE (p)-[:ACCUSED_IN]->(c)`
	return s.client.Execute(ctx, statement, parameters)
}
