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

func TestLotteryService_Draw_SpecialPrizeDoesNotBlockRegularPrize(t *testing.T) {
	const testTenantID = "test-tenant-special-then-regular"
	service := NewLotteryService()

	// Setup:
	// Prize S (Special, DrawFromAll=true)
	// Prize A (Regular, DrawFromAll=false)
	service.AddPrize(testTenantID, "PrizeS", "Special Item", 1, true)
	service.AddPrize(testTenantID, "PrizeA", "Regular Item", 1, false)

	// Participant: User1
	service.AddParticipant(testTenantID, "001", "User1")

	// Step 1: Draw Prize S (Special)
	resultS, err := service.Draw(testTenantID, "PrizeS")
	if err != nil {
		t.Fatalf("Failed to draw Special Prize: %v", err)
	}
	if resultS.WinnerID != "001" {
		t.Fatalf("Expected User1 to win Special Prize, but got %s", resultS.WinnerID)
	}

	// Step 2: Draw Prize A (Regular)
	// User1 should STILL be eligible because Prize S is special.
	resultA, err := service.Draw(testTenantID, "PrizeA")
	if err != nil {
		t.Fatalf("Failed to draw Regular Prize: %v", err)
	}
	if resultA.WinnerID != "001" {
		t.Fatalf("Expected User1 to win Regular Prize, but got %s", resultA.WinnerID)
	}

	// Verify User1 has both prizes
	session := service.getSession(testTenantID)
	wins := session.Winners["001"]
	if !wins["PrizeS"] || !wins["PrizeA"] {
		t.Errorf("Expected User1 to have both prizes, but wins map is: %v", wins)
	}
}

func TestLotteryService_Draw_MultipleSpecialPrizes(t *testing.T) {
	const testTenantID = "test-tenant-multi-special"
	service := NewLotteryService()

	// Setup:
	// Prize A (Regular)
	// Prize S (Special)
	// Prize SS (Special)
	service.AddPrize(testTenantID, "PrizeA", "Regular", 1, false)
	service.AddPrize(testTenantID, "PrizeS", "Special1", 1, true)
	service.AddPrize(testTenantID, "PrizeSS", "Special2", 1, true)

	// Participant: User1
	service.AddParticipant(testTenantID, "001", "User1")

	// 1. Draw Prize S
	service.Draw(testTenantID, "PrizeS")

	// 2. Draw Prize SS (Should be eligible)
	_, err := service.Draw(testTenantID, "PrizeSS")
	if err != nil {
		t.Fatalf("Failed to draw second Special Prize (SS): %v", err)
	}

	// 3. Draw Prize A (Should be eligible, as S and SS are special)
	_, err = service.Draw(testTenantID, "PrizeA")
	if err != nil {
		t.Fatalf("Failed to draw Regular Prize (A): %v", err)
	}

	// Verify User1 has all 3 prizes
	session := service.getSession(testTenantID)
	wins := session.Winners["001"]
	if len(wins) != 3 {
		t.Errorf("Expected User1 to have 3 prizes, but got %d", len(wins))
	}
}
