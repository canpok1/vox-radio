package config

import (
	"os"
	"strings"
)

// SlackAPIURLEnv は Slack API のベース URL を上書きする環境変数名（テスト・検証用）。
const SlackAPIURLEnv = "VOX_RADIO_SLACK_API_URL"

// SlackConfig holds Slack integration settings shared across programs.
type SlackConfig struct {
	BotTokenEnv string `yaml:"bot_token_env"`
}

// EffectiveAPIURL は環境変数で指定された Slack API のベース URL を返す。
// 未設定なら空文字を返し、slack-go のデフォルト URL が使われる。
// slack-go はベース URL とメソッド名を単純連結するため、末尾スラッシュを補正する。
func (c SlackConfig) EffectiveAPIURL() string {
	v := os.Getenv(SlackAPIURLEnv)
	if v == "" {
		return ""
	}
	if !strings.HasSuffix(v, "/") {
		v += "/"
	}
	return v
}
