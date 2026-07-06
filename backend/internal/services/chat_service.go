package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repositories"
)

type ChatService struct {
	chatRepo      *repositories.ChatRepository
	caseRepo      *repositories.CaseRepository
	analyticsRepo *repositories.AnalyticsRepository
}

type ChatResponse struct {
	Answer    string      `json:"answer"`
	Language  string      `json:"language"`
	Citations interface{} `json:"citations,omitempty"`
}

func NewChatService(chatRepo *repositories.ChatRepository, caseRepo *repositories.CaseRepository, analyticsRepo *repositories.AnalyticsRepository) *ChatService {
	return &ChatService{
		chatRepo:      chatRepo,
		caseRepo:      caseRepo,
		analyticsRepo: analyticsRepo,
	}
}

func (s *ChatService) ProcessQuery(sessionID string, userID int, unitID int, message string, language string) (*ChatResponse, error) {
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

	messageLower := strings.ToLower(message)
	var answer string
	var citationJSON []byte
	var citationsObj interface{}

	hotspotRegex := regexp.MustCompile(`(hotspot|crime rates|burglary)`)
	offenderRegex := regexp.MustCompile(`(repeat offender|criminals|recidiv)`)
	caseRegex := regexp.MustCompile(`(case|fir|crime no|crime number)\s*[:#\s]*\s*([0-9a-zA-Z]+)`)
	networkRegex := regexp.MustCompile(`(network|connection|graph|co-accused)\s*for\s*accused\s*([0-9]+)`)

	if networkRegex.MatchString(messageLower) {
		matches := networkRegex.FindStringSubmatch(messageLower)
		accusedID, _ := strconv.Atoi(matches[2])
		nodes, edges, err := s.analyticsRepo.GetCoaccusalGraphForUnit(accusedID, unitID)
		if err != nil {
			answer = fmt.Sprintf("I could not retrieve a scoped co-accusal graph for accused ID %d.", accusedID)
		} else {
			answer = fmt.Sprintf("Retrieved co-accusal network for accused ID %d within your station scope. Found %d related nodes and %d connections.", accusedID, len(nodes), len(edges))
			citationsObj = map[string]interface{}{
				"graph_nodes": nodes,
				"graph_edges": edges,
			}
			citationJSON, _ = json.Marshal(citationsObj)
		}
	} else if caseRegex.MatchString(messageLower) {
		matches := caseRegex.FindStringSubmatch(messageLower)
		crimeNo := matches[2]
		filters := repositories.SearchFilters{Keyword: crimeNo, Limit: 1, ScopeUnitID: unitID}
		cases, _, err := s.caseRepo.Search(filters)
		if err != nil || len(cases) == 0 {
			answer = fmt.Sprintf("I could not find a case matching Crime Number or ID %s within your station scope.", crimeNo)
		} else {
			targetCase := cases[0]
			answer = fmt.Sprintf("Found Case %s registered on %s. Major Head: %s. Minor Head: %s. Details: %s",
				targetCase.CrimeNo,
				targetCase.CrimeRegisteredDate.Format("02-Jan-2006"),
				targetCase.CrimeHead.CrimeGroupName,
				targetCase.CrimeSubHead.CrimeHeadName,
				targetCase.BriefFacts,
			)
			citationsObj = map[string]interface{}{
				"case_id":             targetCase.CaseMasterID,
				"crime_no":            targetCase.CrimeNo,
				"case_no":             targetCase.CaseNo,
				"registered_date":     targetCase.CrimeRegisteredDate,
				"brief_facts":         targetCase.BriefFacts,
				"police_station_name": targetCase.PoliceStation.UnitName,
			}
			citationJSON, _ = json.Marshal(citationsObj)
		}
	} else if hotspotRegex.MatchString(messageLower) {
		hotspots, err := s.analyticsRepo.GetBurglaryHotspotsForUnit(unitID)
		if err != nil {
			answer = "I could not retrieve burglary hotspots within your station scope."
		} else {
			answer = fmt.Sprintf("Retrieved burglary hotspot counts for your station over the last 90 days. Found %d active rows.", len(hotspots))
			citationsObj = hotspots
			citationJSON, _ = json.Marshal(citationsObj)
		}
	} else if offenderRegex.MatchString(messageLower) {
		offenders, err := s.analyticsRepo.GetRepeatOffendersForUnit(3, unitID)
		if err != nil {
			answer = "I could not retrieve repeat offenders within your station scope."
		} else {
			answer = fmt.Sprintf("Retrieved repeat offenders within your station scope (threshold >= 3 cases). Found %d offenders.", len(offenders))
			citationsObj = offenders
			citationJSON, _ = json.Marshal(citationsObj)
		}
	} else {
		answer = "Hello Officer! I am your Crime Analytics Assistant. You can ask me about:\n1. Burglary hotspots ('show hotspots')\n2. Repeat offenders ('show repeat offenders')\n3. Case details ('details of case <CrimeNo>')\n4. Accused connection networks ('show network for accused <AccusedID>')"
	}

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

	return &ChatResponse{
		Answer:    answer,
		Language:  language,
		Citations: citationsObj,
	}, nil
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
