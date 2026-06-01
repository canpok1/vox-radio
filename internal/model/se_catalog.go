package model

// AssetCatalogEntry represents a single audio asset entry passed to the LLM.
// Description is optional; when empty it is omitted from JSON (omitempty).
type AssetCatalogEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AssetCatalog holds the available SE asset entries for the LLM.
// BGM and Jingle are now configured per-corner in the profile and are not passed to the LLM.
type AssetCatalog struct {
	SE []AssetCatalogEntry `json:"se"`
}
