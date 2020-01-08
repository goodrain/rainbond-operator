package types

// HistoryStatus history status about processing installation history.
type HistoryStatus string

const (
	// HistoryStatusTrue means a successful history
	HistoryStatusTrue HistoryStatus = "True"
	// HistoryStatusFalse means a failed history
	HistoryStatusFalse HistoryStatus = "False"
)

// ExtractionHistory represents one extraction history.
type ExtractionHistory struct {
	Status HistoryStatus `json:"status"`
	Reason string        `json:"reason"`
}
