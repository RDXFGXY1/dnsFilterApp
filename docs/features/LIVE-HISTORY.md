# Live DNS history and health

The dashboard's Live view keeps up to the latest 200 DNS decisions in the local
database, so reloading the page does not start the list from zero. The dashboard
also summarizes the top 10 blocked domains from the last 24 hours and shows a
timeline of service and DNS health events. The top-domain summary reads the
separate blocked-query log; it is not a packet capture.

Live privacy settings let you choose a retention period of 1–720 hours, hide
client IP addresses in saved Live entries, or keep only blocked entries. New
settings update existing saved Live history. The Vault must be unlocked to
change these settings. The bounded 200-entry history can be shorter than the
chosen retention period on a busy machine.

These options concern the Live history view; they do not erase separate security
or audit records. DNS decisions are still evaluated normally when Live storage
is reduced.
