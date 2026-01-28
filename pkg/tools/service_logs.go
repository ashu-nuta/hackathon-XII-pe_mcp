package tools

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/crypto/ssh"
)

const (
	envServiceHost     = "SSH_HOST"
	envServiceUsername = "SSH_USERNAME"
	envServicePassword = "SSH_PASSWORD"
	envServicePort     = "SSH_PORT"
	envSSHLogRoot      = "SSH_LOG_ROOT"
)

// FetchService defines the fetch_service tool
func FetchService() mcp.Tool {
	return mcp.NewTool("fetch_service",
		mcp.WithDescription("Fetch a file from /home/nutanix/data/logs via SSH and save locally"),
		mcp.WithString("path",
			mcp.Description("Optional file path under /home/nutanix/data/logs (defaults to narsil.out)"),
		),
	)
}

// FetchServiceHandler implements the handler for the fetch_service tool
func FetchServiceHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx

		host := getEnvTrimmedService(envServiceHost)
		username := getEnvTrimmedService(envServiceUsername)
		password := getEnvTrimmedService(envServicePassword)
		root := getEnvTrimmedService(envSSHLogRoot)
		if host == "" {
			return nil, fmt.Errorf("%s is required", envServiceHost)
		}
		if username == "" {
			return nil, fmt.Errorf("%s is required", envServiceUsername)
		}
		if password == "" {
			return nil, fmt.Errorf("%s is required", envServicePassword)
		}
		if root == "" {
			return nil, fmt.Errorf("%s is required", envSSHLogRoot)
		}

		port, err := getEnvPortService(envServicePort, 22)
		if err != nil {
			return nil, err
		}

		cfg := &ServiceConfig{
			Host:     host,
			Port:     port,
			User:     username,
			Password: password,
			Root:     root,
			Insecure: true,
			Timeout:  10 * time.Second,
		}

		requestedPath := "narsil.out"
		if request.Params.Arguments != nil {
			if arg, ok := request.Params.Arguments["path"].(string); ok && arg != "" {
				requestedPath = arg
			}
		}

		resolvedPath := path.Join(cfg.Root, requestedPath)

		data, err := fetchViaSSHCommand(cfg, resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("Error: %w", err)
		}

		outputPath := path.Base(requestedPath)
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return nil, fmt.Errorf("Error writing file: %w", err)
		}

		message := fmt.Sprintf("Successfully fetched %s (%d bytes) to %s", resolvedPath, len(data), outputPath)
		return mcp.NewToolResultText(message), nil
	}
}

type ServiceConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Root     string
	Insecure bool
	Timeout  time.Duration
}

func fetchViaSSHCommand(cfg *ServiceConfig, filePath string) ([]byte, error) {
	auth := []ssh.AuthMethod{ssh.Password(cfg.Password)}
	hostKeyCallback := ssh.InsecureIgnoreHostKey()

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.Timeout,
	}

	address := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	client, err := ssh.Dial("tcp", address, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(fmt.Sprintf("cat %s", filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w (output: %s)", err, string(output))
	}

	return output, nil
}

func getEnvTrimmedService(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func getEnvPortService(key string, defaultPort int) (int, error) {
	portRaw := getEnvTrimmedService(key)
	if portRaw == "" {
		return defaultPort, nil
	}
	parsed, err := strconv.Atoi(portRaw)
	if err != nil || parsed <= 0 || parsed > 65535 {
		return 0, fmt.Errorf("invalid %s: %s", key, portRaw)
	}
	return parsed, nil
}
