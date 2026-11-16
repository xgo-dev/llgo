#!/usr/bin/env python3
"""Keep LLGo development setup out of the user-facing README."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[2]
DEVELOPMENT_COMMAND = re.compile(
    r"\b(?:mise|pixi)\b"
    r"|(?:\./|\b(?:bash|source)\s+)install\.sh\b"
    r"|\bgo\s+(?:build|install|test|run|fmt|vet)\b[^\n]*\./"
    r"|(?:\./|\b(?:bash|source)\s+)dev/"
    r"|\bgit\s+clone\s+\S*github\.com[/:]xgo-dev/llgo(?:\.git)?\b",
    re.IGNORECASE,
)
DEVELOPMENT_HEADING = re.compile(
    r"^#{1,6}\s+.*(?:\b(?:mise|pixi)\b|\b(?:manual|source)\s+"
    r"(?:installation|install|builds?)\b|\binstall\s+from\s+source\b|"
    r"\bdevelopment\s+(?:environment|setup)\b)",
    re.IGNORECASE,
)


def check_readme(source: str) -> list[str]:
    errors = []
    fence = None
    for number, line in enumerate(source.splitlines(), 1):
        marker = re.match(r"^\s*(`{3,}|~{3,})", line)
        if marker:
            token = marker[1]
            if fence is None:
                fence = token
            elif token[0] == fence[0] and len(token) >= len(fence):
                fence = None
            continue
        if (fence and DEVELOPMENT_COMMAND.search(line)) or (
            not fence and DEVELOPMENT_HEADING.search(line)
        ):
            errors.append(
                f"README.md:{number}: move development setup to CONTRIBUTION.md"
            )
    if not re.search(r"\]\(CONTRIBUTION\.md(?:#[^)]+)?\)", source):
        errors.append("README.md: link to CONTRIBUTION.md for contributor setup")
    return errors


def main() -> int:
    errors = check_readme((ROOT / "README.md").read_text())
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print("Documentation audience checks passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
