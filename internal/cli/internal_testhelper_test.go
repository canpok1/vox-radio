package cli

import (
	"os"
	"testing"
)

// chdirTemp は一時ディレクトリを作成してそこへ cwd を移動し、作成した
// ディレクトリのパスを返す。テスト終了時に元の cwd へ自動で復帰する。
// os.Chdir はプロセス全体の cwd を変更するため、利用するテストは並列禁止。
func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}
