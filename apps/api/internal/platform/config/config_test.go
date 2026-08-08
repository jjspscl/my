package config

import "testing"

func TestLoadSecureCookiesExplicitValue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "true", value: "true", want: true},
		{name: "false", value: "false", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MY_SECURE_COOKIES", tc.value)
			t.Setenv("MY_WEB_URL", "http://localhost:5173")

			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.SecureCookies != tc.want {
				t.Fatalf("SecureCookies = %v, want %v", cfg.SecureCookies, tc.want)
			}
		})
	}
}

func TestLoadSecureCookiesRejectsInvalidValue(t *testing.T) {
	t.Setenv("MY_SECURE_COOKIES", "maybe")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid boolean error")
	}
}

func TestLoadSecureCookiesDerivesFromWebURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		url  string
		want bool
	}{
		{name: "https", url: "https://my.example", want: true},
		{name: "http", url: "http://localhost:5173", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MY_WEB_URL", tc.url)
			t.Setenv("MY_SECURE_COOKIES", "")

			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.SecureCookies != tc.want {
				t.Fatalf("SecureCookies = %v, want %v", cfg.SecureCookies, tc.want)
			}
		})
	}
}
