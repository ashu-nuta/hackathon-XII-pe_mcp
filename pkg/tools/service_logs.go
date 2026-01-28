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
		mcp.WithDescription("Fetch a file from /home/nutanix/data/logs via SSH for each SSH_HOST entry and save locally"),
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

		sshCfg := &SSHConfig{
			Host:     cfg.Host,
			Port:     cfg.Port,
			User:     cfg.User,
			Password: cfg.Password,
			Timeout:  cfg.Timeout,
		}

		hosts, err := getSSHHosts(sshCfg)
		if err != nil {
			return nil, err
		}

		baseName := path.Base(requestedPath)
		var outputBuilder strings.Builder
		successes := 0
		for i, host := range hosts {
			if i > 0 {
				outputBuilder.WriteString("\n")
			}

			hostCfg := *cfg
			hostCfg.Host = host
			data, err := fetchViaSSHCommand(&hostCfg, resolvedPath)
			if err != nil {
				outputBuilder.WriteString(fmt.Sprintf("ssh_host=%s error: %s", host, err.Error()))
				continue
			}

			outputPath := fmt.Sprintf("%s_%s", sanitizeHostForFileName(host), baseName)
			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				outputBuilder.WriteString(fmt.Sprintf("ssh_host=%s error writing file: %s", host, err.Error()))
				continue
			}

			successes++
			outputBuilder.WriteString(fmt.Sprintf("ssh_host=%s fetched %s (%d bytes) to %s", host, resolvedPath, len(data), outputPath))
		}

		if successes == 0 {
			return nil, fmt.Errorf("failed to fetch %s from all SSH hosts", resolvedPath)
		}

		return mcp.NewToolResultText(outputBuilder.String()), nil
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

	applyShellSafeEnv(session)

	output, err := session.CombinedOutput(fmt.Sprintf("cat %s", filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w (output: %s)", err, string(output))
	}

	return output, nil
}

func sanitizeHostForFileName(host string) string {
	sanitized := strings.TrimSpace(host)
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	return sanitized
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
