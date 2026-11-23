package config

import (
	"errors"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name                    string
		port, apiSecret, listen string
		wantPort, wantListen    string
		wantErr                 error
	}{
		{"defaults", "", "secret", "", "59232", "0.0.0.0", nil},
		{"explicit values", "8080", "secret", "127.0.0.1", "8080", "127.0.0.1", nil},
		{"missing API secret", "8080", "", "", "", "", ErrMissingAPISecret},
		{"whitespace API secret", "8080", " \t ", "", "", "", ErrMissingAPISecret},
		{"API secret with spaces kept as given", "", " secret ", "", "59232", "0.0.0.0", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.port, tt.apiSecret, tt.listen)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Load error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if cfg != nil {
					t.Errorf("Load returned config %+v alongside an error", cfg)
				}
				return
			}
			if cfg.Port != tt.wantPort || cfg.Listen != tt.wantListen || cfg.APISecret != tt.apiSecret {
				t.Errorf("Load = %+v, want port %q listen %q secret %q", cfg, tt.wantPort, tt.wantListen, tt.apiSecret)
			}
		})
	}
}
