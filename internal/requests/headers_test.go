package requests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAcceptEncoding(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   map[string]float64
	}{
		{
			name:   "single encoding",
			header: "gzip",
			want:   map[string]float64{"gzip": 1.0},
		},
		{
			name:   "multiple with q",
			header: "gzip;q=0.8, deflate;q=0.6",
			want:   map[string]float64{"gzip": 0.8, "deflate": 0.6},
		},
		{
			name:   "empty",
			header: "",
			want:   map[string]float64{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAcceptEncoding(tt.header)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseContentEncoding(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   []string
	}{
		{
			name:   "single",
			header: "gzip",
			want:   []string{"gzip"},
		},
		{
			name:   "multiple",
			header: "gzip, deflate",
			want:   []string{"gzip", "deflate"},
		},
		{
			name:   "empty",
			header: "",
			want:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseContentEncoding(tt.header)
			assert.Equal(t, tt.want, got)
		})
	}
}
