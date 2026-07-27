package cli

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/retro"
)

func TestRetroTryItems_MissingTryFileYieldsNoInjection(t *testing.T) {
	dir := chdirTemp(t)
	tf, err := retro.LoadTryFile(programTryPath("no-such-program"))
	if err != nil {
		t.Fatalf("LoadTryFile: %v", err)
	}
	_ = dir

	items := retroTryItems(tf)
	if items != nil {
		t.Errorf("retroTryItems for missing try file = %+v, want nil (injection stops)", items)
	}
}

func TestRetroTryItems_EmptyProblemsYieldsNoInjection(t *testing.T) {
	items := retroTryItems(retro.TryFile{Problems: []retro.Problem{}})
	if items != nil {
		t.Errorf("retroTryItems for empty problems = %+v, want nil (injection stops)", items)
	}
}

func TestRetroTryItems_PopulatesFromProblems(t *testing.T) {
	tf := retro.TryFile{Problems: []retro.Problem{
		{ID: "p1", Problem: "説明調", Action: "疑問形で崩す"},
	}}

	items := retroTryItems(tf)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Problem != "説明調" || items[0].Action != "疑問形で崩す" {
		t.Errorf("items[0] = %+v, unexpected", items[0])
	}
}
