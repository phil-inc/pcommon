// Package bedrockagent provides functionality to interact with AWS Bedrock Agent
package bedrock_agent

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	config             *Config
	awsConfig          *aws.Config
	agentClient        BedrockAgentClient
	agentRuntimeClient BedrockAgentRuntimeClient
}

type Builder struct {
	config             *Config
	awsConfig          *aws.Config
	agentClient        BedrockAgentClient
	agentRuntimeClient BedrockAgentRuntimeClient
}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) Build() (*Service, error) {
	if b.config == nil {
		return nil, fmt.Errorf("agent config is required")
	}
	if b.awsConfig == nil {
		awsConfig, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(b.config.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		b.awsConfig = &awsConfig
	}

	return &Service{
		config:             b.config,
		awsConfig:          b.awsConfig,
		agentClient:        bedrockagent.NewFromConfig(*b.awsConfig),
		agentRuntimeClient: bedrockagentruntime.NewFromConfig(*b.awsConfig),
	}, nil
}

func (b *Builder) Config(config *Config) *Builder {
	b.config = config
	return b
}

func (b *Builder) AWSConfig(awsConfig *aws.Config) *Builder {
	b.awsConfig = awsConfig
	return b
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
