package services

import (
	"errors"
	"lottery/internal/models"
	"math/rand"
	"time"
)

// LotteryService holds the state of the lottery application.
type LotteryService struct {
	Prizes         []*models.Prize
	Participants   []*models.Participant
	Winners        map[string]bool // Key: Participant.ID, Value: true if they have won
	LotteryResults []*models.LotteryResult
}

var RandSeed *rand.Rand

// NewLotteryService creates and initializes a new LotteryService.
func NewLotteryService() *LotteryService {
	RandSeed = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &LotteryService{
		Prizes:         make([]*models.Prize, 0),
		Participants:   make([]*models.Participant, 0),
		Winners:        make(map[string]bool),
		LotteryResults: make([]*models.LotteryResult, 0),
	}
}

// AddPrize adds a new prize to the lottery.
func (s *LotteryService) AddPrize(name, item string, quantity int, drawFromAll bool) {
	s.Prizes = append(s.Prizes, &models.Prize{Name: name, Item: item, Quantity: quantity, DrawFromAll: drawFromAll})
}

func (s *LotteryService) ClearPrize() {
	s.Prizes = []*models.Prize{}
}

// AddParticipant adds a new participant to the lottery.
func (s *LotteryService) AddParticipant(id, name string) {
	// Avoid adding duplicate participants
	for _, p := range s.Participants {
		if p.ID == id {
			return
		}
	}
	s.Participants = append(s.Participants, &models.Participant{ID: id, Name: name})
}

func (s *LotteryService) ClearParticipant() {
	s.Participants = []*models.Participant{}
}

// Draw performs the lottery draw for a specific prize.
func (s *LotteryService) Draw(prizeName string) (*models.LotteryResult, error) {
	var targetPrize *models.Prize
	for _, p := range s.Prizes {
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
		eligibleParticipants = s.Participants
	} else {
		for _, p := range s.Participants {
			if !s.Winners[p.ID] {
				eligibleParticipants = append(eligibleParticipants, p)
			}
		}
	}

	if len(eligibleParticipants) == 0 {
		return nil, errors.New("沒有符合資格的參與者可供抽獎")
	}

	winnerIndex := RandSeed.Intn(len(eligibleParticipants))
	winner := eligibleParticipants[winnerIndex]

	// Update state
	targetPrize.Quantity--
	s.Winners[winner.ID] = true

	result := &models.LotteryResult{
		PrizeName:  targetPrize.Name,
		PrizeItem:  targetPrize.Item,
		WinnerID:   winner.ID,
		WinnerName: winner.Name,
	}
	s.LotteryResults = append(s.LotteryResults, result)

	return result, nil
}
