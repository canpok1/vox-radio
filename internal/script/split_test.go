package script_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script"
)

func TestSplitLinesIntoSentences_TextSplitting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "単一の終止記号で終わる文はそのまま1件",
			input: "これはテストです。",
			want:  []string{"これはテストです。"},
		},
		{
			name:  "複数文は終止記号で分割される",
			input: "こんにちは。元気ですか？",
			want:  []string{"こんにちは。", "元気ですか？"},
		},
		{
			name:  "連続する終止記号はまとめて1つの区切りとして扱う",
			input: "やった！？よかったね。",
			want:  []string{"やった！？", "よかったね。"},
		},
		{
			name:  "括弧の内側の終止記号では分割しない",
			input: "「大丈夫？」と聞かれた。",
			want:  []string{"「大丈夫？」と聞かれた。"},
		},
		{
			name:  "多重ネストの括弧内側でも分割しない",
			input: "（「入れ子？」だよ）ね。",
			want:  []string{"（「入れ子？」だよ）ね。"},
		},
		{
			name:  "終止記号直後の閉じ括弧はそこまで含めて区切る",
			input: "そうなの？」でも大丈夫。",
			want:  []string{"そうなの？」", "でも大丈夫。"},
		},
		{
			name:  "読点・三点リーダーでは分割しない",
			input: "ちょっと待って、うーん…そうだね。",
			want:  []string{"ちょっと待って、うーん…そうだね。"},
		},
		{
			name:  "終止記号を含まないセリフは分割せずそのまま1件",
			input: "これはテストです",
			want:  []string{"これはテストです"},
		},
		{
			name:  "分割後に空白のみになる断片は破棄する",
			input: "終わりだよ。 ",
			want:  []string{"終わりだよ。"},
		},
		{
			name:  "半角の終止記号も対象",
			input: "ファイル名はREADME.mdです!次はtest.goです?",
			want:  []string{"ファイル名はREADME.mdです!", "次はtest.goです?"},
		},
		{
			name:  "半角の閉じ括弧も終止記号直後に含めて区切る",
			input: "了解!)よろしく。",
			want:  []string{"了解!)", "よろしく。"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := script.SplitLinesIntoSentences([]model.Line{{SpeakerRole: "zundamon", Text: tt.input}})
			gotTexts := make([]string, len(got))
			for i, l := range got {
				gotTexts[i] = l.Text
			}
			if len(gotTexts) != len(tt.want) {
				t.Fatalf("got %d sentences %q, want %d sentences %q", len(gotTexts), gotTexts, len(tt.want), tt.want)
			}
			for i := range tt.want {
				if gotTexts[i] != tt.want[i] {
					t.Errorf("sentence[%d] = %q, want %q", i, gotTexts[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitLinesIntoSentences_InheritsSpeakerRoleAndStyle(t *testing.T) {
	input := []model.Line{
		{SpeakerRole: "zundamon", Style: "ノーマル", Text: "文1。文2。"},
		{SpeakerRole: "metan", Style: "あまあま", Text: "単文のみ"},
	}

	got := script.SplitLinesIntoSentences(input)

	want := []model.Line{
		{SpeakerRole: "zundamon", Style: "ノーマル", Text: "文1。"},
		{SpeakerRole: "zundamon", Style: "ノーマル", Text: "文2。"},
		{SpeakerRole: "metan", Style: "あまあま", Text: "単文のみ"},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d lines: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSplitLinesIntoSentences_EmptyInput(t *testing.T) {
	got := script.SplitLinesIntoSentences(nil)
	if len(got) != 0 {
		t.Errorf("got %d lines, want 0", len(got))
	}
}
