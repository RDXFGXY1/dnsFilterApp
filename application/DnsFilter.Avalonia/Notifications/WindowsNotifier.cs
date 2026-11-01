using System.Diagnostics;
using System.Text;
using System.Text.Json;
using System.Xml.Linq;

namespace DnsFilter.Avalonia.Notifications;

internal sealed class WindowsNotifier
{
    internal async Task ShowAsync(string key, string title, string body, CancellationToken cancellationToken)
    {
        if (!OperatingSystem.IsWindows()) throw new PlatformNotSupportedException("Native deadline notifications are available on Windows.");
        var xml = BuildXml(title, body);
        var powershell = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.Windows), "System32", "WindowsPowerShell", "v1.0", "powershell.exe");
        var start = new ProcessStartInfo(powershell) { UseShellExecute = false, CreateNoWindow = true, RedirectStandardInput = true, RedirectStandardOutput = true, RedirectStandardError = true, StandardInputEncoding = new UTF8Encoding(false) };
        foreach (var argument in new[] { "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", Path.Combine(AppContext.BaseDirectory, "Notifications", "Show-Notification.ps1") }) start.ArgumentList.Add(argument);
        using var process = Process.Start(start) ?? throw new InvalidOperationException("Could not start the Windows notification provider.");
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        timeout.CancelAfter(TimeSpan.FromSeconds(15));
        var stdout = process.StandardOutput.ReadToEndAsync(); var stderr = process.StandardError.ReadToEndAsync();
        try
        {
            await process.StandardInput.WriteAsync(JsonSerializer.Serialize(new { xml, tag = key[..Math.Min(16, key.Length)] }));
            process.StandardInput.Close();
            await process.WaitForExitAsync(timeout.Token);
            var error = await stderr; await stdout;
            if (process.ExitCode != 0) throw new InvalidOperationException(string.IsNullOrWhiteSpace(error) ? "Windows rejected the notification." : error.Trim());
        }
        catch { if (!process.HasExited) process.Kill(true); throw; }
    }
    internal static string BuildXml(string title, string body) => new XElement("toast", new XAttribute("activationType", "protocol"), new XAttribute("launch", "dnsfilter-app://discipline"),
            new XElement("visual", new XElement("binding", new XAttribute("template", "ToastGeneric"),
                new XElement("image", new XAttribute("placement", "appLogoOverride"), new XAttribute("src", new Uri(WindowsIntegration.IconPath).AbsoluteUri), new XAttribute("alt", "DNS Filter")),
                new XElement("text", title), new XElement("text", body)))).ToString(SaveOptions.DisableFormatting);
}
