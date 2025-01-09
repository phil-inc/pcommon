# ai

The `ai` package is a library for useful ai-related functions for PHIL applications. 

## Example Usage

```
import (
    "github.com/phil-inc/pcommon/ai"
)

knowledgeBaseID := "KnowledgeBaseID from AWS console"
inputText := "Your input text here"
sessionID := "session-12345"

response, err := ai.InvokeAWSBedrockKnowledgeBase(knowledgeBaseID, inputText, sessionID)
if err != nil {
    fmt.Printf("Error: %s\n", err)
}

fmt.Printf("Response: %s\n", response)
```