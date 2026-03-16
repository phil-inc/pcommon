# AWS Communication Client SDK

A Go client SDK for AWS Communication API supporting SMS, Voice, Email, and Fax services.

## Installation

```bash
go get github.com/yourusername/pcommon/pkg/awscomm
```

## Usage

### Initialize Client

```go
import "github.com/yourusername/pcommon/pkg/awscomm"

client := awscomm.NewClient("https://api.example.com", "your-api-key")
```

### Send SMS

```go
ctx := context.Background()
request := &awscomm.SMSRequest{
    CallbackURL: "https://callback.example.com",
    Payload: awscomm.SMSPayload{
        ToPhoneNumber: "+1234567890",
        Message:       "Hello, World!",
    },
}

response, err := client.SendSMS(ctx, request, nil)
```

### Send Email

```go
request := &awscomm.EmailRequest{
    CallbackURL: "https://callback.example.com",
    Payload: awscomm.EmailPayload{
        To: []awscomm.EmailRecipient{
            {Email: "recipient@example.com", Type: "to"},
        },
        Subject: "Test Email",
        Text:    "Hello from SDK",
        HTML:    "<p>Hello from SDK</p>",
    },
}

response, err := client.SendEmail(ctx, request, nil)
```

### Send Fax

**⚠️ Memory Usage Warning**: Fax file methods (`SendFaxByFileName`, `SendFaxByContent`, `SendFaxByContentBytes`) load the entire file into memory (not chunked/streamed). Maximum file size is **20 MB**.

```go
// From URL
request := &awscomm.FaxRequest{
    CallbackURL: "https://callback.example.com",
    Payload: awscomm.FaxPayload{
        ToFaxNumber: "+1234567890",
        FileURL:     "https://example.com/document.pdf",
    },
}
response, err := client.SendFax(ctx, request, nil)

// From local file (loads entire file into memory)
response, err := client.SendFaxByFileName(ctx, "+1234567890", "https://callback.example.com", "document.pdf", nil)

// From HTML content (loads entire content into memory)
response, err := client.SendFaxByContent(ctx, "+1234567890", "https://callback.example.com", "<h1>Hello</h1>", "html", nil)

// From PDF bytes (loads entire content into memory)
pdfBytes, _ := os.ReadFile("document.pdf")
response, err := client.SendFaxByContentBytes(ctx, "+1234567890", "https://callback.example.com", pdfBytes, nil)
```

### Send Voice Mail

```go
request := &awscomm.VoiceMailRequest{
    CallbackURL: "https://callback.example.com",
    Payload: awscomm.VoiceMailPayload{
        ToPhoneNumber: "+1234567890",
        Message:       "This is a voice message",
    },
}

response, err := client.SendVoiceMail(ctx, request)
```

## Query Parameters

All send methods support optional query parameters:

```go
queryParams := map[string]string{
    "param1": "value1",
    "param2": "value2",
}

response, err := client.SendSMS(ctx, request, queryParams)
```

## Error Handling

The SDK provides structured error handling:

```go
response, err := client.SendSMS(ctx, request, nil)
if err != nil {
    // Check if it's a comm SDK error
    if awscomm.IsCommError(err) {
        // Handle SDK-specific error
    }
    log.Fatal(err)
}
```

## Webhook Handling

The SDK provides utilities for handling webhook callbacks with HMAC signature verification.

### Validate and Parse Webhook

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    // Read request body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Get signature from header
    signature := r.Header.Get("X-Webhook-Signature")
    secret := "your-webhook-secret"

    // Parse and validate webhook
    webhook, err := awscomm.ParseWebhookPayload(bodyBytes, signature, secret)
    if err != nil {
        http.Error(w, "Invalid webhook", http.StatusUnauthorized)
        return
    }

    // Process webhook based on type
    switch webhook.CommType {
    case "sms":
        // Handle SMS webhook
    case "email":
        // Handle email webhook
    case "fax":
        // Handle fax webhook
    case "voice_mail":
        // Handle voice mail webhook
    }

    w.WriteHeader(http.StatusOK)
}
```

### Webhook Payload Structure

```go
type WebhookPayload struct {
    Payload      interface{}
    Metadata     interface{}
    Type         string // "internal" or "external"
    CommType     string // "sms", "email", "fax", "voice_mail"
    Status       string // "QUEUED", "SENT", "FAILED"
    FailedReason string // Error message if status is FAILED
}
```

### Manual Signature Validation

```go
// Validate signature manually
valid := awscomm.ValidateWebhookSignature(payloadJSON, signature, secret)

// Or validate with error details
valid, err := awscomm.ValidateWebhookRequest(payloadBytes, signature, secret)
```
