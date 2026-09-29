# 1. Problem, Goals, and Product Story

## The problem

DNS protection can stop a request to a known dangerous domain, but a downloaded file becomes a different security object. After the bytes reach the computer, DNS cannot answer important questions such as:

- What file was created?
- Was the file renamed?
- Is it executable?
- Is its publisher trusted?
- Does its digital signature remain valid?
- Did it come from the domain seen by the DNS filter?
- Has the file changed since it was downloaded?
- Was it opened, restored, quarantined, or deleted?

Browsers show a download list, and antivirus products may scan files, but the user often receives no coherent explanation connecting the website, network destination, file identity, and final action.

## Product goal

Build a local, explainable Download Guardian that gives the user one timeline:

> Website → network connection → downloaded file → identity analysis → risk decision → user action.

## Security goals

1. Detect new files in configured download locations.
2. Wait until a file is stable before hashing or moving it.
3. Record immutable identity evidence: SHA-256, size, timestamps, extension, signature, and publisher.
4. Detect misleading names such as `invoice.pdf.exe`.
5. Use DNS and connection history as supporting evidence when reliable correlation exists.
6. Calculate an explainable score from 0 to 100.
7. Isolate high-risk files without permanently deleting them.
8. Make restore and delete operations explicit, authenticated, and audited.
9. Keep analysis local unless the user explicitly enables an external reputation provider.
10. Remain useful when the internet is unavailable.

## Non-goals

Download Guardian is not initially intended to:

- Replace Microsoft Defender or a full antivirus engine.
- Guarantee that a low-risk file is safe.
- Decompile every executable.
- inspect encrypted archive contents without a password.
- attribute every file to an exact URL using filesystem events alone.
- upload files to a cloud scanner automatically.
- monitor every file on the entire disk by default.

## User story

1. The user downloads `DiscordSetup.exe`.
2. Download Guardian sees a new file appear in Downloads.
3. It waits until the browser finishes writing it.
4. It records the file hash and checks the Authenticode signature.
5. It determines that the file is executable, signed, and published by a recognized company.
6. It finds a recent connection to the matching source domain.
7. It produces a low or review-level score and explains why.
8. The desktop application shows the event immediately.
9. If the same filename later arrives with a different publisher or invalid signature, it receives a much higher score and can be quarantined.

## Success criteria

- No partially downloaded file is treated as complete.
- Every automated action has a human-readable reason.
- Quarantined files cannot be executed from their original path.
- Restoring a file requires an explicit decision.
- Restarting the service does not lose event history or quarantine state.
- Normal document downloads do not create excessive alerts.

