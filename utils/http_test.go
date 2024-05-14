package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReplaceAnySchemeWithHttp(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "add HTTP to no scheme",
			endpoint: "localhost:26657",
			want:     "http://localhost:26657",
		},
		{
			name:     "add HTTP to relative-scheme",
			endpoint: "://localhost:26657",
			want:     "http://localhost:26657",
		},
		{
			name:     "add HTTP to no-scheme",
			endpoint: "//localhost:26657",
			want:     "http://localhost:26657",
		},
		{
			name:     "keep HTTP",
			endpoint: "http://localhost:26657",
			want:     "http://localhost:26657",
		},
		{
			name:     "keep HTTPS",
			endpoint: "https://localhost:26657",
			want:     "https://localhost:26657",
		},
		{
			name:     "tcp should be replaced with http",
			endpoint: "tcp://localhost:26657",
			want:     "http://localhost:26657",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplaceAnySchemeWithHttp(tt.endpoint); got != tt.want {
				t.Errorf("ReplaceAnySchemeWithHttp() = %v, want %v", got, tt.want)
			}
		})
	}
}

//goland:noinspection HttpUrlsUsage
func TestNormalizeRpcEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "normal",
			endpoint: "http://google.com",
			want:     "http://google.com:80",
		},
		{
			name:     "normal TLS",
			endpoint: "https://google.com",
			want:     "https://google.com:443",
		},
		{
			name:     "normal ws",
			endpoint: "ws://google.com",
			want:     "ws://google.com:80",
		},
		{
			name:     "normal ws TLS",
			endpoint: "wss://google.com",
			want:     "wss://google.com:443",
		},
		{
			name:     "without protocol",
			endpoint: "google.com",
			want:     "google.com:80",
		},
		{
			name:     "without protocol, with sub-path",
			endpoint: "google.com/rpc",
			want:     "google.com:80/rpc",
		},
		{
			name:     "with suffix /",
			endpoint: "http://google.com/",
			want:     "http://google.com:80",
		},
		{
			name:     "normal TLS with suffix /",
			endpoint: "https://google.com/",
			want:     "https://google.com:443",
		},
		{
			name:     "with sub-path",
			endpoint: "http://google.com/rpc/",
			want:     "http://google.com:80/rpc",
		},
		{
			name:     "TLS with sub-path",
			endpoint: "https://google.com/rpc/",
			want:     "https://google.com:443/rpc",
		},
		{
			name:     "keep port",
			endpoint: "http://google.com:80",
			want:     "http://google.com:80",
		},
		{
			name:     "keep port",
			endpoint: "http://google.com:123",
			want:     "http://google.com:123",
		},
		{
			name:     "keep port",
			endpoint: "https://google.com:123",
			want:     "https://google.com:123",
		},
		{
			name:     "keep port",
			endpoint: "https://google.com:123/rpc/",
			want:     "https://google.com:123/rpc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, NormalizeRpcEndpoint(tt.endpoint))
		})
	}
}
