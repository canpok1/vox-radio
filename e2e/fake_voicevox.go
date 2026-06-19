//go:build e2e

package e2e

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
)

// fakeVoicevox は VOICEVOX Engine の /audio_query と /synthesis を模倣するモックサーバー。
// /synthesis は ffprobe が解析できる正規の WAV（無音）を返す。
type fakeVoicevox struct {
	server *httptest.Server
}

func newFakeVoicevox() *fakeVoicevox {
	f := &fakeVoicevox{}
	mux := http.NewServeMux()
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"0.14.7"`))
	})
	mux.HandleFunc("/audio_query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"accent_phrases": [],
			"speedScale": 1.0,
			"pitchScale": 0.0,
			"intonationScale": 1.0,
			"volumeScale": 1.0,
			"prePhonemeLength": 0.1,
			"postPhonemeLength": 0.1,
			"outputSamplingRate": 24000,
			"outputStereo": false
		}`))
	})
	mux.HandleFunc("/synthesis", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(buildSilentWAV(0.3))
	})
	f.server = httptest.NewServer(mux)
	return f
}

func (f *fakeVoicevox) URL() string { return f.server.URL }

func (f *fakeVoicevox) Close() { f.server.Close() }

// buildSilentWAV は PCM16/mono/24kHz の無音 WAV バイト列を生成する。
func buildSilentWAV(durationSec float64) []byte {
	const (
		sampleRate    = 24000
		bitsPerSample = 16
		numChannels   = 1
	)
	numSamples := int(float64(sampleRate) * durationSec)
	dataSize := numSamples * numChannels * bitsPerSample / 8

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))                                     // fmt chunk size
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))                                      // PCM
	_ = binary.Write(&buf, binary.LittleEndian, uint16(numChannels))                            //nolint:gosec // 定数
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))                             //nolint:gosec // 定数
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*numChannels*bitsPerSample/8)) // byte rate
	_ = binary.Write(&buf, binary.LittleEndian, uint16(numChannels*bitsPerSample/8))            // block align
	_ = binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))                          //nolint:gosec // 定数
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(dataSize)) //nolint:gosec // テスト用の小さな値
	buf.Write(make([]byte, dataSize))
	return buf.Bytes()
}
