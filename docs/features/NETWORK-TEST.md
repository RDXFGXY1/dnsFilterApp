# Network Test

Open **Monitor → Network Test** in the local dashboard. Tests run only when you
start them. They do not inspect applications or capture packet contents.

## What it measures

- **Ping:** repeated TCP connections to the test server on port 443.
- **Jitter:** variation between successful latency samples.
- **Packet loss:** failed TCP connection attempts in the sample, not raw ICMP
  packet loss.
- **Download and upload:** transfer speed during the selected test.

The page shows each metric and its progress separately. A running test can be
cancelled; starting a new test cancels the previous one.

## Choose a test

| Mode | Transfer or duration |
|---|---|
| Quick | About 5 MB down and 2 MB up |
| Full | About 25 MB down and 5 MB up |
| Custom data | 5–500 MB total, split about 80% download and 20% upload |
| Custom time | 5–60 seconds, split between download and upload; data use depends on speed |

The selected MB value is a **data budget**, not a speed cap. HTTP framing can
make the exact amount vary slightly. A timed test has no fixed data budget.

## Saved results and privacy

The latest 30 results are stored in the local database and remain available
after a page reload or service restart. The page compares results from the same
test mode. A test contacts Cloudflare's public speed-test host over HTTPS; that
host can observe the public IP address and normal connection metadata. DNS
Filter does not include account details, browsing history, local logs, or saved
rules in the request. The generated upload payload contains no user files.

## Interpreting results

Results vary with Wi-Fi, VPNs, proxies, firewall inspection, other transfers,
the test server, and the route to it. This is a practical connection check,
not a laboratory benchmark. The same Go test implementation is used on Windows
and Arch Linux.
