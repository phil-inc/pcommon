package ai

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

type Bedrock struct {
	cfg     aws.Config
	agentID string
	client  *bedrockagentruntime.Client
}

func NewBedrock(cfg aws.Config, agentID string) *Bedrock {
	return &Bedrock{
		cfg:     cfg,
		agentID: agentID,
		client:  bedrockagentruntime.NewFromConfig(cfg),
	}
}

func (b *Bedrock) Config() aws.Config {
	return b.cfg
}

func (b *Bedrock) AgentID() string {
	return b.agentID
}

// InvokeAWSBedrockAgent invokes the AWS Bedrock agent with the provided parameters.
func (b *Bedrock) InvokeAWSBedrockAgent(ctx context.Context, inputText, sessionID string) (string, error) {
	// Invoke the Bedrock agent
	response, err := b.client.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		// TODO: Allow parameters to be passed in herre
		// AgentId:         aws.String(agentID),
		// InputText:       aws.String(inputText),
		// SessionId:       aws.String(sessionID),
	})
	if err != nil {
		return "", fmt.Errorf("error invoking agent: %w", err)
	}

	// Process the response
	completion := ""
	for event := range response.GetStream().Events() {
		chunk := (string)(event.(*types.ResponseStreamMemberChunk).Value.Bytes)
		completion += chunk
	}

	return completion, nil
}
