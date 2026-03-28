package httpsign

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
)

// AddSignature sets X-Timestamp and X-Signature headers on req.
// body is the raw request body bytes (nil or empty for GET requests).
// If kp is nil, AddSignature is a no-op (headers are not added).
func AddSignature(req *http.Request, kp *identity.KeyPair, body []byte) error {
	if kp == nil {
		return nil
	}

	ts := time.Now().Unix()
	sig := kp.Sign(req.Method, req.URL.Path, body, ts)

	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Signature", base64.StdEncoding.EncodeToString(sig))
	return nil
}
