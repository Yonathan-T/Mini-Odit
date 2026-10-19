package engine

import (
	"path/filepath"
	"testing"

	"odition/config"
	"odition/pkg/model"
	"odition/pkg/router"
)

func TestParseRealBlogSMS(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")

	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	realSMS := `Dear Robel
You have transferred ETB 20.00 to dagim alemu (2519****5183) on 12/09/2026 00:08:40. Your transaction number is DIC6NJHLOC. The service fee is  ETB 0.87 and  15% VAT on the service fee is ETB 0.13. Your current E-Money Account  balance is ETB 6,127.70. To download your payment information please click this link: https://transactioninfo.ethiotelecom.et/receipt/DIC6NJHLOC.

Thank you for using telebirr
Ethio telecom`

	txn, err := eng.Parse(realSMS)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if txn.PatternName != "TRANSFER_TB_FROM_TB" {
		t.Errorf("Expected pattern TRANSFER_TB_FROM_TB, got %s", txn.PatternName)
	}
	if txn.Provider != "telebirr" {
		t.Errorf("Expected provider telebirr, got %s", txn.Provider)
	}
	if txn.Direction != model.DirectionOutgoing {
		t.Errorf("Expected direction OUTGOING, got %s", txn.Direction)
	}
	if txn.Principal != 20.00 {
		t.Errorf("Expected principal 20.00, got %f", txn.Principal)
	}
	if txn.Fees != 1.00 {
		t.Errorf("Expected fees 1.00, got %f", txn.Fees)
	}
	if txn.TotalAmount != 21.00 {
		t.Errorf("Expected total amount 21.00, got %f", txn.TotalAmount)
	}
	if txn.Balance == nil || *txn.Balance != 6127.70 {
		t.Errorf("Expected balance 6127.70, got %v", txn.Balance)
	}
	if txn.TransactionIDs["telebirr"] != "DIC6NJHLOC" {
		t.Errorf("Expected txn ID DIC6NJHLOC, got %s", txn.TransactionIDs["telebirr"])
	}
}

func TestParseBOADebit(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	sms := `Dear Yonatan, your account 1*00 was debited with ETB 201.20. Available Balance: ETB 3,935.20.
Receipt: https://cs.bankofabyssinia.com/slip/?trx=FT260000000000001
Feedback: https://cs.bankofabyssinia.com/cs/?trx=DFT2600000000
Link your Fayda: https://cs.bankofabyssinia.com/fayda_connect 
For help, call 8397 (24/7 Toll-Free). Bank of Abyssinia.`

	txn, err := eng.Parse(sms)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if txn.PatternName != "BOA_DEBIT" {
		t.Errorf("Expected BOA_DEBIT, got %s", txn.PatternName)
	}
	if txn.Direction != model.DirectionOutgoing {
		t.Errorf("Expected OUTGOING, got %s", txn.Direction)
	}
	if txn.Principal != 201.20 {
		t.Errorf("Expected 201.20, got %f", txn.Principal)
	}
	if txn.Balance == nil || *txn.Balance != 3935.20 {
		t.Errorf("Expected balance 3935.20, got %v", txn.Balance)
	}
	if txn.TransactionIDs["boa"] != "FT260000000000001" {
		t.Errorf("Expected FT260000000000001, got %s", txn.TransactionIDs["boa"])
	}
	if len(txn.Participants) == 0 || txn.Participants[0].Name != "Yonatan" {
		t.Errorf("Expected participant Yonatan, got %+v", txn.Participants)
	}
}

func TestParseBOACreditP2P(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	sms := `Dear Yonatan, your account 1*00 was credited with ETB 3,275.00 by  Gizealew Lulie Gared . Available Balance: ETB 7,432.95.
Receipt: https://cs.bankofabyssinia.com/slip/?trx=FT260000000000003
Feedback: https://cs.bankofabyssinia.com/cs/?trx=CFT2600000002
Link your Fayda: https://cs.bankofabyssinia.com/fayda_connect 
For help, call 8397 (24/7 Toll-Free). Bank of Abyssinia.`

	txn, err := eng.Parse(sms)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if txn.PatternName != "BOA_CREDIT_P2P" {
		t.Errorf("Expected BOA_CREDIT_P2P, got %s", txn.PatternName)
	}
	if txn.Direction != model.DirectionIncoming {
		t.Errorf("Expected INCOMING, got %s", txn.Direction)
	}
	if txn.Principal != 3275.00 {
		t.Errorf("Expected 3275.00, got %f", txn.Principal)
	}
	if txn.Balance == nil || *txn.Balance != 7432.95 {
		t.Errorf("Expected balance 7432.95, got %v", txn.Balance)
	}
	if txn.TransactionIDs["boa"] != "FT260000000000003" {
		t.Errorf("Expected FT260000000000003, got %s", txn.TransactionIDs["boa"])
	}

	if len(txn.Participants) != 2 {
		t.Fatalf("Expected 2 participants, got %d", len(txn.Participants))
	}
	sender := txn.Participants[0]
	if sender.Name != "Gizealew Lulie Gared" || sender.Role != "SENDER" {
		t.Errorf("Expected sender Gizealew Lulie Gared, got %+v", sender)
	}
	receiver := txn.Participants[1]
	if receiver.Name != "Yonatan" || receiver.Role != "RECEIVER" {
		t.Errorf("Expected receiver Yonatan, got %+v", receiver)
	}
}

func TestParseTelebirrAmharic(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	sms := `ውድ YONATHAN
ወደ Nebyu Elias(0911****00) 500.00 ብር በ 30/08/2026 18:33:14  ልከዋል። የሂሳብ እንቅስቃሴ ቁጥርዎ DHU000000000 ነዉ። የአገልግሎት ክፍያው 1.74 ብር ፤ የአገልግሎት ክፍያው 15% VAT 0.26 ብር ነዉ።  አሁን ያለዎት ቀሪ ሂሳብ 108.00 ብር ነው። የክፍያ መረጃዎን ለማግኘት ማስፈንጠርያውን ይጫኑ፡ https://transactioninfo.ethiotelecom.et/receipt/DHU000000000
በቴሌብር ስለተገለገሉ እናመሰግናለን 
ኢትዮ ቴሌኮም`

	txn, err := eng.Parse(sms)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if txn.PatternName != "TELEBIRR_TRANSFER_AMHARIC" {
		t.Errorf("Expected TELEBIRR_TRANSFER_AMHARIC, got %s", txn.PatternName)
	}
	if txn.Direction != model.DirectionOutgoing {
		t.Errorf("Expected OUTGOING, got %s", txn.Direction)
	}
	if txn.Principal != 500.00 {
		t.Errorf("Expected 500.00, got %f", txn.Principal)
	}
	if txn.Fees != 2.00 {
		t.Errorf("Expected fees 2.00, got %f", txn.Fees)
	}
	if txn.TotalAmount != 502.00 {
		t.Errorf("Expected total 502.00, got %f", txn.TotalAmount)
	}
	if txn.Balance == nil || *txn.Balance != 108.00 {
		t.Errorf("Expected balance 108.00, got %v", txn.Balance)
	}
	if txn.TransactionIDs["telebirr"] != "DHU000000000" {
		t.Errorf("Expected DHU000000000, got %s", txn.TransactionIDs["telebirr"])
	}

	if len(txn.Participants) != 2 {
		t.Fatalf("Expected 2 participants, got %d", len(txn.Participants))
	}
	sender := txn.Participants[0]
	if sender.Name != "YONATHAN" || sender.Role != "SENDER" {
		t.Errorf("Expected sender YONATHAN, got %+v", sender)
	}
	receiver := txn.Participants[1]
	if receiver.Name != "Nebyu Elias" || receiver.Phone != "0911****00" || receiver.Role != "RECEIVER" {
		t.Errorf("Expected receiver Nebyu Elias, got %+v", receiver)
	}
}

func TestParseTelebirrReceiveP2P(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	sms := `Dear YONATHAN 
You have received ETB 1.00 from Fereiwot Demelash( 251911******)  on 09/11/2024 16:29:33. Your transaction number is
BK0000000000. Your current E-money Account balance is ETB 1.00.
Thank you for using telebirr
Ethio telecom`

	txn, err := eng.Parse(sms)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if txn.PatternName != "TELEBIRR_RECEIVE_P2P" {
		t.Errorf("Expected TELEBIRR_RECEIVE_P2P, got %s", txn.PatternName)
	}
	if txn.Direction != model.DirectionIncoming {
		t.Errorf("Expected INCOMING, got %s", txn.Direction)
	}
	if txn.Principal != 1.00 {
		t.Errorf("Expected 1.00, got %f", txn.Principal)
	}
	if txn.Balance == nil || *txn.Balance != 1.00 {
		t.Errorf("Expected balance 1.00, got %v", txn.Balance)
	}
	if txn.TransactionIDs["telebirr"] != "BK0000000000" {
		t.Errorf("Expected BK0000000000, got %s", txn.TransactionIDs["telebirr"])
	}
	expectedURL := "https://transactioninfo.ethiotelecom.et/receipt/BK0000000000"
	if txn.ReceiptURL != expectedURL {
		t.Errorf("Expected receipt URL %s, got %s", expectedURL, txn.ReceiptURL)
	}

	if len(txn.Participants) != 2 {
		t.Fatalf("Expected 2 participants, got %d", len(txn.Participants))
	}
	sender := txn.Participants[0]
	if sender.Name != "Fereiwot Demelash" || sender.Phone != "251911******" || sender.Role != "SENDER" {
		t.Errorf("Expected sender Fereiwot Demelash, got %+v", sender)
	}
	receiver := txn.Participants[1]
	if receiver.Name != "YONATHAN" || receiver.Role != "RECEIVER" {
		t.Errorf("Expected receiver YONATHAN, got %+v", receiver)
	}
}

func TestFallbackURLDerivation(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	oldSMS := `Dear Robel
You have transferred ETB 20.00 to dagim alemu (2519****5183) on 12/09/2026 00:08:40. Your transaction number is DIC6NJHLOC. The service fee is  ETB 0.87 and  15% VAT on the service fee is ETB 0.13. Your current E-Money Account  balance is ETB 6,127.70.

Thank you for using telebirr
Ethio telecom`

	txn, err := eng.Parse(oldSMS)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expectedURL := "https://transactioninfo.ethiotelecom.et/receipt/DIC6NJHLOC"
	if txn.ReceiptURL != expectedURL {
		t.Errorf("Expected fallback URL %s, got %s", expectedURL, txn.ReceiptURL)
	}
}

func TestNegativeLookaheadTipExclusion(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	tipSMS := `Dear Robel
You have transferred ETB 20.00 Tip to dagim alemu (2519****5183) on 12/09/2026 00:08:40. Your transaction number is DIC6NJHLOC. The service fee is  ETB 0.87 and  15% VAT on the service fee is ETB 0.13. Your current E-Money Account  balance is ETB 6,127.70.`

	_, err := eng.Parse(tipSMS)
	if err == nil {
		t.Fatalf("Expected tip message to be rejected by TRANSFER_TB_FROM_TB identifier, but it matched!")
	}
}

func TestCrossBankRouterFallback(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	topoPath := filepath.Join("..", "..", "Topology", "topology.json")
	top, err := config.Load(topoPath)
	if err != nil {
		t.Fatalf("Failed to load topology: %v", err)
	}
	eng.Router = router.NewRouter(top)

	crossBankSMS := `Dear Yonathan, your account 1*00 was debited with ETB 201.20 via telebirr transfer. Available Balance: ETB 3,935.20.
Receipt: https://cs.bankofabyssinia.com/slip/?trx=FT260000000000001
Feedback: https://cs.bankofabyssinia.com/cs/?trx=DFT2600000000
Link your Fayda: https://cs.bankofabyssinia.com/fayda_connect 
For help, call 8397 (24/7 Toll-Free). Bank of Abyssinia.`

	txn, err := eng.Parse(crossBankSMS)
	if err != nil {
		t.Fatalf("Cross-bank SMS failed to parse with fallback re-routing: %v", err)
	}

	if txn.Provider != "boa" {
		t.Errorf("Expected provider boa after fallback, got %s", txn.Provider)
	}
	if txn.Principal != 201.20 {
		t.Errorf("Expected principal 201.20, got %f", txn.Principal)
	}
}

func TestRouterTrace(t *testing.T) {
	eng := NewEngine()
	patternsDir := filepath.Join("..", "..", "patterns")
	if err := eng.LoadFromDir(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	topoPath := filepath.Join("..", "..", "Topology", "topology.json")
	top, err := config.Load(topoPath)
	if err != nil {
		t.Fatalf("Failed to load topology: %v", err)
	}
	eng.Router = router.NewRouter(top)

	cases := []struct {
		name             string
		sms              string
		expectedRouted   string
		expectedFallback bool
		expectedFinal    string
	}{
		{
			name: "Direct Route - Telebirr P2P",
			sms: `Dear Robel
You have transferred ETB 20.00 to dagim alemu (2519****5183) on 12/09/2026 00:08:40. Your transaction number is DIC6NJHLOC. The service fee is  ETB 0.87 and  15% VAT on the service fee is ETB 0.13. Your current E-Money Account  balance is ETB 6,127.70. To download your payment information please click this link: https://transactioninfo.ethiotelecom.et/receipt/DIC6NJHLOC.

Thank you for using telebirr
Ethio telecom`,
			expectedRouted:   "telebirr",
			expectedFallback: false,
			expectedFinal:    "telebirr",
		},
		{
			name: "Direct Route - BOA Debit",
			sms: `Dear Yonathan, your account 1*00 was debited with ETB 50.00. Available Balance: ETB 3,935.20.
Receipt: https://cs.bankofabyssinia.com/slip/?trx=FT260000000000001
Feedback: https://cs.bankofabyssinia.com/cs/?trx=DFT2600000000
Link your Fayda: https://cs.bankofabyssinia.com/fayda_connect 
For help, call 8397 (24/7 Toll-Free). Bank of Abyssinia.`,
			expectedRouted:   "boa",
			expectedFallback: false,
			expectedFinal:    "boa",
		},
		{
			name: "Fallback Sequential Scan - BOA referencing Telebirr",
			sms: `Dear Yonathan, your account 1*00 was debited with ETB 201.20 via telebirr transfer. Available Balance: ETB 3,935.20.
Receipt: https://cs.bankofabyssinia.com/slip/?trx=FT260000000000001
Feedback: https://cs.bankofabyssinia.com/cs/?trx=DFT2600000000
Link your Fayda: https://cs.bankofabyssinia.com/fayda_connect 
For help, call 8397 (24/7 Toll-Free). Bank of Abyssinia.`,
			expectedRouted:   "telebirr",
			expectedFallback: true,
			expectedFinal:    "boa",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trace := eng.ParseWithTrace(tc.sms)
			if trace.Err != nil {
				t.Fatalf("Parse error: %v", trace.Err)
			}
			t.Logf("[%s]", tc.name)
			t.Logf("  Initial Route Anchor Bucket: %q", trace.RoutedProvider)
			t.Logf("  Fallback Sequential Scan Used: %v", trace.FallbackUsed)
			t.Logf("  Final Matched Provider: %q", trace.FinalProvider)

			if trace.RoutedProvider != tc.expectedRouted {
				t.Errorf("Expected routed provider %q, got %q", tc.expectedRouted, trace.RoutedProvider)
			}
			if trace.FallbackUsed != tc.expectedFallback {
				t.Errorf("Expected fallback %v, got %v", tc.expectedFallback, trace.FallbackUsed)
			}
			if trace.FinalProvider != tc.expectedFinal {
				t.Errorf("Expected final provider %q, got %q", tc.expectedFinal, trace.FinalProvider)
			}
		})
	}
}
