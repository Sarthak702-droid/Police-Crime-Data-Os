package repositories

import (
	"errors"

	"backend/internal/models"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateSession(session *models.ConversationSession) error {
	return r.db.Create(session).Error
}

func (r *ChatRepository) GetSessionByID(sessionID string) (*models.ConversationSession, error) {
	var s models.ConversationSession
	err := r.db.Preload("Turns").Where("SessionID = ?", sessionID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *ChatRepository) GetSessionByIDForUser(sessionID string, userID int) (*models.ConversationSession, error) {
	var s models.ConversationSession
	err := r.db.Preload("Turns").Where("SessionID = ? AND UserID = ?", sessionID, userID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *ChatRepository) GetSessionsByUserID(userID int) ([]models.ConversationSession, error) {
	var sessions []models.ConversationSession
	err := r.db.Where("UserID = ?", userID).Order("created_at desc").Find(&sessions).Error
	return sessions, err
}

func (r *ChatRepository) AddTurn(turn *models.ConversationTurn) error {
	return r.db.Create(turn).Error
}

func (r *ChatRepository) GetTurnsBySessionID(sessionID string) ([]models.ConversationTurn, error) {
	var turns []models.ConversationTurn
	err := r.db.Where("SessionID = ?", sessionID).Order("created_at asc").Find(&turns).Error
	return turns, err
}

func (r *ChatRepository) GetTurnsBySessionIDForUser(sessionID string, userID int) ([]models.ConversationTurn, error) {
	var turns []models.ConversationTurn
	err := r.db.Joins("JOIN ConversationSession ON ConversationSession.SessionID = ConversationTurn.SessionID").
		Where("ConversationTurn.SessionID = ? AND ConversationSession.UserID = ?", sessionID, userID).
		Order("ConversationTurn.created_at asc").
		Find(&turns).Error
	return turns, err
}
