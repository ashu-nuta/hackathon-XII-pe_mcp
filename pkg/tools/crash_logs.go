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
	defaultCrashLogLines = 50
	maxCrashLogLines     = 500
	minCrashLogLines     = 1
)

// CrashLogsCritical defines the crash_logs_critical tool.
func CrashLogsCritical() mcp.Tool {
	return mcp.NewTool("crash_logs_critical",
		mcp.WithDescription("Fetch and summarize critical crash logs from /home/log/crash on the SSH host"),
		mcp.WithString("lines",
			mcp.Description("Optional number of lines per file to return (default 50, max 500)"),
		),
	)
}

// CrashLogsCriticalHandler implements the handler for the crash_logs_critical tool.
func CrashLogsCriticalHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx

		lines, err := parseCrashLogLines(request)
		if err != nil {
			return nil, err
		}

		cfg, err := getSSHConfig()
		if err != nil {
			return nil, err
		}

		command := buildCrashCriticalCommand(lines)
		output, err := executeSSHCommandWithNewClient(cfg, command)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(output)), nil
	}
}

func parseCrashLogLines(request mcp.CallToolRequest) (int, error) {
	lines := defaultCrashLogLines
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
	if parsed < minCrashLogLines || parsed > maxCrashLogLines {
		return 0, fmt.Errorf("lines must be between %d and %d", minCrashLogLines, maxCrashLogLines)
	}

	return parsed, nil
}

func buildCrashCriticalCommand(lines int) string {
	return fmt.Sprintf(
		"sudo -n sh -c 'cd /home/log/crash || exit 1; found=0; for f in *; do if [ -f \"$f\" ]; then found=1; echo \"=== $f ===\"; count=$(grep -ci -E \"panic|fatal|segfault|oops|assert|crash|backtrace|stack\" \"$f\" 2>/dev/null || true); echo \"critical_matches=$count\"; if [ \"$count\" -gt 0 ]; then crit=$(grep -i -E \"panic|fatal|segfault|oops|assert|crash|backtrace|stack\" \"$f\" 2>/dev/null | tail -n %d); if [ -n \"$crit\" ]; then printf \"%%s\\n\" \"$crit\"; fi; else echo \"(no critical markers; showing tail)\"; tail -n %d \"$f\"; fi; fi; done; if [ $found -eq 0 ]; then echo \"no crash logs found\"; fi'",
		lines,
		lines,
	)
}

