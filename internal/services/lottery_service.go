package services

import (
	"errors"
	"lottery/internal/models"
	"math/rand"
	"sync"
	"time"

	"github.com/google/logger"
)

// LotterySession holds the data for a single user/tenant.
type LotterySession struct {
	Prizes         []*models.Prize
	Participants   []*models.Participant
	Winners        map[string]bool // Key: Participant.ID
	LotteryResults []*models.LotteryResult
	LastActivity   time.Time
}

// LotteryService manages multiple lottery sessions.
type LotteryService struct {
	mu       sync.RWMutex
	sessions map[string]*LotterySession // Key: tenantID
}

// NewLotteryService creates and initializes a new LotteryService.
func NewLotteryService() *LotteryService {
	return &LotteryService{
		sessions: make(map[string]*LotterySession),
	}
}

// getSession returns a session for a tenant, creating one if it doesn't exist.
func (s *LotteryService) getSession(tenantID string) *LotterySession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[tenantID]
	if !exists {
		session = &LotterySession{
			Prizes:         make([]*models.Prize, 0),
			Participants:   make([]*models.Participant, 0),
			Winners:        make(map[string]bool),
			LotteryResults: make([]*models.LotteryResult, 0),
		}
		s.sessions[tenantID] = session
	}
	session.LastActivity = time.Now()
	return session
}

// GetPrizes returns the prizes for a specific tenant.
func (s *LotteryService) GetPrizes(tenantID string) []*models.Prize {
	return s.getSession(tenantID).Prizes
}

// GetParticipants returns the participants for a specific tenant.
func (s *LotteryService) GetParticipants(tenantID string) []*models.Participant {
	return s.getSession(tenantID).Participants
}

// GetLotteryResults returns the lottery results for a specific tenant.
func (s *LotteryService) GetLotteryResults(tenantID string) []*models.LotteryResult {
	return s.getSession(tenantID).LotteryResults
}

// AddPrize adds a new prize for a specific tenant.
func (s *LotteryService) AddPrize(tenantID, name, item string, quantity int, drawFromAll bool) {
	session := s.getSession(tenantID)
	session.Prizes = append(session.Prizes, &models.Prize{Name: name, Item: item, Quantity: quantity, DrawFromAll: drawFromAll})
}

// AddParticipant adds a new participant for a specific tenant.
func (s *LotteryService) AddParticipant(tenantID, id, name string) {
	session := s.getSession(tenantID)
	for _, p := range session.Participants {
		if p.ID == id {
			return
		}
	}
	session.Participants = append(session.Participants, &models.Participant{ID: id, Name: name})
}

// Draw performs the lottery draw for a specific tenant and prize.
func (s *LotteryService) Draw(tenantID, prizeName string) (*models.LotteryResult, error) {
	session := s.getSession(tenantID)

	var targetPrize *models.Prize
	for _, p := range session.Prizes {
		if p.Name == prizeName {
			targetPrize = p
			break
		}
	}

	if targetPrize == nil {
		return nil, errors.New("指定的獎項不存在")
	}

	if targetPrize.Quantity <= 0 {
		return nil, errors.New("該獎項已被抽完")
	}

	var eligibleParticipants []*models.Participant
	if targetPrize.DrawFromAll {
		eligibleParticipants = session.Participants
	} else {
		for _, p := range session.Participants {
			if !session.Winners[p.ID] {
				eligibleParticipants = append(eligibleParticipants, p)
			}
		}
	}

	if len(eligibleParticipants) == 0 {
		return nil, errors.New("沒有符合資格的參與者可供抽獎")
	}

	winnerIndex := rand.Intn(len(eligibleParticipants))
	winner := eligibleParticipants[winnerIndex]

	targetPrize.Quantity--
	session.Winners[winner.ID] = true

	result := &models.LotteryResult{
		PrizeName:  targetPrize.Name,
		PrizeItem:  targetPrize.Item,
		WinnerID:   winner.ID,
		WinnerName: winner.Name,
	}
	session.LotteryResults = append(session.LotteryResults, result)

	return result, nil
}

// CleanUpInactiveSessions removes sessions that have been inactive for over an hour.
func (s *LotteryService) CleanUpInactiveSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID, session := range s.sessions {
		if time.Since(session.LastActivity) > time.Hour {
			logger.Infof("sessions: %+v, tenantID: %+v", s.sessions, tenantID)
			delete(s.sessions, tenantID)
		}
	}
}
