package infrastructure

import (
	"strings"
	"testing"
)

func TestSecretBoxRoundTrip(t *testing.T) {
	box, err := NewSecretBox("test-master-key-32-bytes-minimum!")
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	secret := "sk-1234567890abcdef"
	blob, err := box.Encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if strings.Contains(blob, secret) {
		t.Fatal("ciphertext leaked plaintext")
	}
	got, err := box.Decrypt(blob)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != secret {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestSecretBoxTamperDetection(t *testing.T) {
	box, _ := NewSecretBox("test-master-key-32-bytes-minimum!")
	blob, err := box.Encrypt("secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	tampered := blob[:len(blob)-2] + "xx"
	if _, err := box.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext decrypted successfully")
	}
}

func TestSecretBoxFailsClosedWithoutMasterKey(t *testing.T) {
	if _, err := NewSecretBox(""); err == nil {
		t.Fatal("expected error for empty master key")
	}
}

func TestSecretBoxRejectsWrongKeyVersion(t *testing.T) {
	box, _ := NewSecretBox("test-master-key-32-bytes-minimum!")
	blob := "v99.abc"
	if _, err := box.Decrypt(blob); err == nil || !strings.Contains(err.Error(), "key version") {
		t.Fatalf("expected key version error, got %v", err)
	}
}

func TestValidateEndpoint(t *testing.T) {
	cases := []struct {
		endpoint   string
		allowLocal bool
		wantErr    bool
	}{
		{"https://api.openai.com/v1", false, false},
		{"http://api.openai.com/v1", false, true},  // non-loopback http
		{"http://localhost:11434/v1", false, true}, // loopback without flag
		{"http://localhost:11434/v1", true, false}, // loopback with flag
		{"http://127.0.0.1:8080/v1", true, false},  // loopback IP with flag
		{"http://127.0.0.1:8080/v1", false, true},
		{"http://169.254.169.254/latest/meta-data", true, true}, // link-local metadata
		{"file:///etc/passwd", false, true},                     // bad scheme
		{"https://", false, true},                               // no host
		{"http://192.168.1.5:8080", true, true},                 // private network
	}
	for _, c := range cases {
		err := ValidateEndpoint(c.endpoint, c.allowLocal)
		if c.wantErr && err == nil {
			t.Errorf("ValidateEndpoint(%q, %v): expected error", c.endpoint, c.allowLocal)
		}
		if !c.wantErr && err != nil {
			t.Errorf("ValidateEndpoint(%q, %v): unexpected error %v", c.endpoint, c.allowLocal, err)
		}
	}
}
