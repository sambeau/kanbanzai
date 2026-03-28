package buildinfo

import "testing"

func TestDefaultValues(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Version", Version, "dev"},
		{"GitSHA", GitSHA, "unknown"},
		{"BuildTime", BuildTime, "unknown"},
		{"Dirty", Dirty, "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}
