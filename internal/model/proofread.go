package model

// ProofreadCorrection represents a single correction made by the proofreading pass.
type ProofreadCorrection struct {
	CornerIndex int    `json:"corner_index"`
	LineIndex   int    `json:"line_index"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Reason      string `json:"reason,omitempty"`
}

// ProofreadResult holds the result of a proofreading pass, including all corrections applied.
// Non-nil means the proofreading LLM ran successfully (even if Corrections is empty).
type ProofreadResult struct {
	Corrections []ProofreadCorrection `json:"corrections"`
}
