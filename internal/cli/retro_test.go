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

func TestRetroKeepItems_MissingKeepFileYieldsNoInjection(t *testing.T) {
	chdirTemp(t)
	kf, err := retro.LoadKeepFile(programKeepPath("no-such-program"))
	if err != nil {
		t.Fatalf("LoadKeepFile: %v", err)
	}

	items := retroKeepItems(kf)
	if items != nil {
		t.Errorf("retroKeepItems for missing keep file = %+v, want nil (injection stops)", items)
	}
}

func TestRetroKeepItems_EmptyKeepsYieldsNoInjection(t *testing.T) {
	items := retroKeepItems(retro.KeepFile{Keeps: []retro.Keep{}})
	if items != nil {
		t.Errorf("retroKeepItems for empty keeps = %+v, want nil (injection stops)", items)
	}
}

func TestRetroKeepItems_PopulatesFromKeeps(t *testing.T) {
	kf := retro.KeepFile{Keeps: []retro.Keep{
		{ID: "p1", Problem: "説明調", Action: "疑問形で崩す", ProvenAtEpisode: 10},
	}}

	items := retroKeepItems(kf)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Problem != "説明調" || items[0].Action != "疑問形で崩す" {
		t.Errorf("items[0] = %+v, unexpected", items[0])
	}
}
