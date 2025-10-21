package services

import (
	"testing"
)

func TestLotteryService_Draw_Successful(t *testing.T) {
	const testTenantID = "test-tenant-success"
	service := NewLotteryService()

	// Setup
	service.AddPrize(testTenantID, "大獎", "電視", 1, false)
	service.AddParticipant(testTenantID, "001", "Alice")
	service.AddParticipant(testTenantID, "002", "Bob")

	result, err := service.Draw(testTenantID, "大獎")

	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}
	if result == nil {
		t.Fatal("Expected a result, but got nil")
	}

	// Check prize quantity
	prizes := service.GetPrizes(testTenantID)
	var foundPrize bool
	for _, p := range prizes {
		if p.Name == "大獎" {
			foundPrize = true
			if p.Quantity != 0 {
				t.Errorf("Expected prize quantity to be 0, but got %d", p.Quantity)
			}
			break
		}
	}
	if !foundPrize {
		t.Fatal("Could not find the prize '大獎' after drawing")
	}

	// Check winner recording
	session := service.getSession(testTenantID)
	if _, ok := session.Winners[result.WinnerID]; !ok {
		t.Errorf("Expected winner %s to be recorded", result.WinnerID)
	}
	if len(session.LotteryResults) != 1 {
		t.Errorf("Expected 1 result to be recorded, but got %d", len(session.LotteryResults))
	}
}

func TestLotteryService_Draw_EmptyPrizePool(t *testing.T) {
	const testTenantID = "test-tenant-empty-prize"
	service := NewLotteryService()

	// Setup
	service.AddPrize(testTenantID, "大獎", "電視", 1, false)
	service.AddParticipant(testTenantID, "001", "Alice")
	_, err := service.Draw(testTenantID, "大獎") // First draw exhausts the prize
	if err != nil {
		t.Fatalf("Setup draw failed: %v", err)
	}

	// Test drawing again
	_, err = service.Draw(testTenantID, "大獎")
	if err == nil {
		t.Fatal("Expected an error for drawing from an empty prize pool, but got nil")
	}
}

func TestLotteryService_Draw_NoEligibleParticipants(t *testing.T) {
	const testTenantID = "test-tenant-no-eligible"
	service := NewLotteryService()

	// Setup: one participant, one prize. Draw it so everyone is a winner.
	service.AddPrize(testTenantID, "小獎", "馬克杯", 1, false)
	service.AddParticipant(testTenantID, "001", "Alice")
	_, err := service.Draw(testTenantID, "小獎")
	if err != nil {
		t.Fatalf("Setup draw failed: %v", err)
	}

	// Add a new prize that can only be won by non-winners
	service.AddPrize(testTenantID, "安慰獎", "糖果", 1, false)

	// Test drawing with no non-winners left
	_, err = service.Draw(testTenantID, "安慰獎")
	if err == nil {
		t.Fatal("Expected an error for drawing with no eligible participants, but got nil")
	}
}

func TestLotteryService_Draw_FromAllAllowsPreviousWinners(t *testing.T) {
	const testTenantID = "test-tenant-draw-all"
	service := NewLotteryService()

	// Setup
	service.AddPrize(testTenantID, "特別獎", "手機", 1, true) // DrawFromAll is true
	service.AddParticipant(testTenantID, "001", "Alice")

	// Manually mark the only participant as a winner to simulate the condition.
	session := service.getSession(testTenantID)
	session.Winners["001"] = map[string]bool{"previous-prize": true}

	// Test
	result, err := service.Draw(testTenantID, "特別獎")
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}
	if result == nil {
		t.Fatal("Expected a result, but got nil")
	}
	if result.WinnerID != "001" {
		t.Errorf("Expected winner to be 001, but got %s", result.WinnerID)
	}
}