package secret

import "testing"

func TestMaskAPIKey(t *testing.T) {
	got := MaskAPIKey("wrk-xxxxxxxxafaf")
	if got != "wrk-****afaf" {
		t.Fatalf("got %q", got)
	}
	if MaskAPIKey("") != "" {
		t.Fatal("empty")
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	ct, err := Encrypt(key, "wrk-secret")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "wrk-secret" {
		t.Fatalf("got %q", plain)
	}
}
