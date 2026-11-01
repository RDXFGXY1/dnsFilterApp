using Avalonia;
using DnsFilter.Avalonia.Notifications;

namespace DnsFilter.Avalonia;

class Program
{
    internal static EventWaitHandle? ShowRequest;
    [STAThread]
    public static void Main(string[] args)
    {
        if (!OperatingSystem.IsWindows()) { BuildAvaloniaApp().StartWithClassicDesktopLifetime(args); return; }
        // One tray companion per Windows user. Toast activation must not create duplicates.
        var name = "Local\\DNSFilter.Desktop." + Environment.UserName;
        using var mutex = new Mutex(true, name, out var first);
        if (!first)
        {
            if (args.Contains("--open-discipline")) { try { WindowsIntegration.OpenDiscipline(); } catch { } }
            else { try { using var signal = EventWaitHandle.OpenExisting(name + ".Show"); signal.Set(); } catch (WaitHandleCannotBeOpenedException) { } }
            return;
        }
        using var show = new EventWaitHandle(false, EventResetMode.AutoReset, name + ".Show");
        ShowRequest = show;
        try { BuildAvaloniaApp().StartWithClassicDesktopLifetime(args); }
        finally { ShowRequest = null; mutex.ReleaseMutex(); }
    }
    public static AppBuilder BuildAvaloniaApp() => AppBuilder.Configure<App>().UsePlatformDetect().WithInterFont().LogToTrace();
}
