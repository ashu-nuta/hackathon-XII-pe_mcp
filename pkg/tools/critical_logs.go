package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CriticalLogs defines the critical_logs tool.
func CriticalLogs() mcp.Tool {
	return mcp.NewTool("critical_logs",
		mcp.WithDescription("Fetch critical kernel, crash, and fatal service logs from the SSH host"),
		mcp.WithString("lines",
			mcp.Description("Optional number of lines to return per section (default 50, max 500)"),
		),
	)
}

// CriticalLogsHandler implements the handler for the critical_logs tool.
func CriticalLogsHandler() server.ToolHandlerFunc {
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

		command := buildCriticalLogsCommand(lookback, lines)
		output, err := executeSSHCommandWithNewClient(cfg, command)
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(output)), nil
	}
}

func buildCriticalLogsCommand(lookback int, lines int) string {
	return fmt.Sprintf(`sudo -n sh -c '
echo "=== kernel_logs ==="
if [ -f /var/log/messages ]; then
  out=$(tail -n %d /var/log/messages | grep -i "kernel" | egrep -i "crit|critical|panic|oops|fatal|bug|segfault|backtrace|stack" | tail -n %d || true)
  if [ -n "$out" ]; then printf "%%s\n" "$out"; else tail -n %d /var/log/messages; fi
else
  echo "/var/log/messages not found"
fi

echo ""
echo "=== crash_logs ==="
if [ -d /home/log/crash ]; then
  file_count=$(find /home/log/crash -maxdepth 1 -type f 2>/dev/null | wc -l)
  echo "file_count=$file_count"
  found=0
  for f in /home/log/crash/*; do
    if [ -f "$f" ]; then
      found=1
      echo "--- $(basename "$f") ---"
      count=$(grep -ci -E "panic|fatal|segfault|oops|assert|crash|backtrace|stack" "$f" 2>/dev/null || true)
      echo "critical_matches=$count"
      if [ "$count" -gt 0 ]; then
        crit=$(grep -i -E "panic|fatal|segfault|oops|assert|crash|backtrace|stack" "$f" 2>/dev/null | tail -n %d)
        if [ -n "$crit" ]; then printf "%%s\n" "$crit"; fi
      else
        echo "(no critical markers; showing tail)"
        tail -n %d "$f"
      fi
    fi
  done
  if [ "$found" -eq 0 ]; then echo "no crash logs found"; fi
else
  echo "/home/log/crash not found"
fi

echo ""
echo "=== fatal_service_logs ==="
if [ -d /home/nutanix/data/logs ]; then
  fatal_count=$(find /home/nutanix/data/logs -maxdepth 1 -type f -name "*.FATAL" -size +0c 2>/dev/null | wc -l)
  echo "fatal_files=$fatal_count"
  if [ "$fatal_count" -gt 0 ]; then
    for f in /home/nutanix/data/logs/*.FATAL; do
      if [ -s "$f" ]; then
        echo "--- $(basename "$f") ---"
        tail -n %d "$f"
      fi
    done
  else
    echo "no fatal service logs found"
  fi
else
  echo "/home/nutanix/data/logs not found"
fi
'`, lookback, lines, lines, lines, lines, lines)
}
