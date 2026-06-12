package model_test

import (
	"encoding/json"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestNonNil(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantNil   bool
		wantLen   int
		wantItems []string
	}{
		{
			name:    "nil slice returns non-nil empty slice",
			input:   nil,
			wantNil: false,
			wantLen: 0,
		},
		{
			name:    "non-nil empty slice returns non-nil empty slice",
			input:   make([]string, 0),
			wantNil: false,
			wantLen: 0,
		},
		{
			name:      "non-empty slice returns same content",
			input:     []string{"a", "b", "c"},
			wantNil:   false,
			wantLen:   3,
			wantItems: []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.NonNil(tt.input)
			if (got == nil) != tt.wantNil {
				t.Errorf("NonNil() nil = %v, want %v", got == nil, tt.wantNil)
			}
			if len(got) != tt.wantLen {
				t.Errorf("NonNil() len = %d, want %d", len(got), tt.wantLen)
			}
			for i, item := range tt.wantItems {
				if got[i] != item {
					t.Errorf("NonNil()[%d] = %q, want %q", i, got[i], item)
				}
			}
		})
	}
}

func TestNonNil_nil_marshals_as_empty_array(t *testing.T) {
	type wrapper struct {
		Items []string `json:"items"`
	}
	var s []string
	w := wrapper{Items: model.NonNil(s)}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(data) != `{"items":[]}` {
		t.Errorf("NonNil nil result: got %s, want {\"items\":[]}", string(data))
	}
}
