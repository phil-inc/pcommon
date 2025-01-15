package bedrock_agent

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
)

func TestLiveAgent(t *testing.T) {
	cfg := Config{
		AgentID:      "SHXDYV9CYX",
		AgentAliasID: "TSTALIASID",
		PromptID:     "RDMRQ7CVT9",
		Region:       "us-east-1",
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	service := New().Config(cfg).AWSConfig(awsCfg).Build()

	cardHistory := `{"hello"}`

	resp, err := service.ProcessCardHistory(context.Background(), cardHistory)
	if err != nil {
		t.Fatalf("Failed to process card history: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Response == "" {
		t.Error("Response is empty")
	}

	t.Logf("Response: %s", resp.Response)
}
