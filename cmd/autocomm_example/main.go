package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/phil-inc/pcommon/pkg/awscomm"
)

var commServiceName = os.Getenv("COMM_SERVICE_NAME")
var commServiceApiKey = os.Getenv("COMM_SERVICE_API_KEY")
var baseURL = os.Getenv("COMM_BASE_URL")
var toPhoneNumber = ""
var email = ""
var message = "This is test message from sdk example"
var faxHtml = `<!doctype html><html lang="en"><head><title>Example Domain</title><meta name="viewport" content="width=device-width, initial-scale=1"><style>body{background:#eee;width:60vw;margin:15vh auto;font-family:system-ui,sans-serif}h1{font-size:1.5em}div{opacity:0.8}a:link,a:visited{color:#348}</style><body><div><h1>Example Domain</h1><p>This domain is for use in documentation examples without needing permission. Avoid use in operations.<p><a href="https://iana.org/domains/example">Learn more</a></div></body></html>
`

func main() {
	// Validate required environment variables
	if baseURL == "" {
		log.Fatal("COMM_BASE_URL environment variable is required. Example: export COMM_BASE_URL=https://api.example.com")
	}
	if commServiceName == "" || commServiceApiKey == "" {
		log.Fatal("COMM_SERVICE_NAME and COMM_SERVICE_API_KEY environment variable is required. Example: export COMM_SERVICE_NAME=your-api-key")
	}

	ctx := context.Background()

	ac := awscomm.NewClient(baseURL, commServiceName, commServiceApiKey)

	testSms(ctx, ac)
	testEmail(ctx, ac)
	testFax(ctx, ac)
	testFaxV2(ctx, ac)
	testVoiceMail(ctx, ac)
}

func testSms(ctx context.Context, ac *awscomm.Client) {
	smsResp, err := ac.SendSMS(ctx, &awscomm.SMSRequest{
		CallbackURL: "",
		Payload: awscomm.SMSPayload{
			ToPhoneNumber: toPhoneNumber,
			Message:       message,
		},
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("ID: %s , STATUS: %s , type: %s\n", smsResp.CommRequestID, smsResp.Status, smsResp.Type)
}

func testVoiceMail(ctx context.Context, ac *awscomm.Client) {
	smsResp, err := ac.SendVoiceMail(ctx, &awscomm.VoiceMailRequest{
		CallbackURL: "",
		Payload: awscomm.VoiceMailPayload{
			ToPhoneNumber: toPhoneNumber,
			Message:       message,
		},
		Metadata: map[string]any{
			"orderNumber": "1234-1234-1234",
		},
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("ID: %s , STATUS: %s , type: %s\n", smsResp.CommRequestID, smsResp.Status, smsResp.Type)
}

func testEmail(ctx context.Context, ac *awscomm.Client) {
	smsResp, err := ac.SendEmail(ctx, &awscomm.EmailRequest{
		CallbackURL: "",
		Payload: awscomm.EmailPayload{
			To: []awscomm.EmailRecipient{
				{
					Email: email,
					Type:  "to",
				},
			},
			Subject: "Test SDK subject",
			Text:    message,
		},
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("ID: %s , STATUS: %s , type: %s\n", smsResp.CommRequestID, smsResp.Status, smsResp.Type)
}

func testFax(ctx context.Context, ac *awscomm.Client) {
	smsResp, err := ac.SendFax(ctx, &awscomm.FaxRequest{
		Payload: awscomm.FaxPayload{
			ToFaxNumber:    toPhoneNumber,
			StringData:     faxHtml,
			StringDataType: "html",
		},
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("ID: %s , STATUS: %s , type: %s\n", smsResp.CommRequestID, smsResp.Status, smsResp.Type)
}

func testFaxV2(ctx context.Context, ac *awscomm.Client) {
	smsResp, err := ac.SendFaxByContentBytes(ctx, toPhoneNumber, "", stringToPDFBytes(message))
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("ID: %s , STATUS: %s , type: %s\n", smsResp.CommRequestID, smsResp.Status, smsResp.Type)
}

// Test utils function to generate pdf from message
func stringToPDFBytes(text string) []byte {
	// escape PDF special chars
	escaped := make([]rune, 0, len(text))
	for _, r := range text {
		if r == '\\' || r == '(' || r == ')' {
			escaped = append(escaped, '\\')
		}
		escaped = append(escaped, r)
	}
	content := string(escaped)

	pdf := fmt.Sprintf(`%%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]
   /Contents 4 0 R
   /Resources << /Font << /F1 5 0 R >> >> >>
endobj
4 0 obj
<< /Length %d >>
stream
BT
/F1 12 Tf
72 720 Td
(%s) Tj
ET
endstream
endobj
5 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
xref
0 6
0000000000 65535 f
trailer
<< /Size 6 /Root 1 0 R >>
startxref
0
%%%%EOF
`, len(content)+50, content)

	return []byte(pdf)
}
