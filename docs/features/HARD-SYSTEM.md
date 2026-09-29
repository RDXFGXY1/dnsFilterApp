# Hard System browser content protection

Hard System adds a second protection layer after DNS. The DNS filter decides whether a domain is allowed; the browser extension then examines visible page text locally and assigns a risk score from 0 to 100.

## Decisions

- **0–39 — Allow:** no strong local risk signal was detected.
- **40–69 — Warn:** the page should be reviewed, but it is not automatically covered.
- **70–100 — Block:** the extension replaces the page with a local warning screen. The user can go back or allow the page once.

The score uses explicit-content phrases, density, title and description signals, domain patterns, suspicious instructions, and educational or medical context. It is an explainable heuristic, not a guarantee or a cloud AI verdict.

## Install the extension

1. Keep DNS Filter running on port 8080.
2. Open `chrome://extensions` in Chrome, or `brave://extensions` in Brave.
3. Enable **Developer mode**.
4. Select **Load unpacked**.
5. Choose the `extensions/hard-system` folder inside this project.
6. Open **Rules & Intelligence → AI decisions → Hard System** in the dashboard to see recent scores.

The extension popup allows protection to be disabled and the block threshold to be adjusted.

## Privacy and limits

- Analysis happens on this computer; no third-party service receives page content.
- Password fields, form values, cookies, and browser history are not collected.
- Raw page text and full URLs are not stored. The dashboard keeps only domain, score, action, category, reasons, and time for the latest 500 checks.
- The page must begin loading before a browser extension can inspect its visible text. DNS remains the first blocking layer.
- Content inside some frames, closed shadow roots, images, video, or dynamically loaded areas may not be visible.
- False positives and false negatives are possible. **Allow once** exists for recovery, and the threshold can be tuned in the extension popup.
