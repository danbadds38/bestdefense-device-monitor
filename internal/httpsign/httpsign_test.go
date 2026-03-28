package httpsign

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
)

func testKeyPair(t *testing.T) *identity.KeyPair {
	t.Helper()
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}
	return kp
}

// testCanonicalMessage replicates the signed message format for independent verification.
func testCanonicalMessage(method, path string, body []byte, timestamp int64) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%s\n%s\n%s\n%d", method, path, hex.EncodeToString(h[:]), timestamp)
}

func TestAddSignatureSetsTimestampHeader(t *testing.T) {
	kp := testKeyPair(t)
	req := httptest.NewRequest(http.MethodGet, "/agent/commands", nil)

	before := time.Now().Unix()
	if err := AddSignature(req, kp, nil); err != nil {
		t.Fatalf("AddSignature() error: %v", err)
	}
	after := time.Now().Unix()

	tsStr := req.Header.Get("X-Timestamp")
	if tsStr == "" {
		t.Fatal("X-Timestamp header not set")
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		t.Fatalf("X-Timestamp not a valid integer: %q", tsStr)
	}
	if ts < before || ts > after {
		t.Errorf("X-Timestamp %d outside expected range [%d, %d]", ts, before, after)
	}
}

func TestAddSignatureSetsSignatureHeader(t *testing.T) {
	kp := testKeyPair(t)
	req := httptest.NewRequest(http.MethodPost, "/agent/checkin", nil)

	if err := AddSignature(req, kp, []byte(`{"test":true}`)); err != nil {
		t.Fatalf("AddSignature() error: %v", err)
	}

	sig := req.Header.Get("X-Signature")
	if sig == "" {
		t.Fatal("X-Signature header not set")
	}
	decoded, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("X-Signature is not valid base64: %v", err)
	}
	if len(decoded) != ed25519.SignatureSize {
		t.Errorf("signature length = %d, want %d", len(decoded), ed25519.SignatureSize)
	}
}

func TestSignatureIsVerifiableWithPublicKey(t *testing.T) {
	kp := testKeyPair(t)
	body := []byte(`{"platform":"linux"}`)
	req := httptest.NewRequest(http.MethodPost, "/agent/checkin", nil)

	if err := AddSignature(req, kp, body); err != nil {
		t.Fatalf("AddSignature() error: %v", err)
	}

	ts, _ := strconv.ParseInt(req.Header.Get("X-Timestamp"), 10, 64)
	sig, _ := base64.StdEncoding.DecodeString(req.Header.Get("X-Signature"))

	msg := testCanonicalMessage(http.MethodPost, "/agent/checkin", body, ts)
	if !ed25519.Verify(kp.PublicKey, []byte(msg), sig) {
		t.Error("signature failed Ed25519 verification with the corresponding public key")
	}
}

func TestSignatureBodyHashIsIncluded(t *testing.T) {
	kp := testKeyPair(t)
	ts := time.Now().Unix()

	sig1 := kp.Sign(http.MethodPost, "/agent/checkin", []byte(`{"platform":"linux"}`), ts)
	sig2 := kp.Sign(http.MethodPost, "/agent/checkin", []byte(`{"platform":"windows"}`), ts)

	if string(sig1) == string(sig2) {
		t.Error("signatures are identical for different bodies; body hash is not covered")
	}
}

func TestSignatureTimestampIsIncluded(t *testing.T) {
	kp := testKeyPair(t)
	body := []byte(`{"platform":"linux"}`)

	sig1 := kp.Sign(http.MethodPost, "/agent/checkin", body, 1000000)
	sig2 := kp.Sign(http.MethodPost, "/agent/checkin", body, 1000001)

	if string(sig1) == string(sig2) {
		t.Error("signatures are identical for different timestamps; timestamp is not covered")
	}
}
