package ai

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

// InvokeAWSBedrockAgent invokes the AWS Bedrock agent with the provided parameters.
func InvokeAWSBedrockAgent(cfg aws.Config, ctx context.Context, agentID, inputText, sessionID string) (string, error) {
	bedrockAgentRuntime := bedrockagentruntime.NewFromConfig(cfg)

	// Invoke the Bedrock agent
	response, err := bedrockAgentRuntime.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		// TODO: Allow parameters to be passed in herre
		// AgentId:         aws.String(agentID),
		// InputText:       aws.String(inputText),
		// SessionId:       aws.String(sessionID),
	})
	if err != nil {
		return "", fmt.Errorf("error invoking knowledge base: %w", err)
	}

	// Process the response
	completion := ""
	for event := range response.GetStream().Events() {
		chunk := (string)(event.(*types.ResponseStreamMemberChunk).Value.Bytes)
		completion += chunk
	}

	return completion, nil
}
