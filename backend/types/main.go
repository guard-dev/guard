package types

type SummarizeFindings struct {
	Summary  string   `json:"summary"`
	Remedies string   `json:"remedies"`
	Commands []string `json:"commands"`
}
