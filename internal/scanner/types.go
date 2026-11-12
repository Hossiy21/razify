package scanner

// ScanResult represents an individual secret or weak credential detection.
type ScanResult struct {
	Line   int    `json:"line"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
	Risk   string `json:"risk"` // "CRITICAL", "HIGH", "MEDIUM"
}

// ScanSummary aggregates risk counts across a scanned environment file.
type ScanSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Total    int `json:"total"`
}

