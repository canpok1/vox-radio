package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/publish"
	"github.com/canpok1/vox-radio/internal/publish/hosting"
	"github.com/canpok1/vox-radio/internal/publish/hosting/ghpages"
	"github.com/canpok1/vox-radio/internal/publish/hosting/local"
)

func writeJSON(path string, v any) error {
	return fileio.WriteJSON(path, v)
}

func loadConfigAndProfile(profilePath string) (*config.Config, *config.Profile, error) {
	cfg, err := config.LoadConfig("vox-radio.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	p, err := config.LoadProfile(profilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("load profile: %w", err)
	}
	if err := config.ValidateProfileCast(p, cfg.Characters); err != nil {
		return nil, nil, fmt.Errorf("profile validation: %w", err)
	}
	return cfg, p, nil
}

func resolveSiteURL(override, configURL string) string {
	if override != "" {
		return override
	}
	return configURL
}

func resolveKeep(maxItems int) int {
	if maxItems <= 0 {
		return publish.DefaultKeep
	}
	return maxItems
}

func newHosting(hostingType, outDir, siteURL string) (hosting.Hosting, error) {
	switch hostingType {
	case "local":
		return local.New(outDir, siteURL), nil
	case "ghpages":
		return ghpages.New(outDir, siteURL), nil
	default:
		return nil, fmt.Errorf("unknown hosting type %q: must be \"local\" or \"ghpages\"", hostingType)
	}
}
