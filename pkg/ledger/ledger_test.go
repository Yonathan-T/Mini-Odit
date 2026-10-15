package ledger

import (
	"path/filepath"
	"testing"

	"odition/pkg/engine"
	"odition/pkg/ingest"
)

func TestFinancialAggregation(t *testing.T) {
	eng := engine.NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	ingester := ingest.NewIngester(eng)
	messagesFile := filepath.Join("..", "..", "inbox", "messages.txt")

	res, err := ingester.IngestFile(messagesFile)
	if err != nil {
		t.Fatalf("IngestFile failed: %v", err)
	}

	agg := NewAggregator("Yonathan")
	summary := agg.Aggregate(res.Transactions)

	expectedIncome := 3276.00
	if summary.TotalIncome != expectedIncome {
		t.Errorf("Expected TotalIncome %f, got %f", expectedIncome, summary.TotalIncome)
	}

	expectedExpense := 2715.20
	if summary.TotalExpense != expectedExpense {
		t.Errorf("Expected TotalExpense %f, got %f", expectedExpense, summary.TotalExpense)
	}

	expectedFees := 2.00
	if summary.TotalFees != expectedFees {
		t.Errorf("Expected TotalFees %f, got %f", expectedFees, summary.TotalFees)
	}

	expectedNet := 560.80
	diff := summary.NetCashFlow - expectedNet
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("Expected NetCashFlow %f, got %f", expectedNet, summary.NetCashFlow)
	}

	tbAccount, ok := summary.Accounts["telebirr"]
	if !ok || tbAccount.LatestBalance == nil || *tbAccount.LatestBalance != 108.00 {
		t.Errorf("Expected Telebirr latest balance 108.00, got %v", tbAccount)
	}

	boaAccount, ok := summary.Accounts["boa"]
	if !ok || boaAccount.LatestBalance == nil || *boaAccount.LatestBalance != 7432.95 {
		t.Errorf("Expected BOA latest balance 7432.95, got %v", boaAccount)
	}

	if _, isSelfPresent := summary.People["Yonathan"]; isSelfPresent {
		t.Errorf("Expected Yonathan to be excluded from people directory, but was found")
	}

	samson, ok := summary.People["Samson Debebe"]
	if !ok || samson.Received != 3275.00 {
		t.Errorf("Expected Samson Debebe with 3275.00 received, got %+v", samson)
	}

	nebyu, ok := summary.People["Nebyu Elias"]
	if !ok || nebyu.Sent != 500.00 || nebyu.Phone != "0911****00" {
		t.Errorf("Expected Nebyu with 500.00 sent and phone 0911****00, got %+v", nebyu)
	}

	loza, ok := summary.People["Loza Tesfahun"]
	if !ok || loza.Received != 1.00 || loza.Phone != "251911******" {
		t.Errorf("Expected Loza with 1.00 received and phone 251911******, got %+v", loza)
	}
}
