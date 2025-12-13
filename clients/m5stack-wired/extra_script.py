Import("env")
import subprocess

# Get git commit hash
try:
    git_hash = subprocess.check_output(['git', 'rev-parse', '--short', 'HEAD']).decode().strip()
except:
    git_hash = "unknown"

# Add build flag
env.Append(CPPDEFINES=[
    ('FW_VERSION', f'\\"{git_hash}\\"')
])

print(f"Setting FW_VERSION to: {git_hash}")
