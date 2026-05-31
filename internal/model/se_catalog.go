package model

// AssetCatalogEntry represents a single audio asset entry passed to the LLM.
// Description is optional; when empty it is omitted from JSON (omitempty).
type AssetCatalogEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AssetCatalog holds the available asset entries for each category (SE, BGM, Jingle).
// Used to inform the LLM which assets are available for insertion.
type AssetCatalog struct {
	SE     []AssetCatalogEntry `json:"se"`
	BGM    []AssetCatalogEntry `json:"bgm"`
	Jingle []AssetCatalogEntry `json:"jingle"`
}
