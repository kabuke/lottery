package services

import (
	"testing"
)

func TestLotteryService_Draw(t *testing.T) {
	const testTenantID = "test-tenant"
	service := NewLotteryService()

	// Setup: Add prizes and participants for the test tenant
	service.AddPrize(testTenantID, "大獎", "電視", 1, false)
	service.AddPrize(testTenantID, "小獎", "馬克杯", 2, false)
	service.AddParticipant(testTenantID, "001", "Alice")
	service.AddParticipant(testTenantID, "002", "Bob")
	service.AddParticipant(testTenantID, "003", "Charlie")

	t.Run("Test successful draw", func(t *testing.T) {
		result, err := service.Draw(testTenantID, "大獎")

		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if result == nil {
			t.Fatal("Expected a result, but got nil")
		}

		// Check if prize quantity decreased
		prizes := service.GetPrizes(testTenantID)
		if prizes[0].Quantity != 0 {
			t.Errorf("Expected prize quantity to be 0, but got %d", prizes[0].Quantity)
		}

		// Check if winner was recorded
		session := service.getSession(testTenantID)
		if !session.Winners[result.WinnerID] {
			t.Errorf("Expected winner %s to be recorded in the winners map", result.WinnerID)
		}

		// Check if result was recorded
		if len(session.LotteryResults) != 1 {
			t.Errorf("Expected 1 result to be recorded, but got %d", len(session.LotteryResults))
		}
	})

	t.Run("Test drawing from empty prize pool", func(t *testing.T) {
		_, err := service.Draw(testTenantID, "大獎") // Draw again from the same prize
		if err == nil {
			t.Fatal("Expected an error for drawing from an empty prize pool, but got nil")
		}
	})

	t.Run("Test drawing with no eligible participants", func(t *testing.T) {
		// Draw all remaining participants
		_, _ = service.Draw(testTenantID, "小獎")
		_, _ = service.Draw(testTenantID, "小獎")

		// Add a new prize that can only be won by non-winners
		service.AddPrize(testTenantID, "安慰獎", "糖果", 1, false)

		_, err := service.Draw(testTenantID, "安慰獎")
		if err == nil {
			t.Fatal("Expected an error for drawing with no eligible participants, but got nil")
		}
	})

	t.Run("Test DrawFromAll allows previous winners", func(t *testing.T) {
		// Use a new tenant for this specific test to ensure isolation
		const specialTenantID = "special-tenant"
		service.AddPrize(specialTenantID, "特別獎", "手機", 1, true) // DrawFromAll is true
		service.AddParticipant(specialTenantID, "001", "Alice")
		session := service.getSession(specialTenantID)
		session.Winners["001"] = true // Alice is already a winner

		result, err := service.Draw(specialTenantID, "特別獎")
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if result.WinnerID != "001" {
			t.Errorf("Expected winner to be 001, but got %s", result.WinnerID)
		}
	})
}