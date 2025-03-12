package model

type Violation struct {
	Category string  `json:"category"`
	RiskNum  float64 `json:"risk_num"`
	Reason   string  `json:"reason"`
}
