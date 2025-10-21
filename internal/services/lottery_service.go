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
	// Winners maps a participant ID to a set of prize names they have won.
	Winners        map[string]map[string]bool // map[participantID]map[prizeName]true
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
			Winners:        make(map[string]map[string]bool),
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

	eligibleParticipants, err := s.GetEligibleParticipants(tenantID, prizeName)
	if err != nil {
		return nil, err
	}

	// Find the target prize (we know it exists from GetEligibleParticipants)
	var targetPrize *models.Prize
	for _, p := range session.Prizes {
		if p.Name == prizeName {
			targetPrize = p
			break
		}
	}

	winnerIndex := rand.Intn(len(eligibleParticipants))
	winner := eligibleParticipants[winnerIndex]

	// Update state
	targetPrize.Quantity--
	// Ensure the nested map exists before writing to it
	if session.Winners[winner.ID] == nil {
		session.Winners[winner.ID] = make(map[string]bool)
	}
	session.Winners[winner.ID][prizeName] = true

	result := &models.LotteryResult{
		PrizeName:  targetPrize.Name,
		PrizeItem:  targetPrize.Item,
		WinnerID:   winner.ID,
		WinnerName: winner.Name,
	}
	session.LotteryResults = append(session.LotteryResults, result)

	return result, nil
}

// GetEligibleParticipants returns a slice of participants eligible for a specific prize draw.
func (s *LotteryService) GetEligibleParticipants(tenantID, prizeName string) ([]*models.Participant, error) {
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
	for _, p := range session.Participants {
		wins := session.Winners[p.ID]

		if targetPrize.DrawFromAll {
			// Rule: Can win this prize category only once.
			if !wins[prizeName] { // If they have NOT won this specific prize before
				eligibleParticipants = append(eligibleParticipants, p)
			}
		} else {
			// Rule: Can only win one prize in total from the non-drawFromAll pool.
			if len(wins) == 0 { // If their win record is empty
				eligibleParticipants = append(eligibleParticipants, p)
			}
		}
	}

	if len(eligibleParticipants) == 0 {
		return nil, errors.New("沒有符合資格的參與者可供抽獎")
	}

	return eligibleParticipants, nil
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

// ClearSession removes all data associated with a specific tenant.
func (s *LotteryService) ClearSession(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, tenantID)
	logger.Infof("Cleared session for tenant: %s", tenantID)
}