// Package e2e は vox-radio CLI の BDD（Gherkin）ベース e2e テストを提供する。
//
// テスト本体は e2e ビルドタグで保護されており、通常の `go test ./...` からは除外される。
// 実行は `make e2e`（go test -tags=e2e ./e2e/...）で行う。
// シナリオ定義は features/*.feature（日本語 Gherkin）にあり、godog がそのまま実行する。
//
// 外部依存（LLM / VOICEVOX / RSS フィード / Slack API）は httptest のモックサーバーで
// 差し替える。ffmpeg / ffprobe のみ実バイナリを使い、見つからない環境では
// @ffmpeg タグ付きシナリオを自動的にスキップする。
package e2e
