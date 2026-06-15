package cli

import (
	"fmt"
	"os/exec"
	"strings"
)

const readmeURL = "https://github.com/canpok1/vox-radio#readme"

// lookPath is exec.LookPath by default; replaced in tests.
var lookPath = exec.LookPath

// requireMediaTools checks that ffmpeg and ffprobe are available in PATH.
// If any are missing, returns an error listing all absent tools with a link to the README.
func requireMediaTools() error {
	tools := []string{"ffmpeg", "ffprobe"}
	var missing []string
	for _, tool := range tools {
		if _, err := lookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	names := strings.Join(missing, ", ")
	return fmt.Errorf("%s が見つかりません。音声の生成には ffmpeg および ffprobe が必要です。\nインストール手順は vox-radio の README を参照してください:\n%s", names, readmeURL)
}
