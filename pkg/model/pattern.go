package model

import "gopkg.in/yaml.v3"

type Direction string

const (
	DirectionIncoming Direction = "INCOMING"
	DirectionOutgoing Direction = "OUTGOING"
)

type IdentifierDef struct {
	Pattern string `yaml:"pattern"`
	Flags   string `yaml:"flags"`
}

type FallbackDef struct {
	Template string `yaml:"template"`
}

type RegexDef struct {
	Pattern  string      `yaml:"pattern"`
	Flags    string      `yaml:"flags"`
	Source   string      `yaml:"source"`
	Fallback FallbackDef `yaml:"fallback"`
	Optional bool        `yaml:"optional"`
}

func (r *RegexDef) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.Pattern = value.Value
		return nil
	}
	type alias RegexDef
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*r = RegexDef(a)
	return nil
}

type ParticipantFields struct {
	Name  string `yaml:"name"`
	Phone string `yaml:"phone"`
}

type ParticipantDef struct {
	Fields ParticipantFields `yaml:"fields"`
	Role   string            `yaml:"role"`
	Type   string            `yaml:"type"`
}

type AmountsDef struct {
	Principal string   `yaml:"principal"`
	Fees      []string `yaml:"fees"`
	Balance   string   `yaml:"balance"`
}

type ExtractionDef struct {
	Direction      Direction         `yaml:"direction"`
	Amounts        AmountsDef        `yaml:"amounts"`
	Participants   []ParticipantDef  `yaml:"participants"`
	TransactionIDs map[string]string `yaml:"transactionIds"`
	TimestampField string            `yaml:"timestampField"`
}

type PatternDefinition struct {
	Name       string              `yaml:"name"`
	Provider   string              `yaml:"provider"`
	Category   string              `yaml:"category"`
	Order      int                 `yaml:"order"`
	Status     string              `yaml:"status"`
	Identifier IdentifierDef       `yaml:"identifier"`
	Regexes    map[string]RegexDef `yaml:"regexes"`
	Extraction ExtractionDef       `yaml:"extraction"`
}
