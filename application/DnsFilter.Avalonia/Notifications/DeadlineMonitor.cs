using System.Diagnostics;
using System.Text.Json;

namespace DnsFilter.Avalonia.Notifications;

internal sealed class DeadlineMonitor : IDisposable
{
    private readonly NotificationState state;
    private readonly WindowsNotifier notifier;
    private readonly CancellationTokenSource stop = new();
    private readonly SemaphoreSlim gate = new(1, 1);
    internal event Action<string>? StatusChanged;
    internal DeadlineMonitor(NotificationState state, WindowsNotifier notifier) { this.state = state; this.notifier = notifier; }
    internal void Start() => _ = RunAsync();
    private async Task RunAsync()
    {
        while (!stop.IsCancellationRequested)
        {
            try { if (state.Enabled) await CheckAsync(); }
            catch (OperationCanceledException) when (stop.IsCancellationRequested) { return; }
            catch (Exception error) { StatusChanged?.Invoke(error.Message); }
            try { await Task.Delay(TimeSpan.FromSeconds(20), stop.Token); }
            catch (OperationCanceledException) { return; }
        }
    }
    private async Task CheckAsync()
    {
        if (!await gate.WaitAsync(0, stop.Token)) return;
        try
        {
            var root = FindRoot() ?? throw new InvalidOperationException("Cannot find DNS Filter configuration. Keep the desktop app in its installation folder.");
            var helper = Path.Combine(AppContext.BaseDirectory, "dns-filter-notifications.exe");
            if (!File.Exists(helper)) throw new InvalidOperationException("The deadline companion is missing. Update the desktop package.");
            var start = new ProcessStartInfo(helper) { UseShellExecute = false, CreateNoWindow = true, RedirectStandardOutput = true, RedirectStandardError = true };
            start.ArgumentList.Add("-root"); start.ArgumentList.Add(root);
            using var process = Process.Start(start) ?? throw new InvalidOperationException("Could not read deadline data.");
            using var timeout = CancellationTokenSource.CreateLinkedTokenSource(stop.Token); timeout.CancelAfter(TimeSpan.FromSeconds(15));
            var stdout = process.StandardOutput.ReadToEndAsync(); var stderr = process.StandardError.ReadToEndAsync();
            try { await process.WaitForExitAsync(timeout.Token); }
            catch { if (!process.HasExited) process.Kill(true); throw; }
            var raw = await stdout; var error = await stderr;
            if (process.ExitCode != 0) throw new InvalidOperationException(error.Trim());
            var deadlines = JsonSerializer.Deserialize<DeadlineItem[]>(raw) ?? Array.Empty<DeadlineItem>();
            var delivered = 0;
            foreach (var item in deadlines)
            {
                if (!state.Enabled || stop.IsCancellationRequested) break;
                if (state.WasDelivered(item.Key)) continue;
                // Avoid an alert flood when the computer wakes after several hours.
                if (delivered >= 3) break;
                await notifier.ShowAsync(item.Key, item.Title, item.Body, stop.Token);
                state.MarkDelivered(item.Key); delivered++;
            }
            StatusChanged?.Invoke("Deadline notifications are ready");
        }
        finally { gate.Release(); }
    }
    internal static string? FindRoot()
    {
        var directory = AppContext.BaseDirectory;
        for (var level = 0; level < 9; level++)
        {
            if (File.Exists(Path.Combine(directory, "configs", "config.yaml"))) return directory;
            var parent = Directory.GetParent(directory); if (parent is null) break; directory = parent.FullName;
        }
        return null;
    }
    public void Dispose() => stop.Cancel();
}
