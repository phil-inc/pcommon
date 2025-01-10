// Package bedrockagent provides functionality to interact with AWS Bedrock Agent
package bedrock_agent

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	runtimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

// Config holds the configuration for the BedrockAgent client
type Config struct {
	AgentID      string
	AgentAliasID string
	PromptID     string
	Region       string
}

// Response represents the structured response from the agent
type Response struct {
	Response string `json:"response"`
}

// BedrockAgentClient interface defines the methods for interacting with Bedrock Agent
type BedrockAgentClient interface {
	GetPrompt(ctx context.Context, params *bedrockagent.GetPromptInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetPromptOutput, error)
}

// BedrockAgentRuntimeClient interface defines the methods for interacting with Bedrock Agent Runtime
type BedrockAgentRuntimeClient interface {
	InvokeAgent(ctx context.Context, params *bedrockagentruntime.InvokeAgentInput, optFns ...func(*bedrockagentruntime.Options)) (*bedrockagentruntime.InvokeAgentOutput, error)
}

// Service represents the BedrockAgent service
type Service struct {
	config             Config
	agentClient        BedrockAgentClient
	agentRuntimeClient BedrockAgentRuntimeClient
}

// NewService creates a new instance of the BedrockAgent service
func NewService(cfg Config) (*Service, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &Service{
		config:             cfg,
		agentClient:        bedrockagent.NewFromConfig(awsCfg),
		agentRuntimeClient: bedrockagentruntime.NewFromConfig(awsCfg),
	}, nil
}

// NewServiceWithClients creates a new service instance with custom clients (useful for testing)
func NewServiceWithClients(cfg Config, agentClient BedrockAgentClient, runtimeClient BedrockAgentRuntimeClient) *Service {
	return &Service{
		config:             cfg,
		agentClient:        agentClient,
		agentRuntimeClient: runtimeClient,
	}
}

// ProcessCardHistory processes the card history and returns a response
func (s *Service) ProcessCardHistory(ctx context.Context, cardHistory string) (*Response, error) {
	// Get the managed prompt
	promptResp, err := s.agentClient.GetPrompt(ctx, &bedrockagent.GetPromptInput{
		PromptIdentifier: &s.config.PromptID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	if len(promptResp.Variants) == 0 {
		return nil, fmt.Errorf("no prompt variants found")
	}

	promptTemplate := promptResp.Variants[0].TemplateConfiguration

	var promptText string
	switch p := promptTemplate.(type) {
	case *types.PromptTemplateConfigurationMemberText:
		if p.Value.Text == nil {
			return nil, fmt.Errorf("prompt text is nil")
		}
		promptText = *p.Value.Text
	default:
		return nil, fmt.Errorf("unsupported prompt template type")
	}

	// Format the prompt
	formattedPrompt := replaceCardHistory(promptText, cardHistory)

	// Generate session ID
	sessionID := generateSessionID(cardHistory)

	// Query knowledge base
	agentResp, err := s.agentRuntimeClient.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		AgentId:      &s.config.AgentID,
		AgentAliasId: &s.config.AgentAliasID,
		InputText:    &formattedPrompt,
		SessionId:    &sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke agent: %w", err)
	}

	stream := agentResp.GetStream()
	defer stream.Close()

	var responseBuilder strings.Builder

	for event := range stream.Events() {
		if chunk, ok := event.(*runtimeTypes.ResponseStreamMemberChunk); ok {
			responseBuilder.Write(chunk.Value.Bytes)
		}
	}

	return &Response{
		Response: responseBuilder.String(),
	}, nil
}

// replaceCardHistory replaces the cardhistory placeholder in the prompt
func replaceCardHistory(promptText, cardHistory string) string {
	return strings.Replace(promptText, "{{cardhistory}}", cardHistory, -1)
}

// generateSessionID creates a session ID based on card history
func generateSessionID(cardHistory string) string {
	h := fnv.New32a()
	h.Write([]byte(cardHistory))
	return fmt.Sprintf("session-%d", h.Sum32())
}
