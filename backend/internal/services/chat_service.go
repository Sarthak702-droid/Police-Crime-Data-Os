package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/google/uuid"
)

type ChatService struct {
	chatRepo      *repositories.ChatRepository
	caseRepo      *repositories.CaseRepository
	analyticsRepo *repositories.AnalyticsRepository
	orchestrator  *GeminiOrchestrator
}

func (s *ChatService) SetOrchestrator(orchestrator *GeminiOrchestrator) {
	s.orchestrator = orchestrator
}

type ChatResponse struct {
	Answer          string      `json:"answer"`
	Language        string      `json:"language"`
	Citations       interface{} `json:"citations,omitempty"`
	EvidenceTrailID string      `json:"evidence_trail_id"`
	Confidence      float64     `json:"confidence"`
}

func NewChatService(chatRepo *repositories.ChatRepository, caseRepo *repositories.CaseRepository, analyticsRepo *repositories.AnalyticsRepository) *ChatService {
	return &ChatService{
		chatRepo:      chatRepo,
		caseRepo:      caseRepo,
		analyticsRepo: analyticsRepo,
	}
}

func (s *ChatService) ProcessQuery(ctx context.Context, sessionID string, userID int, unitID int, rankHierarchy int, message string, language string) (*ChatResponse, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(message) == "" {
		return nil, errors.New("session and message are required")
	}

	session, err := s.chatRepo.GetSessionByID(sessionID)
	if err != nil {
		return nil, err
	}
	if session != nil && session.UserID != userID {
		return nil, errors.New("chat session not found")
	}
	if session == nil {
		session = &models.ConversationSession{
			SessionID:   sessionID,
			UserID:      userID,
			ContextJSON: "{}",
		}
		if err := s.chatRepo.CreateSession(session); err != nil {
			return nil, err
		}
	}

	userTurnID := fmt.Sprintf("turn_%d_u", time.Now().UnixNano())
	userTurn := &models.ConversationTurn{
		TurnID:    userTurnID,
		SessionID: sessionID,
		Speaker:   "user",
		Content:   message,
	}
	if err := s.chatRepo.AddTurn(userTurn); err != nil {
		return nil, err
	}
	if s.orchestrator != nil {
		result, orchestrationErr := s.orchestrator.Process(ctx, message, language, unitID, rankHierarchy)
		if orchestrationErr == nil {
			return s.persistAssistantResponse(sessionID, userID, unitID, rankHierarchy, result.Answer, result.Language, result.Citations, result.ToolName, result.Arguments, result.ResultIDs, result.Confidence, result.ModelName)
		}
	}

	messageLower := strings.ToLower(message)
	var answer string
	var citationsObj interface{}
	toolName := "help"
	toolArguments := map[string]interface{}{}
	resultIDs := []interface{}{}
	confidence := 0.55

	hotspotRegex := regexp.MustCompile(`(hotspot|crime rates|burglary)`)
	offenderRegex := regexp.MustCompile(`(repeat offender|criminals|recidiv)`)
	caseRegex := regexp.MustCompile(`(case|fir|crime no|crime number)\s*[:#\s]*\s*([0-9a-zA-Z]+)`)
	networkRegex := regexp.MustCompile(`(network|connection|graph|co-accused)\s*for\s*accused\s*([0-9]+)`)

	if networkRegex.MatchString(messageLower) {
		toolName = "get_coaccusal_network"
		matches := networkRegex.FindStringSubmatch(messageLower)
		accusedID, _ := strconv.Atoi(matches[2])
		toolArguments["accused_id"] = accusedID
		nodes, edges, err := s.analyticsRepo.GetCoaccusalGraphForUnit(accusedID, unitID)
		if err != nil {
			answer = fmt.Sprintf("I could not retrieve a scoped co-accusal graph for accused ID %d.", accusedID)
		} else {
			confidence = 0.9
			if rankHierarchy > 5 {
				for i := range nodes {
					if nodes[i].Type == "Person" {
						nodes[i].Label = "REDACTED"
					}
				}
			}
			answer = fmt.Sprintf("Retrieved co-accusal network for accused ID %d within your station scope. Found %d related nodes and %d connections.", accusedID, len(nodes), len(edges))
			citationsObj = map[string]interface{}{
				"graph_nodes": nodes,
				"graph_edges": edges,
			}
			for _, node := range nodes {
				resultIDs = append(resultIDs, node.ID)
			}
		}
	} else if caseRegex.MatchString(messageLower) {
		toolName = "lookup_case"
		matches := caseRegex.FindStringSubmatch(messageLower)
		crimeNo := matches[2]
		toolArguments["query"] = crimeNo
		filters := repositories.SearchFilters{Keyword: crimeNo, Limit: 1, ScopeUnitID: unitID}
		cases, _, err := s.caseRepo.Search(filters)
		if err != nil || len(cases) == 0 {
			answer = fmt.Sprintf("I could not find a case matching Crime Number or ID %s within your station scope.", crimeNo)
		} else {
			targetCase := cases[0]
			confidence = 0.95
			briefFacts := targetCase.BriefFacts
			if rankHierarchy > 5 {
				briefFacts = "REDACTED FOR ROLE"
			}
			answer = fmt.Sprintf("Found Case %s registered on %s. Major Head: %s. Minor Head: %s. Details: %s",
				targetCase.CrimeNo,
				targetCase.CrimeRegisteredDate.Format("02-Jan-2006"),
				targetCase.CrimeHead.CrimeGroupName,
				targetCase.CrimeSubHead.CrimeHeadName,
				briefFacts,
			)
			citationsObj = map[string]interface{}{
				"case_id":             targetCase.CaseMasterID,
				"crime_no":            targetCase.CrimeNo,
				"case_no":             targetCase.CaseNo,
				"registered_date":     targetCase.CrimeRegisteredDate,
				"brief_facts":         briefFacts,
				"police_station_name": targetCase.PoliceStation.UnitName,
			}
			resultIDs = append(resultIDs, targetCase.CaseMasterID)
		}
	} else if hotspotRegex.MatchString(messageLower) {
		toolName = "get_hotspots"
		toolArguments["days"] = 90
		hotspots, err := s.analyticsRepo.GetBurglaryHotspotsForUnit(unitID)
		if err != nil {
			answer = "I could not retrieve burglary hotspots within your station scope."
		} else {
			confidence = 0.9
			answer = fmt.Sprintf("Retrieved burglary hotspot counts for your station over the last 90 days. Found %d active rows.", len(hotspots))
			citationsObj = hotspots
			for _, hotspot := range hotspots {
				resultIDs = append(resultIDs, fmt.Sprintf("unit:%d/week:%s", hotspot.PoliceStationID, hotspot.Week.Format("2006-01-02")))
			}
		}
	} else if offenderRegex.MatchString(messageLower) {
		toolName = "get_repeat_offenders"
		toolArguments["minimum_cases"] = 3
		offenders, err := s.analyticsRepo.GetRepeatOffendersForUnit(3, unitID)
		if err != nil {
			answer = "I could not retrieve repeat offenders within your station scope."
		} else {
			confidence = 0.85
			if rankHierarchy > 5 {
				for i := range offenders {
					offenders[i].AccusedName = "REDACTED"
				}
			}
			answer = fmt.Sprintf("Retrieved repeat offenders within your station scope (threshold >= 3 cases). Found %d offenders.", len(offenders))
			citationsObj = offenders
			resultIDs = append(resultIDs, fmt.Sprintf("repeat-offender-count:%d", len(offenders)))
		}
	} else {
		answer = "Hello Officer! I am your Crime Analytics Assistant. You can ask me about:\n1. Burglary hotspots ('show hotspots')\n2. Repeat offenders ('show repeat offenders')\n3. Case details ('details of case <CrimeNo>')\n4. Accused connection networks ('show network for accused <AccusedID>')"
	}

	return s.persistAssistantResponse(sessionID, userID, unitID, rankHierarchy, answer, language, citationsObj, toolName, toolArguments, resultIDs, confidence, "deterministic-tool-router-v1")
}

func (s *ChatService) persistAssistantResponse(sessionID string, userID, unitID, rankHierarchy int, answer, language string, citations interface{}, toolName string, toolArguments map[string]interface{}, resultIDs []interface{}, confidence float64, modelName string) (*ChatResponse, error) {
	citationJSON, _ := json.Marshal(citations)
	botTurnID := fmt.Sprintf("turn_%d_b", time.Now().UnixNano())
	botTurn := &models.ConversationTurn{
		TurnID:       botTurnID,
		SessionID:    sessionID,
		Speaker:      "bot",
		Content:      answer,
		CitationJSON: string(citationJSON),
	}
	if err := s.chatRepo.AddTurn(botTurn); err != nil {
		return nil, err
	}

	toolJSON, _ := json.Marshal([]map[string]interface{}{{"name": toolName, "arguments": toolArguments}})
	resultJSON, _ := json.Marshal(resultIDs)
	responseHash := fmt.Sprintf("%x", sha256.Sum256([]byte(answer)))
	redactions := "[]"
	if rankHierarchy > 5 {
		redactions = `["person_names","sensitive_case_narrative"]`
	}
	evidenceTrailID := uuid.NewString()
	trail := &models.EvidenceTrail{
		EvidenceTrailID: evidenceTrailID,
		SessionID:       sessionID,
		TurnID:          botTurnID,
		UserID:          userID,
		UnitID:          unitID,
		LanguageCode:    language,
		Intent:          toolName,
		ToolCallsJSON:   string(toolJSON),
		ResultIDsJSON:   string(resultJSON),
		ModelName:       modelName,
		PromptVersion:   "police-assistant-v1",
		Confidence:      confidence,
		ResponseHash:    responseHash,
		RedactionsJSON:  redactions,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.chatRepo.AddEvidenceTrail(trail); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Answer:          answer,
		Language:        language,
		Citations:       citations,
		EvidenceTrailID: evidenceTrailID,
		Confidence:      confidence,
	}, nil
}

func (s *ChatService) Translate(ctx context.Context, input, source, target string) (*TranslationResult, error) {
	if s.orchestrator == nil {
		return nil, errors.New("translation service is not configured")
	}
	return s.orchestrator.Translate(ctx, input, source, target)
}
func (s *ChatService) Transcribe(ctx context.Context, filename string, audio []byte, language, mode string) (*TranscriptionResult, error) {
	if s.orchestrator == nil {
		return nil, errors.New("speech service is not configured")
	}
	return s.orchestrator.Transcribe(ctx, filename, audio, language, mode)
}

func (s *ChatService) GetEvidenceTrailsForUser(sessionID string, userID int) ([]models.EvidenceTrail, error) {
	return s.chatRepo.GetEvidenceTrailsForUser(sessionID, userID)
}

func (s *ChatService) GetHistory(sessionID string) ([]models.ConversationTurn, error) {
	return s.chatRepo.GetTurnsBySessionID(sessionID)
}

func (s *ChatService) GetHistoryForUser(sessionID string, userID int) ([]models.ConversationTurn, error) {
	return s.chatRepo.GetTurnsBySessionIDForUser(sessionID, userID)
}

func (s *ChatService) GetSessions(userID int) ([]models.ConversationSession, error) {
	return s.chatRepo.GetSessionsByUserID(userID)
}
