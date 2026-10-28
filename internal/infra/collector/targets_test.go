package collector

import (
	"reflect"
	"testing"
)

func TestParseHTTPTargets(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []HTTPTarget
		wantErr bool
	}{
		{name: "empty string", raw: "", want: nil},
		{
			name: "single target",
			raw:  "api=https://example.com/health",
			want: []HTTPTarget{{Name: "api", URL: "https://example.com/health"}},
		},
		{
			name: "multiple targets with spaces and trailing comma",
			raw:  " api = https://example.com/health , web=http://10.0.0.5:8080/ping,",
			want: []HTTPTarget{
				{Name: "api", URL: "https://example.com/health"},
				{Name: "web", URL: "http://10.0.0.5:8080/ping"},
			},
		},
		{
			name: "url with query string containing equals",
			raw:  "api=https://example.com/health?full=true",
			want: []HTTPTarget{{Name: "api", URL: "https://example.com/health?full=true"}},
		},
		{name: "missing separator", raw: "https://example.com", wantErr: true},
		{name: "missing name", raw: "=https://example.com", wantErr: true},
		{name: "missing url", raw: "api=", wantErr: true},
		{name: "unsupported scheme", raw: "api=ftp://example.com", wantErr: true},
		{name: "relative url", raw: "api=/health", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHTTPTargets(tt.raw)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
