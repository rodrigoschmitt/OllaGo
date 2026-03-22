#!/usr/bin/env python3
"""
update_models.py
----------------
Queries the locally installed Ollama models via `ollama list`, then patches
the <select id="model-select"> block in static/index.html so the hard-coded
option list reflects whatever is actually installed.

Usage (run from any directory):
    python3 scripts/update_models.py

Optional flags:
    --html PATH   Override the default path to index.html
    --dry-run     Print the new option list without writing the file
"""

import argparse
import re
import subprocess
import sys
from pathlib import Path

# ── Defaults ──────────────────────────────────────────────────────────────────
REPO_ROOT  = Path(__file__).resolve().parent.parent
HTML_PATH  = REPO_ROOT / "static" / "index.html"
OPTION_INDENT = "      "   # 6 spaces — matches the indentation in index.html


# ── Ollama helpers ─────────────────────────────────────────────────────────────

def list_ollama_models() -> list[str]:
    """Run `ollama list` and return a list of model name strings."""
    try:
        result = subprocess.run(
            ["ollama", "list"],
            capture_output=True,
            text=True,
            check=True,
        )
    except FileNotFoundError:
        sys.exit(
            "Error: 'ollama' binary not found.  "
            "Is Ollama installed and available on PATH?"
        )
    except subprocess.CalledProcessError as exc:
        sys.exit(f"Error: 'ollama list' exited with code {exc.returncode}:\n{exc.stderr.strip()}")

    models: list[str] = []
    for line in result.stdout.splitlines()[1:]:   # first line is the header
        parts = line.split()
        if parts:
            models.append(parts[0])               # column 0 is "NAME" (e.g. gemma3:12b)

    return models


# ── HTML helpers ───────────────────────────────────────────────────────────────

def build_options_block(models: list[str]) -> str:
    """Return the indented <option> lines for the given model list."""
    return "\n".join(
        f'{OPTION_INDENT}<option value="{m}">{m}</option>'
        for m in models
    )


# Matches everything between (and including) the opening <select> tag and </select>,
# even when the existing options span multiple lines.
_SELECT_RE = re.compile(
    r'(<select\s+id="model-select"[^>]*>)\s*.*?\s*(</select>)',
    re.DOTALL,
)


def patch_html(path: Path, models: list[str], dry_run: bool = False) -> None:
    html = path.read_text(encoding="utf-8")

    options_block = build_options_block(models)
    replacement   = f"\\1\n{options_block}\n    \\2"

    new_html, substitutions = _SELECT_RE.subn(replacement, html)

    if substitutions == 0:
        sys.exit(
            f'Error: could not locate <select id="model-select"> in {path}.\n'
            "Has the HTML structure changed?"
        )

    if dry_run:
        print("── Dry-run: would write the following options ──")
        print(options_block)
        return

    path.write_text(new_html, encoding="utf-8")


# ── Entry point ────────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--html",
        type=Path,
        default=HTML_PATH,
        metavar="PATH",
        help="Path to index.html (default: static/index.html)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print changes without modifying the file",
    )
    args = parser.parse_args()

    if not args.html.exists():
        sys.exit(f"Error: HTML file not found at {args.html}")

    models = list_ollama_models()

    if not models:
        sys.exit(
            "No models returned by 'ollama list'.\n"
            "Make sure Ollama is running and at least one model is pulled."
        )

    patch_html(args.html, models, dry_run=args.dry_run)

    status = "Would update" if args.dry_run else "Updated"
    print(f"{status} {args.html} with {len(models)} model(s):")
    for m in models:
        print(f"  • {m}")


if __name__ == "__main__":
    main()
