package openrouterapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"guarddev/httpmiddleware"
	"guarddev/logger"
	"guarddev/types"
	"guarddev/utils"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type PropertyType string

const (
	PropertyTypeString  PropertyType = "string"
	PropertyTypeNumber  PropertyType = "number"
	PropertyTypeBoolean PropertyType = "boolean"
	PropertyTypeObject  PropertyType = "object"
	PropertyTypeArray   PropertyType = "array"
)

type Property struct {
	Type        PropertyType        `json:"type"`
	Description string              `json:"description"`
	Enum        []string            `json:"enum,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
}

type Parameters struct {
	Type       PropertyType        `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type ToolWrapper struct {
	Type     string `json:"type"`
	Function Tool   `json:"function"`
}

type ToolChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

type MessageContent struct {
	Type   string    `json:"type"`
	Text   *string   `json:"text,omitempty"`
	Source *ImageUrl `json:"source,omitempty"`
}

type ImageUrl struct {
	Type      string `json:"type"`
	Data      string `json:"data"`
	MediaType string `json:"media_type"`
}

type ChatCompletionInputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatRequestInput struct {
	Model      string                       `json:"model"`
	Messages   []ChatCompletionInputMessage `json:"messages"`
	MaxTokens  int                          `json:"max_tokens"`
	System     *string                      `json:"system,omitempty"`
	Tools      *[]ToolWrapper               `json:"tools,omitempty"`
	ToolChoice *ToolChoice                  `json:"tool_choice,omitempty"`
}

type OpenRouterResponse struct {
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Message struct {
	Role      string      `json:"role"`
	Content   string      `json:"content"`
	ToolCalls []ToolCall  `json:"tool_calls"`
	Refusal   interface{} `json:"refusal"` // Can be changed to a specific type if known
}

type ToolCall struct {
	ID       string   `json:"id"`
	Function Function `json:"function"`
	Type     string   `json:"type"`
}

type Function struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type OpenRouterConnectProps struct {
	Logger *logger.LogMiddleware
}

type OpenRouter struct {
	logger    *logger.LogMiddleware
	semaphore *semaphore.Weighted
}

func Connect(ctx context.Context, args OpenRouterConnectProps) *OpenRouter {
	tracer := otel.Tracer("openrouterapi/Connect")
	ctx, span := tracer.Start(ctx, "Connect")
	defer span.End()

	maxWorkers := 10
	sem := semaphore.NewWeighted(int64(maxWorkers))

	span.SetAttributes(attribute.Int("maxWorkers", maxWorkers))

	return &OpenRouter{logger: args.Logger, semaphore: sem}
}

type MakeAPIRequestProps struct {
	Retries      int
	RequestInput ChatRequestInput
}

func (o *OpenRouter) MakeAPIRequest(ctx context.Context, args MakeAPIRequestProps) (*OpenRouterResponse, error) {
	tracer := otel.Tracer("openrouterapi/MakeAPIRequest")
	ctx, span := tracer.Start(ctx, "MakeAPIRequest")
	defer span.End()

	API_KEY := os.Getenv("OPEN_ROUTER_SECRET_KEY")
	URL := "https://openrouter.ai/api/v1/chat/completions"

	span.SetAttributes(
		attribute.String("api.url", URL),
		attribute.Int("request.max_tokens", args.RequestInput.MaxTokens),
		attribute.String("request.model", args.RequestInput.Model),
	)

	chatGptInput := args.RequestInput
	retries := args.Retries
	originalRetries := args.Retries

	jsonData, err := json.Marshal(chatGptInput)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("Could not generate request body: " + err.Error())
	}

	span.SetAttributes(attribute.Int("retries", retries))

	for retries > 0 {
		sleepTime := utils.GetExponentialDelaySeconds(originalRetries - retries)
		span.SetAttributes(attribute.Int("sleep_time", sleepTime))

		if err := o.semaphore.Acquire(ctx, 1); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("Failed to acquire semaphore.")
		}
		defer o.semaphore.Release(1)

		respBody, err := httpmiddleware.HttpRequest(httpmiddleware.HttpRequestStruct{
			Method: "POST",
			Url:    URL,
			Body:   bytes.NewBuffer(jsonData),
			Headers: map[string]string{
				"authorization": "Bearer " + API_KEY,
				"content-type":  "application/json",
			},
		})

		if err != nil {
			span.RecordError(err)
			o.logger.Logger(ctx).Error(
				"[OpenRouter-API] Could not make request to OpenRouter. Retrying after sleeping.",
				zap.Error(err),
				zap.Int("retries_left", retries),
				zap.Int("sleep_time", sleepTime),
				zap.Any("input", chatGptInput),
			)
			retries -= 1
			time.Sleep(time.Duration(sleepTime) * time.Second)
		} else {
			var messageResponse OpenRouterResponse
			err = json.Unmarshal(respBody, &messageResponse)
			if err != nil || len(messageResponse.Choices) == 0 {
				span.RecordError(err)
				retries -= 1
				o.logger.Logger(ctx).Error(
					"[OpenRouter-API] Could not parse OpenRouter Request. Retrying after sleeping.",
					zap.Int("retries_left", retries),
					zap.Int("sleep_time", sleepTime),
					zap.Any("input", chatGptInput),
					zap.Error(err),
					zap.String("response_body", string(respBody)),
					zap.Int("content_length", len(messageResponse.Choices)),
				)
				time.Sleep(time.Duration(sleepTime) * time.Second)
			} else {
				span.AddEvent("Request successful")
				return &messageResponse, nil
			}
		}
	}

	span.AddEvent("All retries exhausted")
	return nil, fmt.Errorf("OpenRouter Requests Failed")
}

func (a *OpenRouter) generateCommands(ctx context.Context, service, region, accountId, summary, remedies string) ([]string, error) {
	tracer := otel.Tracer("openrouterapi/generateCommands")
	ctx, span := tracer.Start(ctx, "generateCommands")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", service),
		attribute.String("region", region),
		attribute.String("accountId", accountId),
	)

	systemPrompt := fmt.Sprintf(`
		You are a cloud automation expert. You will be generating AWS CLI commands to address issues in AWS resources.
		The commands will be for service %s in region %s for AWS account %s.
	`, service, region, accountId)

	userPrompt := fmt.Sprintf(`
		Based on the following summary and remedies, generate AWS CLI commands to address the issues:

		Summary:
		%s

		Remedies:
		%s

		Provide a list of AWS CLI command(s) which can be used to remediate the issues programmatically.
		Make sure that the commands are effective and do the job. The commands will be executed in AWS account %s.
	`, summary, remedies, accountId)

	generateCommandsFunc := GetGenerateCommandsFunction()

	requestInput := MakeAPIRequestProps{
		Retries: 3,
		RequestInput: ChatRequestInput{
			Model:     "meta-llama/llama-3.3-70b-instruct",
			MaxTokens: 2048,
			Messages: []ChatCompletionInputMessage{
				{
					Role:    "system",
					Content: systemPrompt,
				},
				{
					Role:    "user",
					Content: userPrompt,
				},
			},
			Tools: &[]ToolWrapper{*generateCommandsFunc},
			ToolChoice: &ToolChoice{Type: "function", Function: struct {
				Name string "json:\"name\""
			}{Name: "generate_commands"}},
		},
	}

	resp, err := a.MakeAPIRequest(ctx, requestInput)
	if err != nil {
		return nil, err
	}

	// Parse the tool call response
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("no tool call response received")
	}

	// Parse the `Arguments` field as a stringified JSON object
	data := resp.Choices[0].Message.ToolCalls[0].Function.Arguments

	var jsonString string
	if err := json.Unmarshal(data, &jsonString); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stringified JSON: %w", err)
	}

	// Parse the actual JSON string into the result struct
	var result struct {
		Commands []string `json:"commands"`
	}
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool call response: %v", err)
	}

	return result.Commands, nil
}

func (a *OpenRouter) SummarizeFindings(ctx context.Context, service string, region string, accountId string, findings []string) (*types.SummarizeFindings, error) {
	tracer := otel.Tracer("openrouterapi/SummarizeIssues")
	ctx, span := tracer.Start(ctx, "SummarizeIssues")
	defer span.End()

	log := a.logger.Logger(ctx)

	// First LLM call: Summarize findings
	summary, remedies, err := a.summarizeIssues(ctx, service, region, accountId, findings)
	if err != nil {
		log.Error("Error summarizing issues", zap.Error(err))
		return nil, err
	}

	// Second LLM call: Generate commands
	commands, err := a.generateCommands(ctx, service, region, accountId, summary, remedies)
	if err != nil {
		log.Error("Error generating commands", zap.Error(err))
		return nil, err
	}

	// Combine results
	result := &types.SummarizeFindings{
		Summary:  summary,
		Remedies: remedies,
		Commands: commands,
	}

	return result, nil
}

func (a *OpenRouter) summarizeIssues(ctx context.Context, service, region, accountId string, findings []string) (string, string, error) {
	tracer := otel.Tracer("openrouterapi/summarizeIssues")
	ctx, span := tracer.Start(ctx, "summarizeIssues")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", service),
		attribute.String("region", region),
		attribute.String("accountId", accountId),
		attribute.Int("findingsCount", len(findings)),
	)

	systemPrompt := fmt.Sprintf(`
		You are a cloud auditing expert. You will be analyzing some findings from AWS resources.
		Here are some findings from service %s from region %s in AWS account %s. Some findings might be just noise,
		just simply ignore those.
	`, service, region, accountId)

	maxItems := 50
	if len(findings) < maxItems {
		maxItems = len(findings)
	}

	userPrompt := fmt.Sprintf(`
		Analyze and summarize these findings from %s in region %s for AWS account %s:
		<Findings Start>
		%s
		<Findings End>

		Provide:
		1. A brief summary of the problems found.
		2. A general description of the remedies to address the issues.
	`, service, region, accountId, strings.Join(findings[:maxItems], "\n"))

	summarizeIssuesFunc := GetSummarizeIssuesFunction()

	requestInput := MakeAPIRequestProps{
		Retries: 3,
		RequestInput: ChatRequestInput{
			Model:     "meta-llama/llama-3.3-70b-instruct",
			MaxTokens: 2048,
			Messages: []ChatCompletionInputMessage{
				{
					Role:    "system",
					Content: systemPrompt,
				},
				{
					Role:    "user",
					Content: userPrompt,
				},
			},
			Tools: &[]ToolWrapper{*summarizeIssuesFunc},
			ToolChoice: &ToolChoice{Type: "function", Function: struct {
				Name string "json:\"name\""
			}{Name: "summarize_issues"}},
		},
	}

	resp, err := a.MakeAPIRequest(ctx, requestInput)
	if err != nil {
		return "", "", err
	}

	// Parse the tool call response
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return "", "", fmt.Errorf("no tool call response received")
	}

	// `data` is a stringified JSON object
	data := resp.Choices[0].Message.ToolCalls[0].Function.Arguments

	// First, unmarshal `data` as a string
	var jsonString string
	if err := json.Unmarshal(data, &jsonString); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal stringified JSON: %w", err)
	}

	// Second, unmarshal the resulting JSON string into the struct
	var result struct {
		Summary  string `json:"summary"`
		Remedies string `json:"remedies"`
	}
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return "", "", fmt.Errorf("failed to parse tool call response: %v", err)
	}

	return result.Summary, result.Remedies, nil
}

func GetSummarizeIssuesFunction() *ToolWrapper {
	return &ToolWrapper{
		Type: "function",
		Function: Tool{
			Name:        "summarize_issues",
			Description: "Summarize AWS resource findings and provide remediation suggestions",
			Parameters: Parameters{
				Type: PropertyTypeObject,
				Properties: map[string]Property{
					"summary": {
						Type:        PropertyTypeString,
						Description: "A concise summary of the AWS resource findings and issues identified",
					},
					"remedies": {
						Type:        PropertyTypeString,
						Description: "Detailed remediation steps and best practices to address the identified issues",
					},
				},
				Required: []string{"summary", "remedies"},
			},
		},
	}
}

func GetGenerateCommandsFunction() *ToolWrapper {
	return &ToolWrapper{
		Type: "function",
		Function: Tool{
			Name:        "generate_commands",
			Description: "Generate AWS CLI commands to remediate identified issues",
			Parameters: Parameters{
				Type: PropertyTypeObject,
				Properties: map[string]Property{
					"commands": {
						Type:        PropertyTypeArray,
						Description: "List of AWS CLI commands to remediate the issues",
						Items: &Property{
							Type: PropertyTypeString,
						},
					},
				},
				Required: []string{"commands"},
			},
		},
	}
}
