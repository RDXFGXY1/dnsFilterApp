using Avalonia;
using Avalonia.Controls;
using Avalonia.Controls.ApplicationLifetimes;
using Avalonia.Markup.Xaml;
using Avalonia.Threading;
using DnsFilter.Avalonia.Notifications;

namespace DnsFilter.Avalonia;

public partial class App : Application
{
    private TrayIcon? tray;
    private DeadlineMonitor? monitor;
    private RegisteredWaitHandle? showWait;
    private bool exitRequested;
    public override void Initialize() => AvaloniaXamlLoader.Load(this);
    public override void OnFrameworkInitializationCompleted()
    {
        if (ApplicationLifetime is IClassicDesktopStyleApplicationLifetime desktop)
        {
            var window = new MainWindow();
            if (!OperatingSystem.IsWindows()) desktop.MainWindow = window;
            else
            {
                desktop.ShutdownMode = ShutdownMode.OnExplicitShutdown;
                var state = NotificationState.Load();
                var notifier = new WindowsNotifier();
                var menu = new NativeMenu();
                var open = new NativeMenuItem("Open DNS Filter");
                var discipline = new NativeMenuItem("Open tasks and reminders");
                var notifications = new NativeMenuItem("Deadline notifications") { ToggleType = NativeMenuItemToggleType.CheckBox, IsChecked = state.Enabled };
                var startup = new NativeMenuItem("Start tray with Windows") { ToggleType = NativeMenuItemToggleType.CheckBox, IsChecked = state.StartWithWindows };
                var test = new NativeMenuItem("Send a test notification");
                var status = new NativeMenuItem("Preparing notifications…") { IsEnabled = false };
                var exit = new NativeMenuItem("Exit desktop companion");
                void SetStatus(string text)
                {
                    status.Header = text.Length > 160 ? text[..160] : text;
                    window.SetNotificationStatus(text);
                    try
                    {
                        var directory = Path.GetDirectoryName(NotificationState.FilePath)!;
                        Directory.CreateDirectory(directory);
                        File.WriteAllText(Path.Combine(directory, "notification-status.txt"), text);
                    }
                    catch (IOException) { }
                    catch (UnauthorizedAccessException) { }
                }
                async Task SendTest()
                {
                    try { await notifier.ShowAsync("test-notification", "DNS Filter", "Windows deadline notifications are ready.", CancellationToken.None); SetStatus("Test notification sent"); }
                    catch (Exception error) { SetStatus(error.Message); }
                }
                void ShowWindow()
                {
                    desktop.MainWindow = window; window.Show();
                    if (window.WindowState == WindowState.Minimized) window.WindowState = WindowState.Normal;
                    window.Activate();
                }
                open.Click += (_, _) => ShowWindow();
                discipline.Click += (_, _) => { try { WindowsIntegration.OpenDiscipline(); } catch (Exception error) { status.Header = error.Message; } };
                notifications.Click += (_, _) => { state.Enabled = !state.Enabled; notifications.IsChecked = state.Enabled; try { state.Save(); } catch (Exception error) { status.Header = error.Message; } };
                startup.Click += (_, _) =>
                {
                    try { var enabled = !state.StartWithWindows; WindowsIntegration.SetStartsWithWindows(enabled); state.StartWithWindows = enabled; startup.IsChecked = enabled; state.Save(); }
                    catch (Exception error) { status.Header = error.Message; startup.IsChecked = WindowsIntegration.StartsWithWindows(); }
                };
                test.Click += async (_, _) => await SendTest();
                window.TestNotificationRequested += async () => await SendTest();
                exit.Click += (_, _) => { exitRequested = true; desktop.Shutdown(); };
                foreach (var item in new NativeMenuItemBase[] { open, discipline, new NativeMenuItemSeparator(), notifications, startup, test, status, new NativeMenuItemSeparator(), exit }) menu.Items.Add(item);
                tray = new TrayIcon { Icon = new WindowIcon(Path.Combine(AppContext.BaseDirectory, "Assets", "dns-filter.ico")), ToolTipText = "DNS Filter — protection and deadlines", Menu = menu, IsVisible = true };
                TrayIcon.SetIcons(this, new TrayIcons { tray });
                tray.Clicked += (_, _) => ShowWindow();
                window.Closing += (_, e) =>
                {
                    if (!exitRequested && e.CloseReason == WindowCloseReason.WindowClosing) { e.Cancel = true; window.Hide(); }
                };
                desktop.Exit += (_, _) => { monitor?.Dispose(); showWait?.Unregister(null); tray?.Dispose(); };
                try
                {
                    WindowsIntegration.RegisterIdentity();
                    WindowsIntegration.SetStartsWithWindows(state.StartWithWindows);
                    SetStatus("Deadline notifications are ready");
                    monitor = new DeadlineMonitor(state, notifier);
                    monitor.StatusChanged += text => Dispatcher.UIThread.Post(() => SetStatus(text));
                    monitor.Start();
                }
                catch (Exception error) { SetStatus(error.Message); }
                if (Program.ShowRequest is not null)
                    showWait = ThreadPool.RegisterWaitForSingleObject(Program.ShowRequest, (_, _) => Dispatcher.UIThread.Post(ShowWindow), null, Timeout.Infinite, false);
                if (!(desktop.Args?.Contains("--tray") ?? false)) desktop.MainWindow = window;
                if (desktop.Args?.Contains("--open-discipline") ?? false) { try { WindowsIntegration.OpenDiscipline(); } catch { } }
                if (desktop.Args?.Contains("--test-notification") ?? false)
                    Dispatcher.UIThread.Post(async () => await SendTest());
            }
        }
        base.OnFrameworkInitializationCompleted();
    }
}
