#!/usr/bin/env python3
"""
Hybrid MCP + Cursor Agent Analysis Script

This script:
1. Calls MCP tools directly (no Cursor approval needed)
2. Sends the data to Cursor agent for AI analysis
3. Outputs the analyzed result

Usage:
    python abc2.py
"""

import subprocess
import json
import os
import sys

# MCP Server Configuration
MCP_SERVER = "/home/ashutosh.kumar3/src/mcp-nutanix/bin/mcp-nutanix"

# Environment variables for MCP server
MCP_ENV = {
    **os.environ,
    "NUTANIX_ENDPOINT": "10.96.0.16",
    "NUTANIX_USERNAME": "admin",
    "NUTANIX_PASSWORD": "Nutanix.123",
    "NUTANIX_INSECURE": "true",
    "SSH_HOST": "10.96.0.16;10.96.0.17;10.96.0.18",
    "SSH_USERNAME": "nutanix",
    "SSH_PASSWORD": "RDMCluster.123",
    "SSH_PORT": "22",
    "SSH_LOG_ROOT": "/home/nutanix/data/logs"
}

OUTPUT_FILE = "analysis_output.txt"
DATA_FILE = "cluster_data.txt"  # File to store collected data for Cursor


def call_mcp_tool(tool_name: str, arguments: dict = None, timeout: int = 120):
    """
    Call an MCP tool directly, bypassing Cursor's approval prompts.
    
    Args:
        tool_name: Name of the MCP tool (e.g., "vm_list", "critical_logs")
        arguments: Optional dict of arguments for the tool
        timeout: Timeout in seconds
    
    Returns:
        dict with the tool result
    """
    # MCP uses JSON-RPC 2.0 protocol
    messages = [
        # Initialize the connection
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "abc2-script", "version": "1.0.0"}
            }
        },
        # Send initialized notification
        {"jsonrpc": "2.0", "method": "notifications/initialized"},
        # Call the tool
        {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": arguments or {}
            }
        }
    ]
    
    # Send all messages as newline-delimited JSON
    input_data = "\n".join(json.dumps(msg) for msg in messages) + "\n"
    
    try:
        proc = subprocess.Popen(
            [MCP_SERVER],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=MCP_ENV,
            text=True
        )
        
        stdout, stderr = proc.communicate(input=input_data, timeout=timeout)
        
        # Parse responses (newline-delimited JSON)
        for line in stdout.strip().split("\n"):
            if line:
                try:
                    r = json.loads(line)
                    if r.get("id") == 2:
                        return r.get("result", {})
                except json.JSONDecodeError:
                    continue
        
        return {"error": "No result found", "stdout": stdout, "stderr": stderr}
    
    except subprocess.TimeoutExpired:
        proc.kill()
        return {"error": f"Timeout after {timeout} seconds"}
    except Exception as e:
        return {"error": str(e)}


def extract_text_from_mcp_result(result: dict) -> str:
    """Extract text content from MCP tool result."""
    if "content" in result:
        texts = []
        for item in result["content"]:
            if item.get("type") == "text":
                texts.append(item.get("text", ""))
        return "\n".join(texts)
    elif "error" in result:
        return f"Error: {result['error']}"
    else:
        return json.dumps(result, indent=2)


def analyze_with_cursor(data: str, prompt: str, timeout: int = 300):
    """
    Send data to Cursor agent CLI for AI analysis.
    
    Args:
        data: The raw data to analyze
        prompt: Instructions for the AI
        timeout: Timeout in seconds for agent command
    
    Returns:
        AI analysis as a string
    """
    # Combine prompt and data into a single message
    full_message = f"""{prompt}

Here is the data to analyze:

{data}
"""
    
    try:
        # Use agent command to send prompt to Cursor
        proc = subprocess.Popen(
            ["agent", full_message],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        stdout, stderr = proc.communicate(timeout=timeout)
        
        if proc.returncode != 0:
            return f"Agent CLI error (exit code {proc.returncode}):\n{stderr}\n{stdout}"
        
        return stdout.strip()
    
    except FileNotFoundError:
        return "Error: agent CLI not found. Make sure the 'agent' command is available in your PATH."
    except subprocess.TimeoutExpired:
        proc.kill()
        return f"Error: Agent analysis timed out after {timeout} seconds"
    except Exception as e:
        return f"Error calling agent CLI: {str(e)}"


def save_data_for_cursor(data: str, prompt: str):
    """
    Save data to a file that can be opened in Cursor for analysis.
    
    Args:
        data: The raw data to analyze
        prompt: Instructions for the AI
    
    Returns:
        Path to the saved file
    """
    content = f"""# Cluster Analysis Request

## Analysis Instructions

{prompt}

## Collected Data

```
{data}
```

---
Open this file in Cursor and use Cmd+K or Ctrl+K to ask the AI to analyze the data above.
"""
    
    filepath = os.path.join(os.path.dirname(os.path.abspath(__file__)), DATA_FILE)
    with open(filepath, "w") as f:
        f.write(content)
    
    return filepath


def main():
    print("=" * 60)
    print("MCP + Cursor Agent Analysis Pipeline")
    print("=" * 60)
    
    # =========================================================================
    # STEP 1: Fetch data from MCP (no Cursor approval needed!)
    # =========================================================================
    print("\n[1/4] Fetching critical logs from cluster...")
    
    logs_result = call_mcp_tool("critical_logs", {"lines": "100"})
    logs_data = extract_text_from_mcp_result(logs_result)
    
    print(f"      Got {len(logs_data)} characters of log data")
    
    # Optionally fetch more data
    print("\n[2/4] Fetching VM list...")
    vm_result = call_mcp_tool("vm_list")
    vm_data = extract_text_from_mcp_result(vm_result)
    print(f"      Got {len(vm_data)} characters of VM data")
    
    # Combine all data
    combined_data = f"""
=== CRITICAL LOGS ===
{logs_data}

=== VM LIST ===
{vm_data}
"""
    
    analysis_prompt = """You are a Nutanix cluster expert. Analyze the following cluster data:

1. **Critical Issues**: Identify any errors, crashes, or critical warnings in the logs
2. **VM Health**: Check the VM list for any issues or anomalies
3. **Recommendations**: Provide specific, actionable recommendations to fix any issues
4. **Priority**: Rank issues by severity (Critical, High, Medium, Low)

Be concise but thorough. Format your response with clear sections and bullet points."""

    # =========================================================================
    # STEP 2: Save data file for Cursor
    # =========================================================================
    print("\n[3/4] Saving data for Cursor analysis...")
    
    data_filepath = save_data_for_cursor(combined_data, analysis_prompt)
    print(f"      Data saved to: {data_filepath}")
    
    # =========================================================================
    # STEP 3: Send to Cursor agent for AI analysis
    # =========================================================================
    print("\n[4/4] Sending to Cursor agent for analysis...")
    print("      (This may take a moment...)")
    
    analysis = analyze_with_cursor(combined_data, analysis_prompt)
    
    # =========================================================================
    # STEP 4: Output results
    # =========================================================================
    print("\n" + "=" * 60)
    print("CURSOR AGENT ANALYSIS RESULTS")
    print("=" * 60)
    print(analysis)
    
    # Save to file
    with open(OUTPUT_FILE, "w") as f:
        f.write("=" * 60 + "\n")
        f.write("CURSOR AGENT ANALYSIS RESULTS\n")
        f.write(f"Generated by abc2.py\n")
        f.write("=" * 60 + "\n\n")
        f.write(analysis)
        f.write("\n\n" + "=" * 60 + "\n")
        f.write("RAW DATA COLLECTED\n")
        f.write("=" * 60 + "\n")
        f.write(combined_data)
    
    print(f"\n✓ Full report saved to: {OUTPUT_FILE}")
    print(f"✓ Data file for manual analysis: {data_filepath}")


# =============================================================================
# Alternative: Run specific tools interactively
# =============================================================================

def interactive_mode():
    """Interactive mode to call specific MCP tools."""
    available_tools = [
        ("vm_list", "List all VMs", {}),
        ("vm_count", "Count VMs", {}),
        ("critical_logs", "Get critical logs", {"lines": "50"}),
        ("crash_logs_critical", "Get crash logs", {"lines": "50"}),
        ("kernel_logs_critical", "Get kernel logs", {"lines": "50"}),
        ("ssh_exec", "Execute SSH command", {"command": "uptime"}),
        ("api_namespaces_list", "List API namespaces", {}),
    ]
    
    print("\nAvailable MCP tools:")
    for i, (name, desc, _) in enumerate(available_tools, 1):
        print(f"  {i}. {name} - {desc}")
    
    choice = input("\nEnter tool number (or 'q' to quit): ").strip()
    
    if choice.lower() == 'q':
        return
    
    try:
        idx = int(choice) - 1
        tool_name, desc, default_args = available_tools[idx]
        
        print(f"\nCalling {tool_name}...")
        result = call_mcp_tool(tool_name, default_args)
        text = extract_text_from_mcp_result(result)
        
        print("\nResult:")
        print(text[:2000] + "..." if len(text) > 2000 else text)
        
        # Ask if they want AI analysis
        if input("\nAnalyze this with Cursor agent? (y/n): ").lower() == 'y':
            prompt = input("Analysis prompt (or press Enter for default): ").strip()
            if not prompt:
                prompt = f"Analyze this {desc.lower()} output and provide insights:"
            
            print("\nSending to Cursor agent for analysis...")
            analysis = analyze_with_cursor(text, prompt)
            print("\n" + "=" * 40)
            print("Cursor Agent Analysis:")
            print("=" * 40)
            print(analysis)
    
    except (ValueError, IndexError):
        print("Invalid choice")


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--interactive":
        interactive_mode()
    else:
        main()

