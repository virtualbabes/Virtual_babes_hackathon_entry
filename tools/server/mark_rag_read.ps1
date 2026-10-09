# mark_rag_read.ps1 — sweep utility (TEMP, not part of the app; no product code touched)
# Marks coverage-ledger rows `read` with their recorded ranges.
# Reads pairs ONE PER LINE from a UTF-8 file (avoids all shell-array quoting issues):
#     <ragFile>|<ledgerPath>|<ranges>
# The script REFUSES any pair whose regex does not match EXACTLY ONE row (0 or >1 = loud
# failure, never a guess), and it only ever rewrites the `pending |  |` tail of a matched row.
param([Parameter(Mandatory=$true)][string]$ListFile)

$ledger = 'AI-Brain\RAG\19_coverage_ledger.md'
$s = [IO.File]::ReadAllText($ledger, [Text.Encoding]::UTF8)
$ok = 0; $fail = 0

foreach ($pair in [IO.File]::ReadAllLines($ListFile, [Text.Encoding]::UTF8)) {
    if ([string]::IsNullOrWhiteSpace($pair)) { continue }
    $parts = $pair.Split('|')
    if ($parts.Count -ne 3) { Write-Host "SKIP (need RAG|path|ranges): $pair" -ForegroundColor Red; $fail++; continue }
    $rag = $parts[0].Trim(); $file = $parts[1].Trim(); $range = $parts[2].Trim()

    $rx = '\| ' + [regex]::Escape($rag) + ' \| `' + [regex]::Escape($file) + '` \| ([a-z]+) \| ([0-9]+) \| ([0-9]+) \| full-read \| pending \|  \|'
    $hits = ([regex]::Matches($s, $rx)).Count
    if ($hits -ne 1) { Write-Host "REFUSED $file -> $hits matching rows" -ForegroundColor Red; $fail++; continue }

    $f = $file; $r = $range; $g = $rag
    $ev = [System.Text.RegularExpressions.MatchEvaluator]{
        param($m)
        '| ' + $g + ' | `' + $f + '` | ' + $m.Groups[1].Value + ' | ' + $m.Groups[2].Value + ' | ' + $m.Groups[3].Value + ' | full-read | read | ' + $r + ' |'
    }
    $s = [regex]::Replace($s, $rx, $ev)
    $ok++
    Write-Host "read <- $file" -ForegroundColor Green
}

[IO.File]::WriteAllText($ledger, $s, (New-Object System.Text.UTF8Encoding($false)))
Write-Host "marked=$ok refused=$fail"
