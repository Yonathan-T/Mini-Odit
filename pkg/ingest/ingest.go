package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"odition/pkg/engine"
	"odition/pkg/model"
)

type IngestResult struct {
	TotalParsed     int
	TotalIgnored    int
	TotalUnmatched  int
	TotalScanned    int
	TotalFastRouted int
	TotalFallbacks  int
	Duration        time.Duration
	Transactions    []*model.Transaction
	Unmatched       []string
}

type Ingester struct {
	engine *engine.Engine
}

func NewIngester(eng *engine.Engine) *Ingester {
	return &Ingester{engine: eng}
}

func (in *Ingester) SplitChunks(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	rawContent := string(data)
	chunks := strings.Split(rawContent, "---")
	var result []string
	for _, chunk := range chunks {
		clean := strings.TrimSpace(chunk)
		if clean != "" {
			result = append(result, clean)
		}
	}
	return result, nil
}

func (in *Ingester) IngestChunks(chunks []string) *IngestResult {
	start := time.Now()
	res := &IngestResult{
		TotalScanned: len(chunks),
	}

	for _, clean := range chunks {
		trace := in.engine.ParseWithTrace(clean)
		if trace.Err != nil {
			if errors.Is(trace.Err, engine.ErrIgnored) {
				res.TotalIgnored++
			} else {
				res.TotalUnmatched++
				res.Unmatched = append(res.Unmatched, clean)
			}
			continue
		}
		if trace.FallbackUsed {
			res.TotalFallbacks++
		} else if trace.RoutedProvider != "" {
			res.TotalFastRouted++
		}
		res.TotalParsed++
		res.Transactions = append(res.Transactions, trace.Txn)
	}

	res.Duration = time.Since(start)
	return res
}

func (in *Ingester) IngestFile(filePath string) (*IngestResult, error) {
	chunks, err := in.SplitChunks(filePath)
	if err != nil {
		return nil, err
	}

	res := in.IngestChunks(chunks)

	if len(res.Unmatched) > 0 {
		outPath := filepath.Join(filepath.Dir(filePath), "unmatched.txt")
		_ = os.WriteFile(outPath, []byte(strings.Join(res.Unmatched, "\n---\n")), 0644)
	}

	return res, nil
}
