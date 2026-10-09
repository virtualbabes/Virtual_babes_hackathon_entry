#!/usr/bin/env python3
import argparse
import subprocess
import sys
import tomllib
from pathlib import Path

# Constants for elegant output
class Colors:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'

TOOLS_ROOT = Path(__file__).parent
PROJECT_ROOT = TOOLS_ROOT.parent
CONFIG_FILE = TOOLS_ROOT / "config.toml"

def load_config():
    if not CONFIG_FILE.exists():
        print(f"{Colors.FAIL}Error: Configuration file not found.{Colors.ENDC}")
        sys.exit(1)
    with open(CONFIG_FILE, "rb") as f:
        return tomllib.load(f)

def get_interpreter(path: Path):
    ext = path.suffix.lower()
    if ext == ".ps1": return ["powershell", "-ExecutionPolicy", "Bypass", "-File"]
    if ext == ".sh": return ["bash"]
    if ext == ".py": return [sys.executable]
    if ext == ".js": return ["node"]
    return []

def execute(tool_cfg: dict, args: list = None):
    full_path = PROJECT_ROOT / tool_cfg["path"]
    
    if not full_path.exists():
        print(f"{Colors.FAIL}Error: Tool not found at {full_path}{Colors.ENDC}")
        return

    interpreter = get_interpreter(full_path)
    cmd = interpreter + [str(full_path)] + (args or [])
    
    print(f"{Colors.OKBLUE}{Colors.BOLD}🚀 Executing: {tool_cfg['desc']}{Colors.ENDC}")
    print(f"{Colors.HEADER}{' '.join(cmd)}{Colors.ENDC}")
    
    try:
        subprocess.run(cmd, check=True)
    except subprocess.CalledProcessError as e:
        print(f"{Colors.FAIL}❌ Command failed with exit code {e.returncode}{Colors.ENDC}")
        sys.exit(e.returncode)

def diagnose(config):
    print(f"{Colors.HEADER}🔍 Running Toolchain Diagnostics...{Colors.ENDC}")
    all_ok = True
    for cat, tools in config.items():
        for name, cfg in tools.items():
            path = PROJECT_ROOT / cfg["path"]
            if path.exists():
                print(f"  {Colors.OKGREEN}✓{Colors.ENDC} {cat}/{name}")
            else:
                print(f"  {Colors.FAIL}✗{Colors.ENDC} {cat}/{name} (Missing: {cfg['path']})")
                all_ok = False
    if all_ok:
        print(f"{Colors.OKGREEN}✅ Toolchain integrity verified.{Colors.ENDC}")
    else:
        sys.exit(1)

def main():
    config = load_config()
    parser = argparse.ArgumentParser(description="NFT-Seduction Unified Toolchain")
    subparsers = parser.add_subparsers(dest="category", required=True)

    # Add diagnostic subcommand
    subparsers.add_parser("diagnose")

    # Dynamically add subparsers from config
    for category, tools in config.items():
        cat_parser = subparsers.add_parser(category)
        cat_parser.add_argument("tool", choices=tools.keys(), help="Tool to run")

    args = parser.parse_args()

    if args.category == "diagnose":
        diagnose(config)
    else:
        execute(config[args.category][args.tool])

if __name__ == "__main__":
    main()
