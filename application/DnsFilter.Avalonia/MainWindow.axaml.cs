using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.Interactivity;
using Avalonia.Threading;
using DnsFilter.Desktop;

namespace DnsFilter.Avalonia;

public partial class MainWindow : Window
{
    private ApiClient api = new();
    private readonly DispatcherTimer refreshTimer = new() { Interval = TimeSpan.FromSeconds(3) };
    private bool filteringEnabled;
    private bool dnsBypassEnabled;
    private bool refreshing;
    private readonly bool isWindows = OperatingSystem.IsWindows();
    private readonly bool isLinux = OperatingSystem.IsLinux();

    public MainWindow()
    {
        InitializeComponent();
        PlatformText.Text = isWindows ? "Windows · native controls" : isLinux ? "Linux · shared core" : "Cross-platform client";
        CapabilitySummary.Text = isWindows
            ? "Windows Firewall, WFP, process telemetry, and Authenticode integrations are available."
            : "DNS filtering, dashboard analytics, rules, vault, and productivity features are available. Linux firewall and process telemetry are the next backend milestone.";
        FreezeSupportText.Text = isWindows ? "WINDOWS CONTROL READY" : "LINUX BACKEND PENDING";
        DnsBypassSupportText.Text = isWindows ? "WINDOWS FIREWALL READY" : "NFTABLES BACKEND PENDING";
        StartFreezeButton.IsEnabled = isWindows;
        StopFreezeButton.IsEnabled = isWindows;
        DnsBypassButton.IsEnabled = isWindows;
        refreshTimer.Tick += async (_, _) => await RefreshAsync();
        Opened += async (_, _) => await ProbeServiceAsync();
        Closed += (_, _) => api.Dispose();
    }

    private async Task ProbeServiceAsync()
    {
        var ready = await api.ServiceAvailableAsync();
        ServiceBadge.Text = ready ? "● Service ready" : "● Service offline";
        LoginError.Text = ready ? "Local service ready." : isLinux
            ? "Start the Go service with root permission, then try again."
            : "Start the DNS Filter Go service as administrator, then try again.";
    }

    private async void Login_Click(object? sender, RoutedEventArgs e) => await LoginAsync();
    private async void PasswordBox_KeyDown(object? sender, KeyEventArgs e) { if (e.Key == Key.Enter) await LoginAsync(); }

    private async Task LoginAsync()
    {
        LoginButton.IsEnabled = false;
        LoginError.Text = "";
        try
        {
            var result = await api.LoginAsync(UsernameBox.Text?.Trim() ?? "", PasswordBox.Text ?? "");
            if (!result.Success) throw new InvalidOperationException("Sign in failed.");
            LoginLayer.IsVisible = false;
            PasswordBox.Text = "";
            await RefreshAsync();
            refreshTimer.Start();
        }
        catch (Exception ex) { LoginError.Text = Friendly(ex); }
        finally { LoginButton.IsEnabled = true; }
    }

    private async Task RefreshAsync()
    {
        if (refreshing || LoginLayer.IsVisible) return;
        refreshing = true;
        try
        {
            var blockedTask = api.BlockedStatsAsync();
            var topTask = api.TopBlockedAsync();
            var recentTask = api.RecentAsync();
            var filteringTask = api.FilteringAsync();
            var dnsTask = api.DnsStatusAsync();
            var healthTask = api.ProtectionHealthAsync();
            var freezeTask = api.FreezeAsync();
            await Task.WhenAll(blockedTask, topTask, recentTask, filteringTask, dnsTask, healthTask, freezeTask);

            var blocked = await blockedTask;
            var filtering = await filteringTask;
            var dns = await dnsTask;
            var health = await healthTask;
            var freeze = await freezeTask;
            filteringEnabled = filtering.Enabled;
            FilteringValue.Text = filtering.Enabled ? "Active" : "Disabled";
            FilteringHint.Text = filtering.Enabled ? "Evaluating DNS traffic" : "Rules are not being enforced";
            ToggleFilteringButton.Content = filtering.Enabled ? "Disable filtering" : "Enable filtering";
            DnsValue.Text = dns.Active ? "Protected" : "Review";
            DnsHint.Text = dns.Error ?? $"{dns.Interface ?? "Interface"} · {dns.Dns ?? "DNS unavailable"}";
            Blocked24Value.Text = blocked.Total_Blocked.ToString("N0");
            Blocked24Hint.Text = $"{blocked.Unique_Domains:N0} domains · {blocked.Unique_Clients:N0} clients";
            HealthScoreText.Text = $"{health.Score}/100";
            HealthChecksList.ItemsSource = health.Checks;
            var recent = (await recentTask).OrderByDescending(x => x.Timestamp).ToArray();
            RecentGrid.ItemsSource = recent.Take(50).ToArray();
            LiveGrid.ItemsSource = recent;
            LiveStatusText.Text = $"{recent.Length} recent events · updated {DateTime.Now:T}";
            TopDomainsList.ItemsSource = (await topTask).Take(10).ToArray();
            FreezeStatusText.Text = freeze.Status.Active
                ? (freeze.Status.Manual ? "Active until manually stopped" : $"Active until {freeze.Status.Until:g}")
                : "Off — new connections are allowed";

            if (isWindows)
            {
                var bypass = await api.DnsBypassAsync();
                dnsBypassEnabled = bypass.Enabled && bypass.Enforced;
                DnsBypassText.Text = dnsBypassEnabled ? "Enforced · direct plaintext DNS is blocked" : "Off · applications may bypass the local resolver";
                DnsBypassButton.Content = dnsBypassEnabled ? "Disable protection" : "Enable protection";
            }
            else
            {
                DnsBypassText.Text = "The shared UI is ready; native nftables enforcement has not been connected yet.";
            }
            ServiceBadge.Text = "● Service connected";
        }
        catch (UnauthorizedAccessException)
        {
            refreshTimer.Stop();
            LoginLayer.IsVisible = true;
            LoginError.Text = "Session expired. Sign in again.";
        }
        catch (Exception ex)
        {
            ServiceBadge.Text = "● Service unavailable";
            LiveStatusText.Text = Friendly(ex);
        }
        finally { refreshing = false; }
    }

    private void ShowSection(Control section, string kicker, string title)
    {
        OverviewSection.IsVisible = section == OverviewSection;
        LiveSection.IsVisible = section == LiveSection;
        ControlsSection.IsVisible = section == ControlsSection;
        FreezeSection.IsVisible = section == FreezeSection;
        SecuritySection.IsVisible = section == SecuritySection;
        PageKicker.Text = kicker;
        PageTitle.Text = title;
    }

    private void Overview_Click(object? s, RoutedEventArgs e) => ShowSection(OverviewSection, "PROTECTION OVERVIEW", "Your network at a glance");
    private void Live_Click(object? s, RoutedEventArgs e) => ShowSection(LiveSection, "LIVE MONITOR", "DNS activity as it happens");
    private void Controls_Click(object? s, RoutedEventArgs e) => ShowSection(ControlsSection, "DNS CONTROLS", "Filtering and local routing");
    private void Freeze_Click(object? s, RoutedEventArgs e) => ShowSection(FreezeSection, "CONNECTION CONTROL", "Freeze new outbound connections");
    private void Security_Click(object? s, RoutedEventArgs e) => ShowSection(SecuritySection, "HARD SECURITY", "Platform protection controls");
    private async void Refresh_Click(object? s, RoutedEventArgs e) => await RefreshAsync();
    private async void ToggleFiltering_Click(object? s, RoutedEventArgs e) => await RunAsync(() => api.SetFilteringAsync(!filteringEnabled));
    private async void UpdateLists_Click(object? s, RoutedEventArgs e) => await RunAsync(api.UpdateListsAsync);
    private async void ClearCache_Click(object? s, RoutedEventArgs e) => await RunAsync(api.ClearCacheAsync);
    private async void SetupDns_Click(object? s, RoutedEventArgs e) => await RunAsync(api.SetupDnsAsync);
    private async void ResetDns_Click(object? s, RoutedEventArgs e) => await RunAsync(api.ResetDnsAsync);
    private async void StartFreeze_Click(object? s, RoutedEventArgs e) => await RunAsync(() => api.StartFreezeAsync(0));
    private async void StopFreeze_Click(object? s, RoutedEventArgs e) => await RunAsync(api.StopFreezeAsync);
    private async void ToggleDnsBypass_Click(object? s, RoutedEventArgs e) => await RunAsync(() => api.SetDnsBypassAsync(!dnsBypassEnabled));

    private async Task RunAsync<T>(Func<Task<T>> action)
    {
        try { await action(); await RefreshAsync(); }
        catch (Exception ex) { LiveStatusText.Text = Friendly(ex); }
    }

    private void SignOut_Click(object? s, RoutedEventArgs e)
    {
        refreshTimer.Stop();
        api.Dispose();
        api = new ApiClient();
        LoginLayer.IsVisible = true;
        PasswordBox.Text = "";
        LoginError.Text = "Signed out.";
    }

    private static string Friendly(Exception ex) => ex.InnerException?.Message ?? ex.Message;
}
