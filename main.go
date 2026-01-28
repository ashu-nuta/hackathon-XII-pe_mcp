package main

import (
	"fmt"
	"os"

	"github.com/thunderboltsid/mcp-nutanix/internal/client"
	"github.com/thunderboltsid/mcp-nutanix/pkg/prompts"
	"github.com/thunderboltsid/mcp-nutanix/pkg/resources"
	"github.com/thunderboltsid/mcp-nutanix/pkg/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolRegistration holds a tool function and its handler
type ToolRegistration struct {
	Func    func() mcp.Tool
	Handler server.ToolHandlerFunc
}

// ResourceRegistration represents a resource and its associated tools
type ResourceRegistration struct {
	Tools           []ToolRegistration
	ResourceFunc    func() mcp.ResourceTemplate
	ResourceHandler server.ResourceTemplateHandlerFunc
}

// initializeFromEnvIfAvailable initializes the Prism client only if environment variables are available
func initializeFromEnvIfAvailable() {
	endpoint := os.Getenv("NUTANIX_ENDPOINT")
	username := os.Getenv("NUTANIX_USERNAME")
	password := os.Getenv("NUTANIX_PASSWORD")

	// Only initialize if all required environment variables are set
	// This allows prompt-based initialization to work when env vars are not present
	if endpoint != "" && username != "" && password != "" {
		client.Init(client.PrismClientProvider)
		fmt.Printf("Initialized Prism client from environment variables for endpoint: %s\n", endpoint)
	}
}

func main() {
	// Initialize the Prism client only if environment variables are available
	initializeFromEnvIfAvailable()

	// Define server hooks for logging and debugging
	hooks := &server.Hooks{}
	hooks.AddOnError(func(id any, method mcp.MCPMethod, message any, err error) {
		fmt.Printf("onError: %s, %v, %v, %v\n", method, id, message, err)
	})

	// Log level based on environment variable
	debugMode := os.Getenv("DEBUG") != ""
	if debugMode {
		hooks.AddBeforeAny(func(id any, method mcp.MCPMethod, message any) {
			fmt.Printf("beforeAny: %s, %v, %v\n", method, id, message)
		})
		hooks.AddOnSuccess(func(id any, method mcp.MCPMethod, message any, result any) {
			fmt.Printf("onSuccess: %s, %v, %v, %v\n", method, id, message, result)
		})
		hooks.AddBeforeInitialize(func(id any, message *mcp.InitializeRequest) {
			fmt.Printf("beforeInitialize: %v, %v\n", id, message)
		})
		hooks.AddAfterInitialize(func(id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
			fmt.Printf("afterInitialize: %v, %v, %v\n", id, message, result)
		})
		hooks.AddAfterCallTool(func(id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
			fmt.Printf("afterCallTool: %v, %v, %v\n", id, message, result)
		})
		hooks.AddBeforeCallTool(func(id any, message *mcp.CallToolRequest) {
			fmt.Printf("beforeCallTool: %v, %v\n", id, message)
		})
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Prism Central",
		"0.0.1",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	// Add the prompts
	s.AddPrompt(prompts.SetCredentials(), prompts.SetCredentialsResponse())

	// Add standalone tools
	s.AddTool(tools.ApiNamespacesList(), tools.ApiNamespacesListHandler())
	s.AddTool(tools.CriticalLogs(), tools.CriticalLogsHandler())
	s.AddTool(tools.CrashLogsCritical(), tools.CrashLogsCriticalHandler())
	s.AddTool(tools.FetchService(), tools.FetchServiceHandler())
	s.AddTool(tools.KernelLogsCritical(), tools.KernelLogsCriticalHandler())
	s.AddTool(tools.SSHExec(), tools.SSHExecHandler())
	s.AddTool(tools.SSHExecBatch(), tools.SSHExecBatchHandler())

	// Define all resources and tools
	resourceRegistrations := map[string]ResourceRegistration{
		"vm": {
			Tools: []ToolRegistration{
				{
					Func:    tools.VMList,
					Handler: tools.VMListHandler(),
				},
				{
					Func:    tools.VMCount,
					Handler: tools.VMCountHandler(),
				},
			},
			ResourceFunc:    resources.VM,
			ResourceHandler: resources.VMHandler(),
		},
	}

	// Register all tools and resources
	for name, registration := range resourceRegistrations {
		// Add all tools
		for _, tool := range registration.Tools {
			s.AddTool(tool.Func(), tool.Handler)
			if debugMode {
				fmt.Printf("Registered %s resource and tool\n", name)
			}
		}

		// Add the resource
		s.AddResourceTemplate(registration.ResourceFunc(), registration.ResourceHandler)
	}

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
