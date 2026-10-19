package config

type Topology struct {
	Families []Family `json:"families"`
}
type Family struct {
	ID      string   `json:"id"`
	Code    uint8    `json:"code"`
	Anchors []string `json:"anchors"`
}
