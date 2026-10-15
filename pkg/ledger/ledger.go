package ledger

import (
	"strings"
	"time"

	"odition/pkg/model"
)

type AccountSummary struct {
	Provider        string     `json:"provider"`
	LatestBalance   *float64   `json:"latest_balance,omitempty"`
	LatestTimestamp *time.Time `json:"latest_timestamp,omitempty"`
}

type CounterpartySummary struct {
	Name     string  `json:"name"`
	Phone    string  `json:"phone,omitempty"`
	Received float64 `json:"received"`
	Sent     float64 `json:"sent"`
}

type FinancialSummary struct {
	TotalIncome  float64                         `json:"total_income"`
	TotalExpense float64                         `json:"total_expense"`
	TotalFees    float64                         `json:"total_fees"`
	NetCashFlow  float64                         `json:"net_cash_flow"`
	Accounts     map[string]*AccountSummary      `json:"accounts"`
	People       map[string]*CounterpartySummary `json:"people"`
	Transactions []*model.Transaction            `json:"transactions"`
}

type Aggregator struct {
	UserName string
}

func NewAggregator(userName string) *Aggregator {
	if userName == "" {
		userName = "Yonathan"
	}
	return &Aggregator{UserName: userName}
}

func (agg *Aggregator) Aggregate(txns []*model.Transaction) *FinancialSummary {
	summary := &FinancialSummary{
		Accounts:     make(map[string]*AccountSummary),
		People:       make(map[string]*CounterpartySummary),
		Transactions: txns,
	}

	for _, t := range txns {
		if t.Category == "promo" {
			continue
		}

		if t.Direction == model.DirectionIncoming {
			summary.TotalIncome += t.Principal
		} else if t.Direction == model.DirectionOutgoing {
			summary.TotalExpense += t.TotalAmount
			summary.TotalFees += t.Fees
		}

		if t.Balance != nil {
			acc, exists := summary.Accounts[t.Provider]
			if !exists {
				acc = &AccountSummary{Provider: t.Provider}
				summary.Accounts[t.Provider] = acc
			}
			if acc.LatestTimestamp == nil || (t.ParsedTime != nil && t.ParsedTime.After(*acc.LatestTimestamp)) {
				acc.LatestBalance = t.Balance
				if t.ParsedTime != nil {
					acc.LatestTimestamp = t.ParsedTime
				}
			}
		}

		for _, p := range t.Participants {
			if p.Name == "" || agg.isSelf(p.Name) {
				continue
			}

			person, exists := summary.People[p.Name]
			if !exists {
				person = &CounterpartySummary{
					Name:  p.Name,
					Phone: p.Phone,
				}
				summary.People[p.Name] = person
			}
			if p.Phone != "" && person.Phone == "" {
				person.Phone = p.Phone
			}

			if t.Direction == model.DirectionIncoming {
				person.Received += t.Principal
			} else {
				person.Sent += t.Principal
			}
		}
	}

	summary.NetCashFlow = summary.TotalIncome - summary.TotalExpense
	return summary
}

func (agg *Aggregator) isSelf(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), agg.UserName)
}
