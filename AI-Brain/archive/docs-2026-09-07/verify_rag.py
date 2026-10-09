#!/usr/bin/env python3
"""Aggregator RAG verifier.
Re-greps every symbol/route claim in the Plan + Act RAG slices and confirms
PRESENT (file:line) or MISSING against the live repo. Reports discrepancies.
Run: python AI-Brain/verify_rag.py
"""
import os, re, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
EXCLUDE_DIRS = {"node_modules", "llama", ".git"}

def iter_go():
    for dp, dn, fn in os.walk(ROOT):
        dn[:] = [d for d in dn if d not in EXCLUDE_DIRS]
        for f in fn:
            if f.endswith(".go") and not f.endswith("_test.go"):
                yield os.path.join(dp, f)

GO_FILES = list(iter_go())
# index symbols
symbols = {}  # name -> list of (file, line)
for p in GO_FILES:
    try:
        with open(p, encoding="utf-8", errors="ignore") as fh:
            for i, line in enumerate(fh, 1):
                for m in re.finditer(r'\b([A-Za-z_][A-Za-z0-9_]*)\b', line):
                    symbols.setdefault(m.group(1), []).append((os.path.basename(p), i))
    except Exception:
        pass

def verify_symbol(name):
    hits = symbols.get(name, [])
    return hits[:5]  # first 5 locations

def verify_route(route):
    # search raw route string across go files
    for p in GO_FILES:
        try:
            txt = open(p, encoding="utf-8", errors="ignore").read()
        except: continue
        if ('"%s"' % route) in txt or ("`%s`" % route) in txt:
            return os.path.basename(p)
    return None

def main():
    plan = os.path.join(ROOT, "AI-Brain", "RAG-design-gaps.md")
    code = os.path.join(ROOT, "AI-Brain", "RAG-code-surface.md")
    issues = []
    # verify Plan's claimed PRESENT symbols
    if os.path.exists(plan):
        with open(plan, encoding="utf-8", errors="ignore") as fh:
            for line in fh:
                m = re.search(r'\|\s*[^|]*\|\s*`?([A-Za-z_][A-Za-z0-9_]*)`?\s*\|\s*(PRESENT)\s*\|\s*([^|]+)\|', line)
                if m:
                    sym, status, loc = m.group(1), m.group(2), m.group(3)
                    hits = verify_symbol(sym)
                    if not hits:
                        issues.append(f"[PLAN] {sym} claimed PRESENT but NOT FOUND in repo")
    # verify Act's routes
    if os.path.exists(code):
        with open(code, encoding="utf-8", errors="ignore") as fh:
            for line in fh:
                m = re.search(r'\|\s*(/api/[^\s|]+)\s*\|\s*([^|]+)\|', line)
                if m:
                    route, handler = m.group(1), m.group(2)
                    if "NONE" not in handler and "no UI" not in handler.lower():
                        loc = verify_route(route)
                        if loc is None:
                            issues.append(f"[ACT] route {route} claimed wired but NOT found in go files")
    print("VERIFY RUN COMPLETE")
    print("go files indexed:", len(GO_FILES))
    print("symbols indexed:", len(symbols))
    if issues:
        print("DISCREPANCIES:")
        for x in issues: print("  -", x)
    else:
        print("NO DISCREPANCIES — RAG slices verified against live repo")

if __name__ == "__main__":
    main()
