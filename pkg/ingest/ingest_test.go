package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"odition/pkg/engine"
)

func TestIngestFile(t *testing.T) {
	eng := engine.NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	ingester := NewIngester(eng)
	messagesFile := filepath.Join("..", "..", "inbox", "messages.txt")

	res, err := ingester.IngestFile(messagesFile)
	if err != nil {
		t.Fatalf("IngestFile failed: %v", err)
	}

	if res.TotalParsed != 7 {
		t.Errorf("Expected 7 parsed messages, got %d", res.TotalParsed)
	}
	if res.TotalIgnored != 1 {
		t.Errorf("Expected 1 ignored message, got %d", res.TotalIgnored)
	}
	if res.TotalUnmatched != 0 {
		t.Errorf("Expected 0 unmatched messages, got %d", res.TotalUnmatched)
	}
	if len(res.Transactions) != 7 {
		t.Fatalf("Expected 7 transactions in result, got %d", len(res.Transactions))
	}

	expectedPrincipals := []float64{201.20, 2012.00, 3275.00, 1.00, 500.00, 0, 0}
	for i, expected := range expectedPrincipals {
		actual := res.Transactions[i].Principal
		if actual != expected {
			t.Errorf("Transaction %d: expected principal %f, got %f", i, expected, actual)
		}
	}
}

func TestIngestBatch(t *testing.T) {
	eng := engine.NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	ingester := NewIngester(eng)
	messagesFile := filepath.Join("..", "..", "inbox", "messages.txt")
	data, err := os.ReadFile(messagesFile)
	if err != nil {
		t.Fatalf("Failed to read messages: %v", err)
	}

	rawSingleBatch := string(data)
	var sb strings.Builder
	const batchMultiplier = 20
	for i := 0; i < batchMultiplier; i++ {
		sb.WriteString(rawSingleBatch)
		sb.WriteString("\n---\n")
	}

	tempDir := t.TempDir()
	largeFile := filepath.Join(tempDir, "large_messages.txt")
	if err := os.WriteFile(largeFile, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	res, err := ingester.IngestFile(largeFile)
	if err != nil {
		t.Fatalf("IngestFile failed on batch: %v", err)
	}

	expectedParsed := 7 * batchMultiplier
	if res.TotalParsed != expectedParsed {
		t.Errorf("Expected %d parsed, got %d", expectedParsed, res.TotalParsed)
	}

	expectedIgnored := 1 * batchMultiplier
	if res.TotalIgnored != expectedIgnored {
		t.Errorf("Expected %d ignored, got %d", expectedIgnored, res.TotalIgnored)
	}

	if res.TotalUnmatched != 0 {
		t.Errorf("Expected 0 unmatched, got %d", res.TotalUnmatched)
	}

	if len(res.Transactions) != expectedParsed {
		t.Errorf("Expected %d transactions, got %d", expectedParsed, len(res.Transactions))
	}
}
