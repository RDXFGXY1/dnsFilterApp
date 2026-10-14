# Live Network Test

The Network Test page performs an explicit, on-demand connection-quality test.
It does not inspect applications, capture packet contents, or run automatically
in the background.

## Where to find it

Open **Monitor → Network Test** in the local web dashboard.

## Measurements

- **Ping** is measured with repeated TCP connection attempts to the test server
  on port 443. TCP latency remains useful when traditional ICMP echo is blocked.
- **Jitter** is the average change between consecutive successful latency samples.
- **Packet loss** is the percentage of failed connection attempts in the sample.
- **Download** is calculated while a bounded test file is received.
- **Upload** is calculated while a bounded generated payload is sent.

Every measurement and the graph update while the test is running.

## Test sizes

- **Quick test** uses approximately 5 MB for download and 2 MB for upload.
- **Full test** uses approximately 25 MB for download and 5 MB for upload.

The exact transferred amount may vary slightly because of HTTP framing. Tests are
manual and a running test can be cancelled at any time. Starting another test
cancels the previous one.

## External service and privacy

The current beta uses Cloudflare's public speed-test host. Running a test sends a
normal HTTPS request to that host, which can observe the public IP address and
standard connection metadata. DNS Filter does not send account details, browsing
history, local logs, or saved rules with the request.

The upload payload is generated in memory and contains no user files or personal
content. Results remain in memory and are not written to the database in this
version.

## Accuracy limits

This is a practical connection-health measurement, not a laboratory benchmark.
Results can change because of Wi-Fi quality, VPNs, proxies, firewall inspection,
other downloads, test-server load, and the route between the device and server.
TCP packet loss is inferred from failed connection attempts and is not identical
to raw ICMP packet-loss measurement.

The feature uses the same Go implementation on Windows 11 and Arch Linux.
