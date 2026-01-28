package tools

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Timeout  time.Duration
}

// SSHExec defines the ssh_exec tool
func SSHExec() mcp.Tool {
	return mcp.NewTool("ssh_exec",
		mcp.WithDescription("Execute a command on each SSH_HOST entry and return combined output"),
		mcp.WithString("command",
			mcp.Description("Command to execute on each SSH host"),
		),
	)
}

// SSHExecHandler implements the handler for the ssh_exec tool
func SSHExecHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx

		command := ""
		if request.Params.Arguments != nil {
			if arg, ok := request.Params.Arguments["command"].(string); ok {
				command = strings.TrimSpace(arg)
			}
		}
		if command == "" {
			return nil, fmt.Errorf("command is required")
		}

		cfg, err := getSSHConfig()
		if err != nil {
			return nil, err
		}

		output, err := runSSHCommandOnHosts(cfg, command)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(output)), nil
	}
}

// SSHExecBatch defines the ssh_exec_batch tool
func SSHExecBatch() mcp.Tool {
	return mcp.NewTool("ssh_exec_batch",
		mcp.WithDescription("Execute multiple commands on each SSH_HOST entry in order"),
		mcp.WithString("commands",
			mcp.Description("Newline-separated commands to execute in order on each SSH host"),
		),
	)
}

// SSHExecBatchHandler implements the handler for the ssh_exec_batch tool
func SSHExecBatchHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx

		commandsRaw := ""
		if request.Params.Arguments != nil {
			if arg, ok := request.Params.Arguments["commands"].(string); ok {
				commandsRaw = arg
			}
		}
		commands := parseCommands(commandsRaw)
		if len(commands) == 0 {
			return nil, fmt.Errorf("commands is required")
		}

		cfg, err := getSSHConfig()
		if err != nil {
			return nil, err
		}

		output, err := runSSHCommandsOnHosts(cfg, commands)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(output), nil
	}
}

func getSSHConfig() (*SSHConfig, error) {
	host := getEnvTrimmedService(envServiceHost)
	username := getEnvTrimmedService(envServiceUsername)
	password := getEnvTrimmedService(envServicePassword)
	if host == "" {
		return nil, fmt.Errorf("%s is required", envServiceHost)
	}
	if username == "" {
		return nil, fmt.Errorf("%s is required", envServiceUsername)
	}
	if password == "" {
		return nil, fmt.Errorf("%s is required", envServicePassword)
	}

	port, err := getEnvPortService(envServicePort, 22)
	if err != nil {
		return nil, err
	}

	return &SSHConfig{
		Host:     host,
		Port:     port,
		User:     username,
		Password: password,
		Timeout:  10 * time.Second,
	}, nil
}

func parseCommands(raw string) []string {
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	commands := make([]string, 0, len(lines))
	for _, line := range lines {
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		commands = append(commands, cmd)
	}
	return commands
}

func executeSSHCommandWithNewClient(cfg *SSHConfig, command string) ([]byte, error) {
	client, err := newSSHClient(cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return executeSSHCommand(client, command)
}

func newSSHClient(cfg *SSHConfig) (*ssh.Client, error) {
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
	return client, nil
}

func executeSSHCommand(client *ssh.Client, command string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	applyShellSafeEnv(session)

	output, err := session.CombinedOutput(command)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w (output: %s)", err, string(output))
	}
	return output, nil
}

func applyShellSafeEnv(session *ssh.Session) {
	// Prevent non-interactive shells from sourcing problematic startup files.
	_ = session.Setenv("BASH_ENV", "/dev/null")
	_ = session.Setenv("ENV", "/dev/null")
}
