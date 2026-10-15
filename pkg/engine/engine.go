package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"gopkg.in/yaml.v3"

	"odition/pkg/model"
)

var (
	ErrIgnored          = errors.New("message is in ignored set")
	ErrNoMatch          = errors.New("no pattern matched message")
	ErrMissingRequired  = errors.New("missing required field in extraction")
)

var placeholderRegex = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

type CompiledRegex struct {
	Re       *regexp2.Regexp
	Fallback string
	Optional bool
}

type CompiledPattern struct {
	Def        model.PatternDefinition
	Identifier *regexp2.Regexp
	FieldRes   map[string]CompiledRegex
}

type Engine struct {
	IgnoredSet   map[string]struct{}
	ExactPromos  map[string]string
	Patterns     []CompiledPattern
	ProviderMap  map[string][]CompiledPattern
}

func NewEngine() *Engine {
	return &Engine{
		IgnoredSet:  make(map[string]struct{}),
		ExactPromos: make(map[string]string),
		ProviderMap: make(map[string][]CompiledPattern),
	}
}

func (e *Engine) LoadFromDir(dir string) error {
	ignoredPath := filepath.Join(dir, "_ignored.yml")
	if _, err := os.Stat(ignoredPath); err == nil {
		if err := e.loadIgnored(ignoredPath); err != nil {
			return fmt.Errorf("loading _ignored.yml: %w", err)
		}
	}

	exactPath := filepath.Join(dir, "_exact.yml")
	if _, err := os.Stat(exactPath); err == nil {
		if err := e.loadExact(exactPath); err != nil {
			return fmt.Errorf("loading _exact.yml: %w", err)
		}
	}

	var loadedPatterns []CompiledPattern

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "_") {
			return nil
		}
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading pattern %s: %w", path, err)
		}

		var def model.PatternDefinition
		if err := yaml.Unmarshal(content, &def); err != nil {
			return fmt.Errorf("parsing pattern %s: %w", path, err)
		}

		compiled, err := compilePattern(def)
		if err != nil {
			return fmt.Errorf("compiling pattern %s: %w", path, err)
		}

		loadedPatterns = append(loadedPatterns, compiled)
		return nil
	})

	if err != nil {
		return err
	}

	sort.Slice(loadedPatterns, func(i, j int) bool {
		if loadedPatterns[i].Def.Order != loadedPatterns[j].Def.Order {
			return loadedPatterns[i].Def.Order < loadedPatterns[j].Def.Order
		}
		return loadedPatterns[i].Def.Name < loadedPatterns[j].Def.Name
	})

	e.Patterns = loadedPatterns

	e.ProviderMap = make(map[string][]CompiledPattern)
	for _, p := range loadedPatterns {
		prov := strings.ToLower(p.Def.Provider)
		e.ProviderMap[prov] = append(e.ProviderMap[prov], p)
	}

	return nil
}

func cleanSMS(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var cleaned []string
	for _, l := range lines {
		cleaned = append(cleaned, strings.TrimRight(l, " \t\r"))
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func (e *Engine) loadIgnored(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var list []string
	if err := yaml.Unmarshal(content, &list); err != nil {
		return err
	}
	for _, s := range list {
		clean := cleanSMS(s)
		if clean != "" {
			e.IgnoredSet[clean] = struct{}{}
		}
	}
	return nil
}

func (e *Engine) loadExact(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var m map[string]string
	if err := yaml.Unmarshal(content, &m); err != nil {
		return err
	}
	for k, v := range m {
		clean := cleanSMS(k)
		if clean != "" {
			e.ExactPromos[clean] = v
		}
	}
	return nil
}

func parseOptions(flags string) regexp2.RegexOptions {
	opts := regexp2.None
	if strings.Contains(flags, "i") {
		opts |= regexp2.IgnoreCase
	}
	if strings.Contains(flags, "m") {
		opts |= regexp2.Multiline
	}
	if strings.Contains(flags, "s") {
		opts |= regexp2.Singleline
	}
	return opts
}

func compilePattern(def model.PatternDefinition) (CompiledPattern, error) {
	identOpts := parseOptions(def.Identifier.Flags)
	identRe, err := regexp2.Compile(def.Identifier.Pattern, identOpts)
	if err != nil {
		return CompiledPattern{}, fmt.Errorf("invalid identifier regex: %w", err)
	}

	fieldRes := make(map[string]CompiledRegex, len(def.Regexes))
	for name, rDef := range def.Regexes {
		var re *regexp2.Regexp
		if rDef.Pattern != "" {
			fOpts := parseOptions(rDef.Flags)
			var err error
			re, err = regexp2.Compile(rDef.Pattern, fOpts)
			if err != nil {
				return CompiledPattern{}, fmt.Errorf("invalid regex for field %s: %w", name, err)
			}
		}
		fieldRes[name] = CompiledRegex{
			Re:       re,
			Fallback: rDef.Fallback.Template,
			Optional: rDef.Optional,
		}
	}

	return CompiledPattern{
		Def:        def,
		Identifier: identRe,
		FieldRes:   fieldRes,
	}, nil
}

func (e *Engine) Parse(rawSMS string) (*model.Transaction, error) {
	return e.ParseWithProvider("", rawSMS)
}

func (e *Engine) ParseWithProvider(provider string, rawSMS string) (*model.Transaction, error) {
	clean := cleanSMS(rawSMS)
	if clean == "" {
		return nil, errors.New("empty message")
	}

	if _, ignored := e.IgnoredSet[clean]; ignored {
		return nil, ErrIgnored
	}

	if promoName, exists := e.ExactPromos[clean]; exists {
		return &model.Transaction{
			Provider:        "system",
			PatternName:     promoName,
			Category:        "promo",
			Direction:       model.DirectionIncoming,
			RawMessage:      clean,
			ExtractedAt:     time.Now().UTC(),
			ExtractedFields: map[string]string{"type": "promo"},
		}, nil
	}

	candidates := e.Patterns
	if provider != "" {
		if provPatterns, ok := e.ProviderMap[strings.ToLower(provider)]; ok {
			candidates = provPatterns
		}
	}

	for _, p := range candidates {
		matched, err := p.Identifier.MatchString(clean)
		if err != nil || !matched {
			continue
		}

		txn, err := e.extractTransaction(p, clean)
		if err != nil {
			continue
		}
		return txn, nil
	}

	return nil, ErrNoMatch
}

func (e *Engine) extractTransaction(p CompiledPattern, raw string) (*model.Transaction, error) {
	extracted := make(map[string]string)

	for name, f := range p.FieldRes {
		if f.Re != nil {
			m, err := f.Re.FindStringMatch(raw)
			if err == nil && m != nil {
				groups := m.Groups()
				val := ""
				for i := 1; i < len(groups); i++ {
					if groups[i].Length > 0 {
						val = groups[i].String()
						break
					}
				}
				if val == "" {
					val = m.String()
				}
				extracted[name] = strings.TrimSpace(val)
			}
		}
	}

	for name, f := range p.FieldRes {
		if extracted[name] == "" && f.Fallback != "" {
			resolved, ok := resolveFallback(f.Fallback, extracted)
			if ok {
				extracted[name] = resolved
			}
		}
	}

	for name, f := range p.FieldRes {
		if !f.Optional && extracted[name] == "" {
			return nil, fmt.Errorf("%w: %s", ErrMissingRequired, name)
		}
	}

	extDef := p.Def.Extraction

	principal, _ := parseAmount(extracted[extDef.Amounts.Principal])

	var totalFees float64
	for _, feeField := range extDef.Amounts.Fees {
		feeVal, _ := parseAmount(extracted[feeField])
		totalFees += feeVal
	}

	var balancePtr *float64
	if balStr := extracted[extDef.Amounts.Balance]; balStr != "" {
		if balVal, err := parseAmount(balStr); err == nil {
			balancePtr = &balVal
		}
	}

	var participants []model.Participant
	for _, partDef := range extDef.Participants {
		part := model.Participant{
			Name:  extracted[partDef.Fields.Name],
			Phone: extracted[partDef.Fields.Phone],
			Role:  partDef.Role,
			Type:  partDef.Type,
		}
		if part.Name != "" || part.Phone != "" {
			participants = append(participants, part)
		}
	}

	txnIDs := make(map[string]string)
	for prov, fieldName := range extDef.TransactionIDs {
		if val := extracted[fieldName]; val != "" {
			txnIDs[prov] = val
		}
	}

	totalAmount := principal
	if extDef.Direction == model.DirectionOutgoing {
		totalAmount += totalFees
	}

	var parsedTime *time.Time
	if rawTs := extracted[extDef.TimestampField]; rawTs != "" {
		for _, layout := range []string{
			"02/01/2006 15:04:05",
			"02/01/2006 15:04",
			"2006-01-02 15:04:05",
			"02/01/2006",
		} {
			if pt, err := time.Parse(layout, rawTs); err == nil {
				parsedTime = &pt
				break
			}
		}
	}

	return &model.Transaction{
		Provider:        p.Def.Provider,
		PatternName:     p.Def.Name,
		Category:        p.Def.Category,
		Direction:       extDef.Direction,
		Principal:       principal,
		Fees:            totalFees,
		TotalAmount:     totalAmount,
		Balance:         balancePtr,
		Participants:    participants,
		TransactionIDs:  txnIDs,
		ReceiptURL:      extracted["RECEIPT_URL"],
		RawTimestamp:    extracted[extDef.TimestampField],
		ParsedTime:      parsedTime,
		ExtractedFields: extracted,
		RawMessage:      raw,
		ExtractedAt:     time.Now().UTC(),
	}, nil
}

func resolveFallback(template string, fields map[string]string) (string, bool) {
	matches := placeholderRegex.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return template, true
	}

	result := template
	for _, match := range matches {
		placeholderKey := match[1]
		val, exists := fields[placeholderKey]
		if !exists || val == "" {
			return "", false
		}
		result = strings.ReplaceAll(result, "{"+placeholderKey+"}", val)
	}
	return result, true
}

func parseAmount(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	clean := strings.ReplaceAll(s, ",", "")
	clean = strings.TrimSpace(clean)
	return strconv.ParseFloat(clean, 64)
}
