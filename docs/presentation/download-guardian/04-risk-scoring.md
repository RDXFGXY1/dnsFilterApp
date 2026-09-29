# 4. Risk Scoring and Decision Engine

## Design principles

The first risk engine should be deterministic and explainable. A machine-learning model may be added later, but it must not replace visible evidence.

The score is a prioritization signal, not a mathematical probability that a file is malware.

## Example score ranges

| Score | Level | Default behavior |
|---:|---|---|
| 0–24 | Low | Log quietly |
| 25–49 | Review | Show in review queue |
| 50–74 | High | Notify prominently |
| 75–100 | Critical | Quarantine when automatic protection is enabled |

Thresholds are configurable and stored as policy, not hard-coded into the UI.

## Example positive risk signals

| Signal | Example contribution |
|---|---:|
| Executable or installer | +15 |
| Script capable of executing commands | +20 |
| Dangerous extension such as `.scr` | +35 |
| Double extension such as `.pdf.exe` | +25 |
| Extension disagrees with file header | +25 |
| Invalid Authenticode signature | +35 |
| Unsigned executable | +15 |
| Newly registered or suspicious source domain | +15 |
| Source was already blocked by DNS policy | +30 |
| Filename uses excessive Unicode/confusable characters | +10 |
| Same filename previously had a different signer | +30 |
| Known trusted hash changed at the same path | +25 |
| Password-protected archive that cannot be inspected | +10 review signal |

## Example score reductions

| Signal | Example contribution |
|---|---:|
| Valid signature from explicitly trusted publisher | −25 |
| Hash explicitly trusted by the user | −40 |
| Source domain explicitly trusted | −10 |
| Non-executable document with consistent header | −10 |

Reductions cannot erase certain critical facts. For example, an invalid signature must not become “safe” merely because the domain is trusted.

## Example explanation

```text
Risk score: 82 / 100 — Critical

+25 Double extension: invoice.pdf.exe
+35 Digital signature is invalid
+15 Executable file type
+15 Source domain has no trusted history
-08 Download source matched a recent browser connection

Recommended action: Quarantine before opening
```

## Rule precedence

1. Explicit malicious hash or administrator deny rule.
2. File integrity failure or invalid signature policy.
3. Explicit trusted hash.
4. Trusted publisher policy.
5. File-type and filename rules.
6. Source reputation and DNS evidence.
7. Default score threshold.

Trusting a publisher does not automatically trust every filename, source, or future certificate. Policies must support expiration and revocation.

## Avoiding false positives

- Treat unknown as unknown, not malicious.
- Separate “cannot inspect” from “dangerous.”
- Do not punish large files simply for taking longer to hash.
- Do not analyze partial browser downloads as complete files.
- Recognize common signed installers and update packages without silently trusting all executables.
- Allow users to see precisely why a threshold was crossed.

