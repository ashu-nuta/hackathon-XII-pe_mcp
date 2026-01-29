import subprocess
import time
from datetime import datetime

# Configuration
SESSION_PREFIX = "agent"    # Base name for tmux sessions (will add timestamp)
WAIT_TIME = 60              # Seconds to wait for agent to complete
OUTPUT_DIR = "/home/ashutosh.kumar3/src/mcp-nutanix"  # Where to save output files

# List of commands to send to the Cursor Agent's input box
commands = [
    "analyse critical logs and return response in json format",
    # Add more commands here
]

def generate_session_name():
    """Generate unique session name with timestamp"""
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    return f"{SESSION_PREFIX}_{timestamp}"

def generate_output_filename(cmd):
    """Generate unique output filename with timestamp"""
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    # Create a short sanitized version of the command for the filename
    cmd_short = cmd[:30].replace(" ", "_").replace("/", "-")
    return f"{OUTPUT_DIR}/response_{timestamp}_{cmd_short}.txt"

def create_new_session(session_name):
    """Create a new detached tmux session and start agent inside"""
    try:
        # Create new detached session
        subprocess.run(["tmux", "new-session", "-d", "-s", session_name], check=True)
        print(f"✓ Created tmux session: {session_name}")
        
        # Start the agent inside the session
        time.sleep(1)
        subprocess.run(["tmux", "send-keys", "-t", session_name, "agent", "Enter"], check=True)
        print(f"✓ Started Cursor Agent in session")
        
        # Wait for agent to initialize
        time.sleep(3)
        return True
    except subprocess.CalledProcessError as e:
        print(f"ERROR: Failed to create session: {e}")
        return False

def get_pane_line_count(session):
    """Get current number of lines in the pane"""
    try:
        result = subprocess.run(
            ["tmux", "capture-pane", "-t", session, "-p", "-S", "-10000", "-E", "-1"],
            capture_output=True, text=True, check=True
        )
        return len(result.stdout.splitlines())
    except:
        return 0

def clear_pane_history(session):
    """Clear the tmux pane history"""
    try:
        subprocess.run(["tmux", "clear-history", "-t", session], check=True)
        return True
    except:
        return False

def send_to_tmux(session, text):
    """Send text to tmux session (works with TUI input boxes)"""
    try:
        subprocess.run(["tmux", "send-keys", "-t", session, text], check=True)
        subprocess.run(["tmux", "send-keys", "-t", session, "Enter"], check=True)
        return True
    except subprocess.CalledProcessError:
        print(f"ERROR: Failed to send to tmux session '{session}'")
        return False

def capture_response_only(session, start_line):
    """Capture only the new content after start_line"""
    try:
        result = subprocess.run(
            ["tmux", "capture-pane", "-t", session, "-p", "-S", "-10000", "-E", "-1"],
            capture_output=True, text=True, check=True
        )
        all_lines = result.stdout.splitlines()
        # Get only lines after where we started (the response)
        response_lines = all_lines[start_line:]
        return "\n".join(response_lines)
    except subprocess.CalledProcessError as e:
        print(f"ERROR: Failed to capture output: {e}")
        return None

def save_response(response, output_file, cmd):
    """Save the response to a text file"""
    with open(output_file, 'w') as f:
        f.write(f"Command: {cmd}\n")
        f.write(f"Captured: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write("=" * 50 + "\n\n")
        f.write(response)
    return True

def cleanup_session(session):
    """Kill the tmux session"""
    try:
        subprocess.run(["tmux", "kill-session", "-t", session], check=True)
        print(f"✓ Cleaned up session: {session}")
    except:
        pass

def main():
    # Generate unique session name
    session_name = generate_session_name()
    
    print("=" * 50)
    print(f"Session: {session_name}")
    print(f"Wait time: {WAIT_TIME} seconds")
    print("=" * 50)
    
    # Create new tmux session with agent
    if not create_new_session(session_name):
        return
    
    print("=" * 50)
    
    # Send each command and capture its response separately
    for cmd in commands:
        output_file = generate_output_filename(cmd)
        
        # Get current line count BEFORE sending command
        start_line = get_pane_line_count(session_name)
        print(f"Pane has {start_line} lines before command")
        
        print(f"Sending: {cmd}")
        success = send_to_tmux(session_name, cmd)
        if not success:
            break
        
        # Wait for agent to process
        print(f"Waiting {WAIT_TIME} seconds for response...")
        time.sleep(WAIT_TIME)
        
        # Capture ONLY the response (lines after start_line)
        print(f"Capturing response...")
        response = capture_response_only(session_name, start_line)
        
        if response:
            save_response(response, output_file, cmd)
            print(f"✓ Response saved to: {output_file}")
        else:
            print(f"✗ Failed to capture response")
        
        print("-" * 50)
    
    # Optionally cleanup (comment out to keep session for debugging)
    # cleanup_session(session_name)
    
    print("=" * 50)
    print(f"Done!")
    print(f"Attach to session: tmux attach -t {session_name}")

if __name__ == "__main__":
    main()
