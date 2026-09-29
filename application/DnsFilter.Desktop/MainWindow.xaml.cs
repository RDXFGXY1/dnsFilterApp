using System.Net.Http;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Shapes;
using System.Windows.Threading;
using System.Windows.Interop;
using Forms = System.Windows.Forms;
using Color = System.Windows.Media.Color;
using MessageBox = System.Windows.MessageBox;
using Point = System.Windows.Point;

namespace DnsFilter.Desktop;

public partial class MainWindow : Window
{
    private ApiClient api = new();
    private bool filteringEnabled;
    private bool dnsBypassEnabled;
    private bool refreshing;
    private TimelinePoint[] timeline = [];
    private readonly DispatcherTimer liveTimer = new() { Interval = TimeSpan.FromSeconds(3) };
    private readonly Forms.NotifyIcon destinationNotifier = new();
    private readonly HashSet<string> knownDestinations = [];
    private bool destinationAlertsInitialized;

    public MainWindow()
    {
        InitializeComponent();
        var informationalVersion = typeof(MainWindow).Assembly
            .GetCustomAttribute<AssemblyInformationalVersionAttribute>()?.InformationalVersion;
        ProductVersionText.Text = $"Version {informationalVersion ?? "unknown"}";
        SourceInitialized += (_, _) => EnableDarkTitleBar();
        destinationNotifier.Icon = System.Drawing.SystemIcons.Shield;
        destinationNotifier.Text = "DNS Filter connection alerts";
        destinationNotifier.Visible = true;
        destinationNotifier.BalloonTipClicked += (_, _) => Dispatcher.Invoke(() => { Show(); Activate(); NewConnectionsSection.BringIntoView(); });
        liveTimer.Tick += async (_, _) => await RefreshAsync();
    }

    [DllImport("dwmapi.dll")]
    private static extern int DwmSetWindowAttribute(IntPtr hwnd, int attribute, ref int value, int size);

    private void EnableDarkTitleBar()
    {
        var enabled = 1;
        var handle = new WindowInteropHelper(this).Handle;
        _ = DwmSetWindowAttribute(handle, 20, ref enabled, sizeof(int));
        _ = DwmSetWindowAttribute(handle, 19, ref enabled, sizeof(int));
    }

    private async void Window_Loaded(object sender, RoutedEventArgs e)
    {
        if (await api.ServiceAvailableAsync()) { LoginError.Text = "Local service ready."; return; }
        if (!LocalService.TryStart(out var message)) { LoginError.Text = message; return; }
        LoginError.Foreground = System.Windows.Media.Brushes.LightGray;
        LoginError.Text = message;
        for (var attempt = 0; attempt < 12; attempt++)
        {
            await Task.Delay(500);
            if (await api.ServiceAvailableAsync()) { LoginError.Text = "Local service ready."; return; }
        }
        LoginError.Text = "The service did not become ready. Check the server logs and retry.";
    }

    private async void Login_Click(object sender, RoutedEventArgs e) => await LoginAsync();
    private async void PasswordBox_KeyDown(object sender, System.Windows.Input.KeyEventArgs e) { if (e.Key == Key.Enter) await LoginAsync(); }

    private async Task LoginAsync()
    {
        LoginButton.IsEnabled = false; LoginError.Text = "";
        try
        {
            var result = await api.LoginAsync(UsernameBox.Text.Trim(), PasswordBox.Password);
            if (!result.Success) throw new InvalidOperationException("Sign in failed.");
            LoginLayer.Visibility = Visibility.Collapsed; PasswordBox.Clear();
            ActivityText.Text = "Signed in securely.";
            if (!string.IsNullOrWhiteSpace(result.Recovery_Key)) ShowRecoveryKey(result.Recovery_Key);
            await RefreshAsync();
            liveTimer.Start();
        }
        catch (Exception ex) { LoginError.Text = Friendly(ex); }
        finally { LoginButton.IsEnabled = true; }
    }

    private void ShowRecoveryKey(string recoveryKey)
    {
        var code = new System.Windows.Controls.TextBox
        {
            Text = recoveryKey,
            IsReadOnly = true,
            TextWrapping = TextWrapping.Wrap,
            FontFamily = new System.Windows.Media.FontFamily("Consolas"),
            FontSize = 16,
            Padding = new Thickness(12),
            Margin = new Thickness(0, 14, 0, 14),
            Background = new SolidColorBrush(Color.FromRgb(9, 9, 9)),
            Foreground = System.Windows.Media.Brushes.White,
            BorderBrush = new SolidColorBrush(Color.FromRgb(57, 65, 80))
        };
        var saved = new System.Windows.Controls.CheckBox { Content = "I saved this key somewhere safe and separate.", Margin = new Thickness(0, 0, 0, 14) };
        var close = new System.Windows.Controls.Button { Content = "Continue", IsEnabled = false, MinWidth = 120 };
        var copy = new System.Windows.Controls.Button { Content = "Copy key", MinWidth = 110, Margin = new Thickness(0, 0, 8, 0) };
        var actions = new StackPanel { Orientation = System.Windows.Controls.Orientation.Horizontal, HorizontalAlignment = System.Windows.HorizontalAlignment.Right };
        actions.Children.Add(copy);
        actions.Children.Add(close);
        var content = new StackPanel { Margin = new Thickness(28) };
        content.Children.Add(new TextBlock { Text = "One-time account recovery key", FontSize = 22, FontWeight = FontWeights.SemiBold });
        content.Children.Add(new TextBlock { Text = "Save this key now. It will never be shown again and is required if you forget the dashboard password.", TextWrapping = TextWrapping.Wrap, Foreground = new SolidColorBrush(Color.FromRgb(174, 184, 201)), Margin = new Thickness(0, 8, 0, 0) });
        content.Children.Add(code);
        content.Children.Add(saved);
        content.Children.Add(actions);
        var dialog = new Window
        {
            Title = "Save your DNS Filter recovery key",
            Owner = this,
            Width = 520,
            Height = 330,
            ResizeMode = ResizeMode.NoResize,
            WindowStartupLocation = WindowStartupLocation.CenterOwner,
            Background = new SolidColorBrush(Color.FromRgb(10, 10, 10)),
            Foreground = System.Windows.Media.Brushes.White,
            Content = content
        };
        copy.Click += (_, _) => { System.Windows.Clipboard.SetText(recoveryKey); copy.Content = "Copied"; };
        saved.Checked += (_, _) => close.IsEnabled = true;
        saved.Unchecked += (_, _) => close.IsEnabled = false;
        close.Click += (_, _) => dialog.DialogResult = true;
        dialog.Closing += (_, e) => { if (dialog.DialogResult != true) e.Cancel = true; };
        dialog.ShowDialog();
    }

    private async Task RefreshAsync()
    {
        if (refreshing || LoginLayer.Visibility == Visibility.Visible) return;
        refreshing = true;
        try
        {
            var statsTask = api.StatsAsync(); var blockedTask = api.BlockedStatsAsync(); var topTask = api.TopBlockedAsync(); var timelineTask = api.TimelineAsync(); var recentTask = api.RecentAsync();
            var filterTask = api.FilteringAsync(); var dnsTask = api.DnsStatusAsync(); var freezeTask = api.FreezeAsync(); var healthTask = api.ProtectionHealthAsync(); var destinationTask = api.DestinationAlertsAsync(); var notificationTask = api.NotificationSettingsAsync(); var bypassTask = api.DnsBypassAsync(); var quarantineTask = api.QuarantineAsync(); var downloadsTask = api.DownloadsAsync(); var downloadAnalyticsTask = api.DownloadAnalyticsAsync();
            await Task.WhenAll(statsTask, blockedTask, topTask, timelineTask, recentTask, filterTask, dnsTask, freezeTask, healthTask, destinationTask, notificationTask, bypassTask, quarantineTask, downloadsTask, downloadAnalyticsTask);
            var stats = await statsTask; var blocked = await blockedTask; var filtering = await filterTask; var dns = await dnsTask; var freeze = await freezeTask;
            filteringEnabled = filtering.Enabled;
            FilteringValue.Text = filtering.Enabled ? "Active" : "Disabled"; FilteringHint.Text = filtering.Enabled ? "Evaluating DNS traffic" : "Rules are not being enforced";
            ToggleFilteringButton.Content = filtering.Enabled ? "Disable filtering" : "Enable filtering";
            DnsValue.Text = dns.Active ? "Protected" : "Review"; DnsHint.Text = dns.Error ?? $"{dns.Interface ?? "Adapter"} · {dns.Dns ?? "DNS unavailable"}";
            BlockedValue.Text = stats.Blocked_Domains.ToString("N0");
            Blocked24Value.Text = blocked.Total_Blocked.ToString("N0");
            Blocked24Hint.Text = $"{blocked.Unique_Domains:N0} domains · {blocked.Unique_Clients:N0} clients";
            FreezeStatusText.Text = freeze.Status.Active ? freeze.Status.Manual ? "Active until you stop it manually" : $"Active until {freeze.Status.Until:g}" : "Off — new connections are allowed";
            var top = await topTask; TopDomainsList.ItemsSource = top.Take(8).ToArray();
            timeline = await timelineTask; DrawTimeline();
            var recent = await recentTask; LiveLogGrid.ItemsSource = recent.OrderByDescending(x => x.Timestamp).ToArray();
            LiveCountText.Text = $"{recent.Length} recent events"; LastLiveRefreshText.Text = $"Updated {DateTime.Now:T}";
            var health = await healthTask; HealthScoreText.Text = $"{health.Score}/100"; HealthChecksList.ItemsSource = health.Checks;
            HealthQueriesText.Text = health.Queries.ToString("N0"); HealthBlockedText.Text = health.Blocked.ToString("N0"); HealthCacheText.Text = health.Cached.ToString("N0");
            var destinations = await destinationTask; UpdateDestinationNotifications(destinations, await notificationTask);
            var bypass = await bypassTask; dnsBypassEnabled = bypass.Enabled && bypass.Enforced; DnsBypassText.Text = dnsBypassEnabled ? "Enforced · plaintext DNS is blocked" : bypass.Enabled ? "Configured · firewall rule needs repair" : "Off · direct plaintext DNS is allowed"; DnsBypassButton.Content = dnsBypassEnabled ? "Disable kill switch" : "Enable kill switch";
            var quarantine = await quarantineTask; QuarantineText.Text = quarantine.Items.Length == 0 ? "No applications are isolated" : $"{quarantine.Items.Length} application{(quarantine.Items.Length == 1 ? "" : "s")} isolated";
            NewConnectionsCount.Text = $"{destinations.Count} alert{(destinations.Count == 1 ? "" : "s")}"; NewConnectionsList.ItemsSource = destinations.Connections.OrderByDescending(x => x.Risk).Take(5).ToArray();
            var downloads = await downloadsTask; var downloadAnalytics = await downloadAnalyticsTask;
            DownloadsFolderText.Text = $"Monitoring {downloads.Folder}"; DownloadsGrid.ItemsSource = downloads.Events;
            DownloadsTodayText.Text = downloadAnalytics.Today.ToString("N0"); DownloadsTotalText.Text = downloadAnalytics.Total.ToString("N0"); DownloadsExecutableText.Text = downloadAnalytics.Executables.ToString("N0"); DownloadsRiskText.Text = downloadAnalytics.High_Risk.ToString("N0"); DownloadsQuarantineText.Text = downloadAnalytics.Quarantined.ToString("N0");
            ServiceBadge.Text = "● Service connected"; ActivityText.Text = $"Last refreshed {DateTime.Now:T}. The advanced web dashboard remains available for detailed rules and logs.";
        }
        catch (UnauthorizedAccessException) { liveTimer.Stop(); LoginLayer.Visibility = Visibility.Visible; LoginError.Text = "Your session expired. Sign in again."; }
        catch (Exception ex) { ServiceBadge.Text = "● Service unavailable"; ActivityText.Text = Friendly(ex); }
        finally { refreshing = false; }
    }

    private void UpdateDestinationNotifications(DestinationAlertsResult result, NotificationSettingsResult settings)
    {
        var current = result.Connections ?? [];
        if (!destinationAlertsInitialized)
        {
            foreach (var item in current) knownDestinations.Add(item.Key);
            destinationAlertsInitialized = true;
            return;
        }
        var fresh = current.Where(item => knownDestinations.Add(item.Key)).ToArray();
        if (!settings.Enabled) return;
        var applications = settings.Applications ?? [];
        fresh = fresh.Where(item => item.Risk >= settings.Min_Risk &&
            (applications.Length == 0 || applications.Any(filter =>
                item.Application.Equals(filter, StringComparison.OrdinalIgnoreCase) ||
                item.Path.Equals(filter, StringComparison.OrdinalIgnoreCase) ||
                item.Path.EndsWith("\\" + filter, StringComparison.OrdinalIgnoreCase)))).ToArray();
        if (fresh.Length == 0) return;
        var first = fresh[0];
        destinationNotifier.BalloonTipTitle = fresh.Length == 1 ? $"New connection · {first.Application}" : $"{fresh.Length} new connections";
        destinationNotifier.BalloonTipText = fresh.Length == 1
            ? $"{first.Destination}\nRisk {first.Risk}/100 · {first.Reasons}"
            : $"Latest: {first.Application} → {first.Destination}";
        destinationNotifier.ShowBalloonTip(8000);
    }

    private async void InspectDomain_Click(object sender, RoutedEventArgs e) => await InspectDomainAsync();
    private async void DomainInput_KeyDown(object sender, System.Windows.Input.KeyEventArgs e) { if (e.Key == Key.Enter) await InspectDomainAsync(); }
    private async Task InspectDomainAsync()
    {
        var domain = DomainInput.Text.Trim();
        if (domain.Length == 0) { ActivityText.Text = "Enter a domain first."; return; }
        try
        {
            var result = await api.InspectDomainAsync(domain);
            DomainInput.Text = result.Domain; DecisionDomainText.Text = result.Domain;
            DecisionBadge.Text = result.Decision.ToUpperInvariant();
            DecisionBadge.Foreground = new SolidColorBrush(result.Blocked ? Color.FromRgb(242, 140, 148) : Color.FromRgb(130, 217, 166));
            DecisionReasonText.Text = $"Layer: {result.Layer} · Reason: {result.Reason} · Previous blocks: {result.Blocked_Visits}";
            DecisionRecommendationText.Text = result.Recommendation;
        }
        catch (Exception ex) { ActivityText.Text = Friendly(ex); }
    }

    private async void QuickBlock_Click(object sender, RoutedEventArgs e) => await AddDomainRuleAsync(true);
    private async void QuickAllow_Click(object sender, RoutedEventArgs e) => await AddDomainRuleAsync(false);
    private async Task AddDomainRuleAsync(bool block)
    {
        var domain = DomainInput.Text.Trim();
        if (domain.Length == 0) { ActivityText.Text = "Inspect or enter a domain first."; return; }
        var verb = block ? "block" : "allow";
        if (MessageBox.Show($"Add {domain} to the {verb} rules?", "Confirm rule", MessageBoxButton.YesNo, MessageBoxImage.Question) != MessageBoxResult.Yes) return;
        await RunAsync(async () => { if (block) await api.BlockDomainAsync(domain); else await api.AllowDomainAsync(domain); }, $"{domain} added to {verb} rules.");
        await InspectDomainAsync();
    }

    private void TimelineCanvas_SizeChanged(object sender, SizeChangedEventArgs e) => DrawTimeline();
    private void DrawTimeline()
    {
        TimelineCanvas.Children.Clear();
        if (timeline.Length == 0 || TimelineCanvas.ActualWidth < 20) return;
        var width = TimelineCanvas.ActualWidth; var height = TimelineCanvas.ActualHeight; var max = Math.Max(1, timeline.Max(x => x.Count));
        PeakText.Text = $"Peak {max:N0}/hour";
        for (var i = 1; i < 4; i++) { var y = height * i / 4; TimelineCanvas.Children.Add(new Line { X1 = 0, X2 = width, Y1 = y, Y2 = y, Stroke = new SolidColorBrush(Color.FromRgb(35, 40, 49)), StrokeThickness = 1 }); }
        var points = new PointCollection();
        for (var i = 0; i < timeline.Length; i++) points.Add(new Point(i * width / Math.Max(1, timeline.Length - 1), height - 14 - timeline[i].Count * (height - 30) / max));
        TimelineCanvas.Children.Add(new Polyline { Points = points, Stroke = new SolidColorBrush(Color.FromRgb(91, 126, 235)), StrokeThickness = 3, StrokeLineJoin = PenLineJoin.Round });
        foreach (var point in points) { var dot = new Ellipse { Width = 6, Height = 6, Fill = new SolidColorBrush(Color.FromRgb(145, 170, 250)) }; Canvas.SetLeft(dot, point.X - 3); Canvas.SetTop(dot, point.Y - 3); TimelineCanvas.Children.Add(dot); }
    }

    private async Task RunAsync(Func<Task> operation, string success)
    {
        try { IsEnabled = false; await operation(); ActivityText.Text = success; await RefreshAsync(); }
        catch (Exception ex) { ActivityText.Text = Friendly(ex); }
        finally { IsEnabled = true; }
    }

    private async void Refresh_Click(object sender, RoutedEventArgs e) => await RefreshAsync();
    private async void ToggleFiltering_Click(object sender, RoutedEventArgs e) => await RunAsync(async () => { await api.SetFilteringAsync(!filteringEnabled); }, "Filtering state changed.");
    private async void ClearCache_Click(object sender, RoutedEventArgs e) => await RunAsync(async () => { await api.ClearCacheAsync(); }, "DNS cache cleared.");
    private async void UpdateLists_Click(object sender, RoutedEventArgs e) => await RunAsync(async () => { await api.UpdateListsAsync(); }, "Blocklists updated.");
    private async void StartFreeze_Click(object sender, RoutedEventArgs e) => await RunAsync(async () => { await api.StartFreezeAsync(0); }, "Connection freeze enabled until manually stopped.");
    private async void StopFreeze_Click(object sender, RoutedEventArgs e) => await RunAsync(async () => { await api.StopFreezeAsync(); }, "Connection freeze stopped.");
    private async void ToggleDnsBypass_Click(object sender, RoutedEventArgs e)
    {
        var enable = !dnsBypassEnabled;
        if (enable && MessageBox.Show("Block direct plaintext DNS on port 53 for this computer?", "DNS Bypass Kill Switch", MessageBoxButton.YesNo, MessageBoxImage.Warning) != MessageBoxResult.Yes) return;
        await RunAsync(async () => { await api.SetDnsBypassAsync(enable); }, enable ? "DNS bypass protection enabled." : "DNS bypass protection disabled.");
    }
    private async void DnsControls_Click(object sender, RoutedEventArgs e)
    {
        var answer = MessageBox.Show("Yes: route Windows DNS through DNS Filter.\nNo: restore automatic DNS (DHCP).", "System DNS", MessageBoxButton.YesNoCancel, MessageBoxImage.Question);
        if (answer == MessageBoxResult.Yes) await RunAsync(async () => { await api.SetupDnsAsync(); }, "Windows DNS now uses the local filter.");
        if (answer == MessageBoxResult.No) await RunAsync(async () => { await api.ResetDnsAsync(); }, "Windows DNS restored to automatic configuration.");
    }
    private void Overview_Click(object sender, RoutedEventArgs e) => MainScroll.ScrollToTop();
    private void ProtectionControls_Click(object sender, RoutedEventArgs e) => ProtectionControlsSection.BringIntoView();
    private void Freeze_Click(object sender, RoutedEventArgs e) => FreezeSection.BringIntoView();
    private void HardSecurity_Click(object sender, RoutedEventArgs e) => HardSecuritySection.BringIntoView();
    private void Analytics_Click(object sender, RoutedEventArgs e) => AnalyticsSection.BringIntoView();
    private void NewConnections_Click(object sender, RoutedEventArgs e) => NewConnectionsSection.BringIntoView();
    private void Downloads_Click(object sender, RoutedEventArgs e) => DownloadsSection.BringIntoView();
    private void DownloadsGrid_SelectionChanged(object sender, SelectionChangedEventArgs e)
    {
        if (DownloadsGrid.SelectedItem is not DownloadEvent item) { QuarantineDownloadButton.IsEnabled = RestoreDownloadButton.IsEnabled = false; return; }
        DownloadDetailTitle.Text = $"{item.Filename} · Risk {item.Risk_Score}/100 ({item.Risk_Level})";
        DownloadDetailText.Text = $"{item.Path}\nSHA-256: {item.Sha256}\nSignature: {item.Signature_Status}{(string.IsNullOrWhiteSpace(item.Publisher) ? "" : " · " + item.Publisher)}\n{item.ReasonText}";
        QuarantineDownloadButton.IsEnabled = item.State != "quarantined" && item.State != "deleted";
        RestoreDownloadButton.IsEnabled = item.State == "quarantined";
    }
    private async void QuarantineDownload_Click(object sender, RoutedEventArgs e)
    {
        if (DownloadsGrid.SelectedItem is not DownloadEvent item) return;
        if (MessageBox.Show($"Move {item.Filename} into protected quarantine?", "Download Guardian", MessageBoxButton.YesNo, MessageBoxImage.Warning) != MessageBoxResult.Yes) return;
        await RunAsync(async () => { await api.QuarantineDownloadAsync(item.Id); }, $"{item.Filename} was quarantined and verified.");
    }
    private async void RestoreDownload_Click(object sender, RoutedEventArgs e)
    {
        if (DownloadsGrid.SelectedItem is not DownloadEvent item) return;
        if (MessageBox.Show($"Restore {item.Filename} to its original location?", "Download Guardian", MessageBoxButton.YesNo, MessageBoxImage.Question) != MessageBoxResult.Yes) return;
        await RunAsync(async () => { await api.RestoreDownloadAsync(item.Id); }, $"{item.Filename} was restored.");
    }
    private void SignOut_Click(object sender, RoutedEventArgs e) { liveTimer.Stop(); api.Dispose(); api = new ApiClient(); knownDestinations.Clear(); destinationAlertsInitialized = false; LoginLayer.Visibility = Visibility.Visible; PasswordBox.Clear(); }
    private static string Friendly(Exception ex) => ex is HttpRequestException ? "Cannot reach DNS Filter. Start the Go service, then retry." : ex.Message;
    protected override void OnClosed(EventArgs e) { liveTimer.Stop(); destinationNotifier.Visible = false; destinationNotifier.Dispose(); api.Dispose(); base.OnClosed(e); }
}
