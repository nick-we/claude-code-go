package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

func main() {
	var prompt string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.Parse()

	if prompt == "" {
		panic("Prompt must not be empty")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))
	resp, err := client.Chat.Completions.New(context.Background(),
		openai.ChatCompletionNewParams{
			Model: "anthropic/claude-haiku-4.5",
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(prompt),
						},
					},
				},
			},
			Tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "Read",
					Description: param.Opt[string]{Value: "Read and return the contents of a file"},
					Parameters: shared.FunctionParameters{
						"type": "object",
						"properties": map[string]shared.FunctionParameters{
							"file_path": {
								"type":        "string",
								"description": "The path to the file to read",
							},
						},
						"required": []string{
							"file_path",
						},
					},
				}),
			},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(resp.Choices) == 0 {
		panic("No choices in response")
	}

	if len(resp.Choices[0].Message.ToolCalls) == 0 {
		fmt.Print(resp.Choices[0].Message.Content)
	} else {
		functionName := resp.Choices[0].Message.ToolCalls[0].Function.Name
		switch functionName {
		case "Read":
			fmt.Print(resp.Choices[0].Message.Content)
			argJsonString := resp.Choices[0].Message.ToolCalls[0].Function.Arguments
			if len(argJsonString) == 0 {
				fmt.Fprintln(os.Stderr, "No arguments in function call")
				os.Exit(1)
			}

			args := make(map[string]any, 0)
			if err := json.Unmarshal([]byte(argJsonString), &args); err != nil {
				fmt.Fprintln(os.Stderr, "Invalid arguments in function call")
				os.Exit(1)
			}

			filePath := args["file_path"].(string)
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading file")
				os.Exit(1)
			}
			fmt.Print(string(fileContent))
		}
	}

	fmt.Print(resp.Choices[0].Message.Content)
}
