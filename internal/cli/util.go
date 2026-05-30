package cli

import (
	"fmt"

	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/publish"
	"github.com/canpok1/vox-radio/internal/publish/hosting"
	"github.com/canpok1/vox-radio/internal/publish/hosting/ghpages"
	"github.com/canpok1/vox-radio/internal/publish/hosting/local"
)

func writeJSON(path string, v any) error {
	return fileio.WriteJSON(path, v)
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
