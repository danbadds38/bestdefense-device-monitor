package serverkey

import "encoding/base64"

// PublicKeyBase64 is the BestDefense server Ed25519 signing public key.
// This is the TEST key (seed = 0x43 * 32) used in development and automated tests.
// For production releases, override at link time:
//
//	go build -ldflags "-X '.../serverkey.PublicKeyBase64=REAL_KEY_BASE64'" ...
//
// The public key is intentionally visible — it is designed to be public.
// Only BestDefense (who holds the private key) can produce a valid signature.
var PublicKeyBase64 = "Ivwpd5Lwtv/Av8/bftsMCqFOAlo2XsDjQuhuOCnLdLY="

// PublicKey decodes and returns the raw 32-byte Ed25519 public key.
// Panics at startup if PublicKeyBase64 is not a valid 32-byte base64 value,
// so misconfiguration is caught immediately rather than at first verification.
func PublicKey() []byte {
	b, err := base64.StdEncoding.DecodeString(PublicKeyBase64)
	if err != nil || len(b) != 32 {
		panic("serverkey: PublicKeyBase64 is not a valid 32-byte Ed25519 public key")
	}
	return b
}
