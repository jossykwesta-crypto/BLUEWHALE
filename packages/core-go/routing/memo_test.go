package routing

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMemoByteLimit(t *testing.T) {
	tests := []struct {
		name    string
		memo    string
		wantErr error
	}{
		{
			name:    "empty memo accepted",
			memo:    "",
			wantErr: nil,
		},
		{
			name:    "exactly 28 bytes accepted",
			memo:    strings.Repeat("a", 28),
			wantErr: nil,
		},
		{
			name:    "29 bytes rejected",
			memo:    strings.Repeat("a", 29),
			wantErr: ErrMemoTooLong,
		},
		{
			name:    "multi-byte UTF-8 counted by bytes",
			memo:    strings.Repeat("\u00e9", 14), // 14 runes, 28 bytes
			wantErr: nil,
		},
		{
			name:    "multi-byte UTF-8 exceeding 28 bytes rejected",
			memo:    strings.Repeat("\u00e9", 15), // 15 runes, 30 bytes
			wantErr: ErrMemoTooLong,
		},
		{
			name:    "4-byte rune boundary accepted",
			memo:    strings.Repeat("\U0001F600", 7), // 7 runes, 28 bytes
			wantErr: nil,
		},
		{
			name:    "4-byte rune boundary rejected",
			memo:    strings.Repeat("\U0001F600", 8), // 8 runes, 32 bytes
			wantErr: ErrMemoTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeMemo(tt.memo)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("normalizeMemo(%q) error = %v, want %v", tt.memo, err, tt.wantErr)
			}
		})
	}
}
