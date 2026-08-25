package direct

import "testing"

func TestSanitizeSpeechText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// The observed failure: OpenJTalk reads 「エーダブリューエス（AWS）」 as
			// エーダブリューエス（エーダブリューエス）, i.e. the same word twice.
			name: "全角括弧の原綴り併記を落とす",
			in:   "動画配信サービスなんて、エーダブリューエス（AWS）を使えば作れるのだ！",
			want: "動画配信サービスなんて、エーダブリューエスを使えば作れるのだ！",
		},
		{
			name: "半角括弧も対象",
			in:   "キータ(Qiita)の人気記事です。",
			want: "キータの人気記事です。",
		},
		{
			name: "1行に複数あってもすべて落とす",
			in:   "イーシーツー（EC2）やエススリー（S3）で進んだそうですわ。",
			want: "イーシーツーやエススリーで進んだそうですわ。",
		},
		{
			name: "括弧内が読みでも落とす",
			in:   "スキルズとエバル（イーバル）の設計です。",
			want: "スキルズとエバルの設計です。",
		},
		{
			name: "鉤括弧は記事タイトル・引用なので残す",
			in:   "「家族で使うエーアイエージェント」という記事ですわ。",
			want: "「家族で使うエーアイエージェント」という記事ですわ。",
		},
		{
			name: "二重鉤括弧も残す",
			in:   "『勝てないと悟った話』です。",
			want: "『勝てないと悟った話』です。",
		},
		{
			name: "括弧が無ければ元のまま",
			in:   "こんにちはなのだ！",
			want: "こんにちはなのだ！",
		},
		{
			name: "入れ子は外側ごと落とす",
			in:   "オーエスエス（オープンソース（OSS））として公開されています。",
			want: "オーエスエスとして公開されています。",
		},
		{
			name: "閉じない括弧は行末を飲み込まず残す",
			in:   "これは（閉じ忘れなのだ",
			want: "これは（閉じ忘れなのだ",
		},
		{
			name: "除去で生じた空白は詰める",
			in:   "シンクレット（THINKLET） と エーピーアイ（API） の話ですわ。",
			want: "シンクレット と エーピーアイ の話ですわ。",
		},
		{
			// The LLM writes the text; nothing keeps the bracket widths consistent.
			name: "全角開き・半角閉じの混在も落とす",
			in:   "キータ（Qiita)の人気記事です。",
			want: "キータの人気記事です。",
		},
		{
			name: "半角開き・全角閉じの混在も落とす",
			in:   "キータ(Qiita）の人気記事です。",
			want: "キータの人気記事です。",
		},
		{
			name: "読める文字が残らない行は合成できなくなるため元のまま",
			in:   "（笑）。",
			want: "（笑）。",
		},
		{
			name: "全体が括弧だけの行は合成できなくなるため元のまま",
			in:   "（笑）",
			want: "（笑）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeSpeechText(tt.in); got != tt.want {
				t.Errorf("sanitizeSpeechText(%q):\n got %q\nwant %q", tt.in, got, tt.want)
			}
		})
	}
}
