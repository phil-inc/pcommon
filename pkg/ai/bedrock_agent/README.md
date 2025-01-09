# Bedrock Agent

This wraps around the AWS Bedrock Agent SDK

Assumptions:

- Bedrock Knowledge Base exists in AWS
- Bedrock Managed Prompt exists in AWS
- Bedrock Agent already exists in AWS

## Usage

See [./bedrock_agent_test.go][].

Requires AWS credentials for PHIL staging.

Inputs are pre-configured for PhilRX SOP knowledge base

```bash
mitchell@mitchell-MS-7A59:~/src/github.com/philinc/pcommon/pkg/ai/bedrock_agent$ go test . -v
=== RUN   TestLiveAgent
    bedrock_agent_test.go:36: Response: {
            "overallSopAdheranceScore": 0,
            "specificStepsNotFollowed": ["No steps identified - insufficient event log data"],
            "patientExperienceIssues": ["Unable to assess patient experience"],
            "hcpExperienceIssues": ["Unable to assess healthcare provider experience"],
            "mostImpactfulDiscrepancy": "Cannot determine without valid event log",
            "keyObservations": [
                "Provided event log is invalid",
                "No meaningful processing information available",
                "Unable to perform comprehensive SOP analysis"
            ],
            "sopAdheranceScoreReason": "Zero score due to lack of processable event log data"
        }
--- PASS: TestLiveAgent (14.42s)
PASS
ok  	github.com/phil-inc/pcommon/pkg/ai/bedrock_agent	14.429s

```
