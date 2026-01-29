import subprocess
import time

# Configuration
SESSION_NAME = "agent"      # tmux session name
TARGET_TTY = "/dev/pts/0"   # Target terminal (run 'tty' in your terminal to find it)

# List of commands to send to the Cursor Agent's input box
commands = [
    "analyse critical logs and list all the vms \n",
    # Add more commands here
]

def send_to_tmux(session, text):
    """Send text to tmux session (works with TUI input boxes)"""
    try:
        # Send the text
        subprocess.run(["tmux", "send-keys", "-t", session, text], check=True)
        # Press Enter
        subprocess.run(["tmux", "send-keys", "-t", session, "Enter"], check=True)
        return True
    except subprocess.CalledProcessError:
        print(f"ERROR: Failed to send to tmux session '{session}'")
        return False

def check_session_exists(session):
    """Check if tmux session exists"""
    result = subprocess.run(["tmux", "has-session", "-t", session], 
                          capture_output=True)
    return result.returncode == 0

def get_tmux_session_tty(session):
    """Get the TTY of a tmux session's active pane"""
    try:
        result = subprocess.run(
            ["tmux", "display-message", "-t", session, "-p", "#{pane_tty}"],
            capture_output=True, text=True, check=True
        )
        return result.stdout.strip()
    except:
        return None

def find_tmux_session_by_tty(target_tty):
    """Find tmux session running in the specified TTY"""
    try:
        result = subprocess.run(
            ["tmux", "list-panes", "-a", "-F", "#{session_name}:#{pane_tty}"],
            capture_output=True, text=True, check=True
        )
        for line in result.stdout.strip().split('\n'):
            if ':' in line:
                session, tty = line.split(':', 1)
                if tty == target_tty:
                    return session
    except:
        pass
    return None

def main():
    print(f"Target TTY: {TARGET_TTY}")
    print(f"Target tmux session: {SESSION_NAME}")
    print("=" * 50)
    
    # First, try to find session by TTY
    session_by_tty = find_tmux_session_by_tty(TARGET_TTY)
    
    if session_by_tty:
        print(f"Found tmux session '{session_by_tty}' running in {TARGET_TTY}")
        target_session = session_by_tty
    elif check_session_exists(SESSION_NAME):
        actual_tty = get_tmux_session_tty(SESSION_NAME)
        print(f"Using tmux session '{SESSION_NAME}' (TTY: {actual_tty})")
        target_session = SESSION_NAME
    else:
        print(f"ERROR: No tmux session found!")
        print("\nTo set up:")
        print(f"  1. Open terminal {TARGET_TTY}")
        print(f"  2. Run: tmux new-session -s {SESSION_NAME}")
        print(f"  3. Inside tmux, start: agent")
        print(f"  4. Then run this script from another terminal")
        print(f"\nTo list existing sessions: tmux ls")
        print(f"To check your terminal: tty")
        return
    
    print("=" * 50)
    
    for cmd in commands:
        print(f"Sending: {cmd}")
        success = send_to_tmux(target_session, cmd)
        if not success:
            break
        time.sleep(2)  # Wait for agent to process before next command
    
    print("=" * 50)
    print(f"Done! Check your tmux session: tmux attach -t {target_session}")

if __name__ == "__main__":
    main()
