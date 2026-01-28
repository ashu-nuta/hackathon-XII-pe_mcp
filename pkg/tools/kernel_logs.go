package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultKernelLogLines = 50
	maxKernelLogLines     = 500
	minKernelLogLines     = 1
)

// KernelLogsCritical defines the kernel_logs_critical tool.
func KernelLogsCritical() mcp.Tool {
	return mcp.NewTool("kernel_logs_critical",
		mcp.WithDescription("Fetch recent critical kernel logs from ~/../../var/log/messages on each SSH_HOST entry"),
		mcp.WithString("lines",
			mcp.Description("Optional number of lines to return (default 50, max 500)"),
		),
	)
}

// KernelLogsCriticalHandler implements the handler for the kernel_logs_critical tool.
func KernelLogsCriticalHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx

		lines, err := parseKernelLogLines(request)
		if err != nil {
			return nil, err
		}

		cfg, err := getSSHConfig()
		if err != nil {
			return nil, err
		}

		lookback := lines * 4
		if lookback < 200 {
			lookback = 200
		}
		if lookback > maxKernelLogLines*4 {
			lookback = maxKernelLogLines * 4
		}

		command := buildKernelCriticalCommand(lookback, lines)
		output, err := runSSHCommandOnHosts(cfg, command)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(output)), nil
	}
}

func parseKernelLogLines(request mcp.CallToolRequest) (int, error) {
	lines := defaultKernelLogLines
	if request.Params.Arguments == nil {
		return lines, nil
	}

	raw, ok := request.Params.Arguments["lines"].(string)
	if !ok {
		return lines, nil
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return lines, nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("lines must be an integer")
	}
	if parsed < minKernelLogLines || parsed > maxKernelLogLines {
		return 0, fmt.Errorf("lines must be between %d and %d", minKernelLogLines, maxKernelLogLines)
	}

	return parsed, nil
}

func buildKernelCriticalCommand(lookback int, lines int) string {
	return fmt.Sprintf(
		"cd ~/../../var/log && sudo -n sh -c 'out=$(tail -n %d messages | grep -i \"kernel\" | egrep -i \"crit|critical|panic|oops|fatal|bug\" | tail -n %d || true); if [ -n \"$out\" ]; then printf \"%%s\\n\" \"$out\"; else tail -n %d messages; fi'",
		lookback,
		lines,
		lines,
	)
}
