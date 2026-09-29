# Windows Lifecycle Resilience

When the user explicitly enables **System DNS** from DNS Filter, the application saves `server.manage_system_dns: true`. DNS Route Watch checks the active default-route adapter every 30 seconds. If Windows changes adapters, resumes with DHCP DNS, or another local event replaces the resolver, the watcher first verifies that the local DNS listener is healthy and then routes the new active adapter back through the filter.

This behavior is opt-in. A fresh installation and a user who selected **Restore automatic DNS** keep `manage_system_dns: false`; Route Watch records changes but does not modify the adapter. Disabling System DNS turns auto-repair off before resetting the adapter, preventing the watcher from undoing the user's choice.

For a one-time upgrade from an older configuration that has no routing-preference field, the application adopts automatic repair only when Windows is already using loopback DNS (`127.0.0.1`/`::1`). An explicit `manage_system_dns: false` is never overridden.

The route event history records the interface, resolver state, protection state, error, and change time. The watcher is closed during graceful application shutdown.

Automated tests simulate an adapter changing from an external resolver to the local resolver and verify both enabled and disabled behavior without changing the real machine's network. A final release test should still perform an actual restart, sleep/resume, Wi-Fi reconnect, and Ethernet/Wi-Fi switch on a Windows test machine.
