package models

// Prize represents a single prize category in the lottery.
// It includes the name of the prize, the specific item, the total quantity,
// and a flag to determine the pool of participants for this prize.
type Prize struct {
	Name        string `json:"name"`
	Item        string `json:"item"`
	Quantity    int    `json:"quantity"`
	DrawFromAll bool   `json:"drawFromAll"` // true: draw from all participants; false: draw from non-winners only
}

// Participant represents a person entering the lottery.

type Participant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LotteryResult stores the outcome of a single draw,
// linking a winner to a specific prize.
type LotteryResult struct {
	PrizeName  string `json:"prizeName"`
	PrizeItem  string `json:"prizeItem"`
	WinnerID   string `json:"winnerId"`
	WinnerName string `json:"winnerName"`
}
