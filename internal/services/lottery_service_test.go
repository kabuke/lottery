package services

import (
	"testing"
)

func TestLotteryService_Draw(t *testing.T) {
	service := NewLotteryService()

	// Setup: Add prizes and participants
	service.AddPrize("大獎", "電視", 1, false)
	service.AddPrize("小獎", "馬克杯", 2, false)
	service.AddParticipant("001", "Alice")
	service.AddParticipant("002", "Bob")
	service.AddParticipant("003", "Charlie")

	t.Run("Test successful draw", func(t *testing.T) {
		result, err := service.Draw("大獎")

		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if result == nil {
			t.Fatal("Expected a result, but got nil")
		}

		// Check if prize quantity decreased
		if service.Prizes[0].Quantity != 0 {
			t.Errorf("Expected prize quantity to be 0, but got %d", service.Prizes[0].Quantity)
		}

		// Check if winner was recorded
		if !service.Winners[result.WinnerID] {
			t.Errorf("Expected winner %s to be recorded in the winners map", result.WinnerID)
		}

		// Check if result was recorded
		if len(service.LotteryResults) != 1 {
			t.Errorf("Expected 1 result to be recorded, but got %d", len(service.LotteryResults))
		}
	})

	t.Run("Test drawing from empty prize pool", func(t *testing.T) {
		_, err := service.Draw("大獎") // Draw again from the same prize
		if err == nil {
			t.Fatal("Expected an error for drawing from an empty prize pool, but got nil")
		}
	})

	t.Run("Test drawing with no eligible participants", func(t *testing.T) {
		// Draw all remaining participants
		_, _ = service.Draw("小獎")
		_, _ = service.Draw("小獎")

		// Add a new prize that can only be won by non-winners
		service.AddPrize("安慰獎", "糖果", 1, false)

		_, err := service.Draw("安慰獎")
		if err == nil {
			t.Fatal("Expected an error for drawing with no eligible participants, but got nil")
		}
	})

	t.Run("Test DrawFromAll allows previous winners", func(t *testing.T) {
		// Reset service state for this specific test
		service = NewLotteryService()
		service.AddPrize("特別獎", "手機", 1, true) // DrawFromAll is true
		service.AddParticipant("001", "Alice")
		service.Winners["001"] = true // Alice is already a winner

		result, err := service.Draw("特別獎")
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if result.WinnerID != "001" {
			t.Errorf("Expected winner to be 001, but got %s", result.WinnerID)
		}
	})
}
