import subprocess
import os

OUTPUT_FILE = "abcd2.txt"

# List of commands to execute
commands = [
    "ls",
    "pwd",
    "whoami",
    "date",
    "agent list all vms",
    "agent analyse critical logs"
]

def main():
    print("Starting execution...")
    
    # Build a single script from all commands
    # Each command's output will be labeled
    script_lines = []
    for cmd in commands:
        # Echo the command being run, then run it
        script_lines.append(f'echo "$ {cmd}"')
        script_lines.append(cmd)
        script_lines.append('echo ""')  # Add blank line between commands
    
    # Join all commands into a single script
    full_script = "\n".join(script_lines)
    
    print("Executing commands in terminal...")
    
    # Run the script in bash and capture output
    # Send multiple newlines (Enter keypresses) to stdin in case any command prompts for input
    # If not needed, they'll be ignored by non-interactive commands
    result = subprocess.run(
        ["/bin/bash", "-c", full_script],
        capture_output=True,
        text=True,
        input="\n" * 10000  # Send 10 "Enter" keypresses - adjust as needed
    )
    
    # Combine stdout and stderr
    output = result.stdout
    if result.stderr:
        output += "\n--- STDERR ---\n" + result.stderr
    
    # Write output to file
    with open(OUTPUT_FILE, "w") as f:
        f.write("=" * 50 + "\n")
        f.write("TERMINAL OUTPUT\n")
        f.write("=" * 50 + "\n\n")
        f.write(output)
        f.write("\n" + "=" * 50 + "\n")
        f.write(f"Exit code: {result.returncode}\n")
        f.write("=" * 50 + "\n")
    
    print(f"Output saved to {OUTPUT_FILE}")
    print(f"Exit code: {result.returncode}")
    
    # Also print the output to console
    print("\n--- Output Preview ---")
    print(output)

if __name__ == "__main__":
    main()