package awscomm

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type WebhookPayload struct {
	Payload      interface{} `json:"payload"`
	Metadata     interface{} `json:"metadata"`
	Type         string      `json:"type"`         // "internal" or "external"
	CommType     string      `json:"commType"`     // e.g., "sms", "email", "fax", "voice_mail"
	Status       string      `json:"status"`       // e.g., "QUEUED", "FAILED"
	FailedReason string      `json:"failedReason"` // Error message if failed
}

// GenerateHMACSignature generates HMAC SHA256 signature for webhook authentication
func GenerateHMACSignature(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateWebhookSignature validates the HMAC signature from a webhook request
// It compares the provided signature with the expected signature generated from the payload
func ValidateWebhookSignature(payloadJSON string, signature string, secret string) bool {
	expectedSignature := GenerateHMACSignature(payloadJSON, secret)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ValidateWebhookRequest validates a webhook request by checking the signature
// payloadBytes: The raw request body (JSON)
// signature: The signature from X-Webhook-Signature header
// secret: The secret key used for HMAC
func ValidateWebhookRequest(payloadBytes []byte, signature string, secret string) (bool, error) {
	if signature == "" {
		return false, NewError("missing X-Webhook-Signature header")
	}

	if secret == "" {
		return false, NewError("webhook secret not configured")
	}

	payloadString := string(payloadBytes)
	return ValidateWebhookSignature(payloadString, signature, secret), nil
}

// ParseWebhookPayload parses and validates a webhook request
// Returns the parsed payload if signature is valid
func ParseWebhookPayload(payloadBytes []byte, signature string, secret string) (*WebhookPayload, error) {
	valid, err := ValidateWebhookRequest(payloadBytes, signature, secret)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, NewError("invalid webhook signature")
	}

	var webhook WebhookPayload
	if err := json.Unmarshal(payloadBytes, &webhook); err != nil {
		return nil, WrapError(err, "failed to parse webhook payload")
	}

	return &webhook, nil
}

// GenerateWebhookPayload generates a webhook payload JSON string with signature
// This is useful for testing or when you need to generate webhook calls
func GenerateWebhookPayload(payload, metadata interface{}, webhookType, commType, status, failedReason, secret string) (payloadJSON, signature string, err error) {
	body := WebhookPayload{
		Payload:      payload,
		Metadata:     metadata,
		Type:         webhookType,
		CommType:     commType,
		Status:       status,
		FailedReason: failedReason,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", "", WrapError(err, "failed to marshal webhook payload")
	}

	payloadJSON = string(bodyBytes)
	signature = GenerateHMACSignature(payloadJSON, secret)

	return payloadJSON, signature, nil
}
