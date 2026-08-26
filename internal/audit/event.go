package audit

import "time"

type Severity string

const (
	Info     Severity = "info"
	Warning  Severity = "warning"
	Critical Severity = "critical"
)

type Event struct {
	ID        string         `json:"id"`
	At        time.Time      `json:"at"`
	Component string         `json:"component"`
	Action    string         `json:"action"`
	Severity  Severity       `json:"severity"`
	Operation string         `json:"operation,omitempty"`
	Subject   string         `json:"subject,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type Query struct {
	Component string
	Subject   string
	Limit     int
}
