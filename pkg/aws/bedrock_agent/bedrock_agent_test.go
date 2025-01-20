package bedrock_agent

import (
	"context"
	"testing"
)

func TestLiveAgent(t *testing.T) {
	cfg := Config{
		AgentID:      "SHXDYV9CYX",
		AgentAliasID: "TSTALIASID",
		PromptID:     "RDMRQ7CVT9",
		Region:       "us-east-1",
	}

	service, err := New().Config(&cfg).Build()
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

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
