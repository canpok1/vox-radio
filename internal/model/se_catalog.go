package model

type SECatalog struct {
	Names []string `json:"names"`
}

// AssetCatalog holds the available asset names for each category (SE, BGM, Jingle).
// Used to inform the LLM which assets are available for insertion.
type AssetCatalog struct {
	SE     []string `json:"se"`
	BGM    []string `json:"bgm"`
	Jingle []string `json:"jingle"`
}
