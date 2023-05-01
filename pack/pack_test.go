package pack

import (
	"encoding/base64"
	"testing"
)

func TestPHPPack(t *testing.T) {
	cases := []struct {
		Format       string
		Args         []any
		Base64Result string
	}{
		{"L3", []any{1234, 5678, "777777"}, "0gQAAC4WAAAx3gsA"},
		{"L4", []any{"1234", "5678", "777777", "654321"}, "0gQAAC4WAAAx3gsA8fsJAA=="},
		{"C3", []any{byte(80), byte(72), byte(80)}, "UEhQ"},
		// phpt(base64)
		{"A5", []any{"foo "}, "Zm9vICA="},
		{"A4", []any{"fooo"}, "Zm9vbw=="},
		{"A4", []any{"foo"}, "Zm9vIA=="},
		// pack.phpt
		{"A9", []any{"hello"}, "aGVsbG8gICAg"},
		{"I", []any{-1000}, "GPz//w=="},
	}

	for i := range cases {
		result, err := PHPPack(cases[i].Format, cases[i].Args...)
		if err != nil {
			t.Errorf("pack failed: %v\n", err)
			return
		}

		base64Result := base64.StdEncoding.EncodeToString(result)
		if base64Result != cases[i].Base64Result {
			t.Errorf("pack failed, format: %s, args: %v, expected: %s, actual: %s\n", cases[i].Format, cases[i].Args, cases[i].Base64Result, base64Result)
			return
		}
	}
}
