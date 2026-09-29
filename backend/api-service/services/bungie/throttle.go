package bungie

import (
	"encoding/json"
	"errors"
	"time"
)

// ErrRateLimited reports that Bungie kept answering HTTP 429 until the retry
// budget ran out. An envelope throttle is reported as its *BungieError
// instead, so callers can read the code and ThrottleSeconds.
var ErrRateLimited = errors.New("bungie: rate limited")

// maxThrottleRetryWait bounds how long one request will wait on a throttle
// before retrying. A longer requested wait is handed back to the caller
// rather than held open in a handler.
const maxThrottleRetryWait = 10 * time.Second

// IsThrottle reports whether code is one of the throttle PlatformErrorCodes
// Bungie's OpenAPI specification defines: ThrottleLimitExceeded (31) and its
// Minutes/Momentarily/Seconds variants (35–37), PerEndpointRequestThrottle-
// Exceeded (51), the three PerApplication throttles (54–56), and
// PerUserThrottleExceeded (57). The specification gives none of them a
// threshold, which is why every one is treated alike.
func IsThrottle(code int) bool {
	switch code {
	case 31, 35, 36, 37, 51, 54, 55, 56, 57:
		return true
	}
	return false
}

// envelopeThrottle returns the throttle carried by a response body's
// envelope, or nil. Only the envelope fields are decoded, so a large
// successful payload is scanned once and not materialized.
func envelopeThrottle(body []byte) *BungieError {
	var e struct {
		ErrorCode       int
		ErrorStatus     string
		Message         string
		ThrottleSeconds int
	}
	if json.Unmarshal(body, &e) != nil || !IsThrottle(e.ErrorCode) {
		return nil
	}
	return &BungieError{ErrorCode: e.ErrorCode, ErrorStatus: e.ErrorStatus, Message: e.Message, ThrottleSeconds: e.ThrottleSeconds}
}
