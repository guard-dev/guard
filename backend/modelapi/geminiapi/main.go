package geminiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"guarddev/logger"
	"guarddev/types"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

type GeminiConnectProps struct {
	Logger *logger.LogMiddleware
}

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

type Gemini struct {
	logger *logger.LogMiddleware
	client *genai.Client
}

func exponentialBackoff(attempt int) time.Duration {
	tracer := otel.Tracer("geminiapi/exponentialBackoff")
	_, span := tracer.Start(context.Background(), "exponentialBackoff")
	defer span.End()

	span.SetAttributes(attribute.Int("attempt", attempt))
	return baseDelay * time.Duration(1<<uint(attempt))
}

func Connect(ctx context.Context, args GeminiConnectProps) *Gemini {
	tracer := otel.Tracer("geminiapi/Connect")
	ctx, span := tracer.Start(ctx, "Connect")
	defer span.End()

	maxWorkers := 200

	span.SetAttributes(attribute.Int("maxWorkers", maxWorkers))

	GEMINI_KEY := os.Getenv("GEMINI_SECRET_KEY")

	client, err := genai.NewClient(ctx, option.WithAPIKey(GEMINI_KEY))
	if err != nil {
		args.Logger.Logger(ctx).Error("[GeminiAPI] Could not create Gemini client")
		os.Exit(21)
	}

	return &Gemini{logger: args.Logger, client: client}
}

func (g *Gemini) SummarizeFindings(ctx context.Context, service string, region string, findings []string) (*types.SummarizeFindings, error) {
	tracer := otel.Tracer("geminiapi/SummarizeIssues")
	ctx, span := tracer.Start(ctx, "SummarizeIssues")
	defer span.End()

	log := g.logger.Logger(ctx)

	// First LLM call: Summarize findings
	summary, remedies, err := g.summarizeIssues(ctx, service, region, findings)
	if err != nil {
		log.Error("Error summarizing issues", zap.Error(err))
		return nil, err
	}

	// Second LLM call: Generate commands
	commands, err := g.generateCommands(ctx, service, region, summary, remedies)
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

func (g *Gemini) summarizeIssues(ctx context.Context, service, region string, findings []string) (string, string, error) {
	tracer := otel.Tracer("geminiapi/summarizeIssues")
	ctx, span := tracer.Start(ctx, "summarizeIssues")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", service),
		attribute.String("region", region),
		attribute.Int("findingsCount", len(findings)),
	)

	systemPrompt := fmt.Sprintf(`
		You are a cloud auditing expert. You will be analyzing some findings from AWS resources.
		Here are some findings from service %s from region %s. Some findings might be just noise,
		just simply ignore those.
	`, service, region)

	maxItems := 50
	if len(findings) < maxItems {
		maxItems = len(findings)
	}

	userPrompt := fmt.Sprintf(`
		Analyze and summarize these findings from %s in region %s:
		<Findings Start>
		%s
		<Findings End>

		Provide:
		1. A brief summary of the problems found.
		2. A general description of the remedies to address the issues.
	`, service, region, strings.Join(findings[:maxItems], "\n"))

	summarizeIssuesFunc := GetSummarizeIssuesFunction()

	model := g.client.GenerativeModel("gemini-1.5-pro-002")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}
	model.Tools = []*genai.Tool{summarizeIssuesFunc}
	model.ToolConfig = &genai.ToolConfig{
		FunctionCallingConfig: &genai.FunctionCallingConfig{
			Mode:                 genai.FunctionCallingAny,
			AllowedFunctionNames: []string{"summarize_issues"},
		},
	}

	resp, err := g.generateContentWithRetry(ctx, model, userPrompt)
	if err != nil {
		return "", "", err
	}

	// Parse the response
	functionCall, ok := resp.Candidates[0].Content.Parts[0].(genai.FunctionCall)
	if !ok {
		return "", "", fmt.Errorf("expected FunctionCall, got %T", resp.Candidates[0].Content.Parts[0])
	}

	data, err := json.Marshal(functionCall.Args)
	if err != nil {
		return "", "", err
	}

	var result struct {
		Summary  string `json:"summary"`
		Remedies string `json:"remedies"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", err
	}

	return result.Summary, result.Remedies, nil
}

func (g *Gemini) generateCommands(ctx context.Context, service, region, summary, remedies string) ([]string, error) {
	tracer := otel.Tracer("geminiapi/generateCommands")
	ctx, span := tracer.Start(ctx, "generateCommands")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", service),
		attribute.String("region", region),
	)

	systemPrompt := fmt.Sprintf(`
		You are a cloud automation expert. You will be generating AWS CLI commands to address issues in AWS resources.
		The commands will be for service %s in region %s.
	`, service, region)

	userPrompt := fmt.Sprintf(`
		Based on the following summary and remedies, generate AWS CLI commands to address the issues:

		Summary:
		%s

		Remedies:
		%s

		Provide a list of AWS CLI command(s) which can be used to remediate the issues programmatically.
		Make sure that the commands are effective and do the job.
	`, summary, remedies)

	generateCommandsFunc := GetGenerateCommandsFunction()

	model := g.client.GenerativeModel("gemini-1.5-flash-002")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}
	model.Tools = []*genai.Tool{generateCommandsFunc}
	model.ToolConfig = &genai.ToolConfig{
		FunctionCallingConfig: &genai.FunctionCallingConfig{
			Mode:                 genai.FunctionCallingAny,
			AllowedFunctionNames: []string{"generate_commands"},
		},
	}

	resp, err := g.generateContentWithRetry(ctx, model, userPrompt)
	if err != nil {
		return nil, err
	}

	// Parse the response
	functionCall, ok := resp.Candidates[0].Content.Parts[0].(genai.FunctionCall)
	if !ok {
		return nil, fmt.Errorf("expected FunctionCall, got %T", resp.Candidates[0].Content.Parts[0])
	}

	data, err := json.Marshal(functionCall.Args)
	if err != nil {
		return nil, err
	}

	var result struct {
		Commands []string `json:"commands"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Commands, nil
}

func (g *Gemini) generateContentWithRetry(ctx context.Context, model *genai.GenerativeModel, prompt string) (*genai.GenerateContentResponse, error) {
	tracer := otel.Tracer("geminiapi/generateContentWithRetry")
	ctx, span := tracer.Start(ctx, "generateContentWithRetry")
	defer span.End()

	var resp *genai.GenerateContentResponse
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		span.AddEvent("Attempt", trace.WithAttributes(attribute.Int("attemptNumber", attempt+1)))

		resp, err = model.GenerateContent(ctx, genai.Text(prompt))

		if err != nil || resp == nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			if err != nil {
				g.logger.Logger(ctx).Warn("Error generating LLM content, retrying...",
					zap.Error(err),
					zap.Int("attempt", attempt+1),
					zap.Int("maxRetries", maxRetries))
				span.RecordError(err)
			} else {
				g.logger.Logger(ctx).Warn("Received empty or invalid response, retrying...",
					zap.Int("attempt", attempt+1),
					zap.Int("maxRetries", maxRetries))
				span.AddEvent("EmptyResponse")
			}

			if attempt < maxRetries-1 {
				delay := exponentialBackoff(attempt)
				span.AddEvent("Backoff", trace.WithAttributes(attribute.Int64("delayMs", delay.Milliseconds())))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
			}
			continue
		}

		// If we get here, we have a valid response
		break
	}

	// Final error check after all retries
	if err != nil {
		g.logger.Logger(ctx).Error("Final error generating LLM content after retries:", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func GetSummarizeIssuesFunction() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "summarize_issues",
			Description: "Summarize a list of findings and provide general remedies.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"summary": {
						Type:        genai.TypeString,
						Description: "Overall summary of the findings from the resources.",
					},
					"remedies": {
						Type:        genai.TypeString,
						Description: "General description of the steps needed to fix the issues.",
					},
				},
				Required: []string{"summary", "remedies"},
			},
		}},
	}
}

func GetGenerateCommandsFunction() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "generate_commands",
			Description: "Generate AWS CLI commands for remediation based on the summary and remedies.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"commands": {
						Type: genai.TypeArray,
						Description: `
							A list of AWS CLI command(s) which can be used to remediate the issues programmatically.
							If there is only one command, do not split it across multiple strings, in that case just simply return a list with one string.
							Make sure that the commands are effective and do the job.
						`,
						Items: &genai.Schema{
							Type: genai.TypeString,
						},
					},
				},
				Required: []string{"commands"},
			},
		}},
	}
}
