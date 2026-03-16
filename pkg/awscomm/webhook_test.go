package awscomm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHMACSignature(t *testing.T) {
	secret := "my-secret-key"
	payload := `{"test":"data"}`

	signature := GenerateHMACSignature(payload, secret)

	// Signature should be 64 characters (SHA256 hex)
	assert.Len(t, signature, 64)
	assert.NotEmpty(t, signature)

	// Same input should produce same signature
	signature2 := GenerateHMACSignature(payload, secret)
	assert.Equal(t, signature, signature2)

	// Different payload should produce different signature
	differentPayload := `{"test":"different"}`
	differentSignature := GenerateHMACSignature(differentPayload, secret)
	assert.NotEqual(t, signature, differentSignature)

	// Different secret should produce different signature
	differentSecret := "different-secret"
	differentSecretSignature := GenerateHMACSignature(payload, differentSecret)
	assert.NotEqual(t, signature, differentSecretSignature)
}

func TestValidateWebhookSignature(t *testing.T) {
	secret := "my-secret-key"
	payload := `{"test":"data"}`

	signature := GenerateHMACSignature(payload, secret)

	// Valid signature should pass
	valid := ValidateWebhookSignature(payload, signature, secret)
	assert.True(t, valid)

	// Invalid signature should fail
	invalidSignature := "invalid-signature"
	valid = ValidateWebhookSignature(payload, invalidSignature, secret)
	assert.False(t, valid)

	// Wrong secret should fail
	valid = ValidateWebhookSignature(payload, signature, "wrong-secret")
	assert.False(t, valid)

	// Modified payload should fail
	modifiedPayload := `{"test":"modified"}`
	valid = ValidateWebhookSignature(modifiedPayload, signature, secret)
	assert.False(t, valid)
}

func TestValidateWebhookRequest(t *testing.T) {
	secret := "my-secret-key"
	payload := `{"test":"data"}`
	signature := GenerateHMACSignature(payload, secret)

	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		wantValid bool
		wantError bool
	}{
		{
			name:      "valid request",
			payload:   []byte(payload),
			signature: signature,
			secret:    secret,
			wantValid: true,
			wantError: false,
		},
		{
			name:      "missing signature",
			payload:   []byte(payload),
			signature: "",
			secret:    secret,
			wantValid: false,
			wantError: true,
		},
		{
			name:      "missing secret",
			payload:   []byte(payload),
			signature: signature,
			secret:    "",
			wantValid: false,
			wantError: true,
		},
		{
			name:      "invalid signature",
			payload:   []byte(payload),
			signature: "invalid",
			secret:    secret,
			wantValid: false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := ValidateWebhookRequest(tt.payload, tt.signature, tt.secret)
			assert.Equal(t, tt.wantValid, valid)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseWebhookPayload(t *testing.T) {
	secret := "my-secret-key"

	webhookData := WebhookPayload{
		Payload: map[string]interface{}{
			"order_number": "1111-1111-1111",
			"phone":        "+1234567890",
		},
		Metadata: map[string]string{
			"request_id": "abc123",
		},
		Type:         "internal",
		CommType:     "sms",
		Status:       "QUEUED",
		FailedReason: "",
	}

	payloadBytes, _ := json.Marshal(webhookData)
	signature := GenerateHMACSignature(string(payloadBytes), secret)

	// Valid request should parse successfully
	parsed, err := ParseWebhookPayload(payloadBytes, signature, secret)
	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Equal(t, "internal", parsed.Type)
	assert.Equal(t, "sms", parsed.CommType)
	assert.Equal(t, "QUEUED", parsed.Status)

	// Invalid signature should fail
	_, err = ParseWebhookPayload(payloadBytes, "invalid-signature", secret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook signature")

	// Missing signature should fail
	_, err = ParseWebhookPayload(payloadBytes, "", secret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing X-Webhook-Signature")
}

func TestGenerateWebhookPayload(t *testing.T) {
	secret := "my-secret-key"

	payload := map[string]interface{}{
		"order_number": "1111-1111-1111",
	}
	metadata := map[string]string{
		"request_id": "abc123",
	}

	body := WebhookPayload{
		Payload:      payload,
		Metadata:     metadata,
		Type:         "internal",
		CommType:     "sms",
		Status:       "QUEUED",
		FailedReason: "",
	}

	payloadJSON, signature, err := GenerateWebhookPayload(body,
		secret,
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, payloadJSON)
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64)

	// Validate that the generated signature is correct
	valid := ValidateWebhookSignature(payloadJSON, signature, secret)
	assert.True(t, valid)

	// Parse the generated payload
	var webhook WebhookPayload
	err = json.Unmarshal([]byte(payloadJSON), &webhook)
	assert.NoError(t, err)
	assert.Equal(t, "internal", webhook.Type)
	assert.Equal(t, "sms", webhook.CommType)
	assert.Equal(t, "QUEUED", webhook.Status)
}
