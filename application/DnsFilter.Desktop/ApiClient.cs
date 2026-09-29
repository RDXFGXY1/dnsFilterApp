using System.Net;
using System.Net.Http;
using System.Net.Http.Json;
using System.Text;
using System.Text.Json;

namespace DnsFilter.Desktop;

public sealed class ApiClient : IDisposable
{
    private readonly HttpClient client;
    private static readonly JsonSerializerOptions Json = new() { PropertyNameCaseInsensitive = true };

    public ApiClient(string baseUrl = "http://127.0.0.1:8080")
    {
        var cookies = new CookieContainer();
        client = new HttpClient(new HttpClientHandler { CookieContainer = cookies, UseCookies = true })
        {
            BaseAddress = new Uri(baseUrl.TrimEnd('/') + "/"),
            Timeout = TimeSpan.FromSeconds(15)
        };
    }

    public async Task<LoginResult> LoginAsync(string username, string password)
    {
        // The current Go login handler checks this media type literally.
        var json = JsonSerializer.Serialize(new { username, password });
        using var body = new StringContent(json, Encoding.UTF8);
        body.Headers.Remove("Content-Type");
        body.Headers.TryAddWithoutValidation("Content-Type", "application/json");
        using var response = await client.PostAsync("login", body);
        var result = await ReadAsync<LoginResult>(response);
        if (!string.IsNullOrWhiteSpace(result.Csrf_Token))
        {
            client.DefaultRequestHeaders.Remove("X-CSRF-Token");
            client.DefaultRequestHeaders.TryAddWithoutValidation("X-CSRF-Token", result.Csrf_Token);
        }
        return result;
    }

    public async Task<bool> ServiceAvailableAsync()
    {
        try { using var response = await client.GetAsync("login"); return response.IsSuccessStatusCode; }
        catch { return false; }
    }

    public Task<StatsResult> StatsAsync() => GetAsync<StatsResult>("api/stats");
    public Task<BlockedStats> BlockedStatsAsync() => GetAsync<BlockedStats>("api/stats/blocked");
    public Task<TopDomain[]> TopBlockedAsync() => GetAsync<TopDomain[]>("api/stats/top-blocked");
    public Task<TimelinePoint[]> TimelineAsync() => GetAsync<TimelinePoint[]>("api/stats/timeline");
    public Task<RecentBlocked[]> RecentAsync() => GetAsync<RecentBlocked[]>("api/recent");
    public Task<ProtectionHealth> ProtectionHealthAsync() => GetAsync<ProtectionHealth>("api/protection/health");
    public Task<DecisionResult> InspectDomainAsync(string domain) => GetAsync<DecisionResult>("api/decision/inspect?domain=" + Uri.EscapeDataString(domain));
    public Task<FilteringResult> FilteringAsync() => GetAsync<FilteringResult>("api/filtering");
    public Task<DnsStatus> DnsStatusAsync() => GetAsync<DnsStatus>("api/system/dns/status");
    public Task<FreezeResult> FreezeAsync() => GetAsync<FreezeResult>("api/connection-freeze");
    public Task<DestinationAlertsResult> DestinationAlertsAsync() => GetAsync<DestinationAlertsResult>("api/network-intelligence/destination-alerts");
    public Task<NotificationSettingsResult> NotificationSettingsAsync() => GetAsync<NotificationSettingsResult>("api/network-intelligence/notifications");
    public Task<DnsBypassStatus> DnsBypassAsync() => GetAsync<DnsBypassStatus>("api/dns-bypass");
    public Task<QuarantineResult> QuarantineAsync() => GetAsync<QuarantineResult>("api/app-network/quarantine");
    public Task<DownloadEventsResult> DownloadsAsync() => GetAsync<DownloadEventsResult>("api/downloads?limit=150");
    public Task<DownloadAnalytics> DownloadAnalyticsAsync() => GetAsync<DownloadAnalytics>("api/downloads/analytics");
    public Task<ApiResult> ClearCacheAsync() => PostAsync<ApiResult>("api/system/clear-cache", new { });
    public Task<ApiResult> UpdateListsAsync() => PostAsync<ApiResult>("api/blocklist/update", new { });
    public Task<ApiResult> SetFilteringAsync(bool enabled) => PostAsync<ApiResult>("api/filtering", new { enabled });
    public Task<ApiResult> SetupDnsAsync() => PostAsync<ApiResult>("api/system/dns/setup", new { });
    public Task<ApiResult> ResetDnsAsync() => PostAsync<ApiResult>("api/system/dns/reset", new { });
    public Task<ApiResult> StartFreezeAsync(int minutes) => PostAsync<ApiResult>("api/connection-freeze/start", new { minutes });
    public Task<ApiResult> StopFreezeAsync() => PostAsync<ApiResult>("api/connection-freeze/stop", new { });
    public Task<ApiResult> BlockDomainAsync(string domain) => PostAsync<ApiResult>("api/custom-blocklist/add", new { domain });
    public Task<ApiResult> AllowDomainAsync(string domain) => PostAsync<ApiResult>("api/whitelist", new { domain });
    public Task<DnsBypassStatus> SetDnsBypassAsync(bool enabled) => PostAsync<DnsBypassStatus>("api/dns-bypass", new { enabled });
    public Task<ApiResult> QuarantineDownloadAsync(long id) => PostAsync<ApiResult>($"api/downloads/{id}/quarantine", new { });
    public Task<ApiResult> RestoreDownloadAsync(long id) => PostAsync<ApiResult>($"api/downloads/{id}/restore", new { });

    private async Task<T> GetAsync<T>(string path)
    {
        using var response = await client.GetAsync(path);
        return await ReadAsync<T>(response);
    }

    private async Task<T> PostAsync<T>(string path, object body)
    {
        using var response = await client.PostAsJsonAsync(path, body);
        return await ReadAsync<T>(response);
    }

    private static async Task<T> ReadAsync<T>(HttpResponseMessage response)
    {
        var text = await response.Content.ReadAsStringAsync();
        if (!response.IsSuccessStatusCode)
        {
            var error = TryError(text) ?? $"Request failed ({(int)response.StatusCode})";
            if (response.StatusCode == HttpStatusCode.Unauthorized) throw new UnauthorizedAccessException(error);
            throw new InvalidOperationException(error);
        }
        try
        {
            return JsonSerializer.Deserialize<T>(text, Json) ?? throw new InvalidOperationException("The service returned an empty response.");
        }
        catch (JsonException)
        {
            throw new InvalidOperationException("The local service returned an unexpected response. Update or restart the Go service, then retry.");
        }
    }

    private static string? TryError(string text)
    {
        try { return JsonDocument.Parse(text).RootElement.TryGetProperty("error", out var e) ? e.GetString() : null; }
        catch { return null; }
    }

    public void Dispose() => client.Dispose();
}

public record LoginResult(bool Success, string? Recovery_Key, string? Recovery_Warning, string? Csrf_Token);
public record ApiResult(bool Success, string? Error = null, string? Hint = null);
public record FilteringResult(bool Enabled, long Paused_Until);
public record DnsStatus(bool Active, string? Interface, string? Dns, string? Error);
public record StatsResult(long Blocked_Domains, JsonElement Stats, long Timestamp);
public record BlockedStats(long Total_Blocked, long Unique_Domains, long Unique_Clients);
public record TopDomain(string Domain, int Count);
public record TimelinePoint(string Label, int Count);
public record RecentBlocked(string Domain, string Client_Ip, long Timestamp)
{
    public DateTime LocalTime => DateTimeOffset.FromUnixTimeSeconds(Timestamp).LocalDateTime;
    public string TimeText => LocalTime.ToString("HH:mm:ss");
}
public record HealthCheck(string Id, string Label, bool Ok, string Detail)
{
    public string StateIcon => Ok ? "✓" : "!";
    public string StateColor => Ok ? "#82D9A6" : "#F2B86B";
}
public record ProtectionHealth(int Score, HealthCheck[] Checks, bool Enabled, ulong Queries, ulong Blocked, ulong Cached, string? Interface, string? Dns, string? Route_Error);
public record DecisionResult(string Domain, string Decision, bool Blocked, string Reason, string Layer, string Risk, string Recommendation, int Blocked_Visits, string? Last_Blocked);
public record FreezeStatus(bool Active, bool Manual, DateTimeOffset? Until, string? Description);
public record FreezeResult(FreezeStatus Status, JsonElement Events);
public record DestinationConnection(long Id, string Application, string Path, string Remote_Ip, int Remote_Port, string Protocol, int Risk, string Reasons, DateTimeOffset Last_Seen, int Seen, string? Hostname)
{
    public string Destination => string.IsNullOrWhiteSpace(Hostname) ? $"{Remote_Ip}:{Remote_Port}" : $"{Hostname}:{Remote_Port}";
    public string Key => $"{Path}|{Remote_Ip}|{Remote_Port}|{Protocol}";
}
public record DestinationAlertsResult(DestinationConnection[] Connections, int Count);
public record NotificationSettingsResult(bool Enabled, int Min_Risk, string[] Applications);
public record DnsBypassStatus(bool Supported, bool Enabled, bool Enforced, string[] Protected_Ports, string? Updated_At);
public record QuarantineEntry(string Path, string Name, string Reason, int Risk, string Quarantined_At);
public record QuarantineResult(QuarantineEntry[] Items, bool Supported);
public record DownloadEvent(long Id, string Path, string Filename, string Sha256, long Size_Bytes, string Extension, string Detected_Type, string Signature_Status, string Publisher, int Risk_Score, string Risk_Level, string[] Reasons, string State, DateTimeOffset Created_At, DateTimeOffset Updated_At)
{
    public string SizeText => Size_Bytes >= 1_048_576 ? $"{Size_Bytes / 1_048_576d:F1} MB" : $"{Math.Max(1, Size_Bytes / 1024d):F0} KB";
    public string ReasonText => string.Join(" · ", Reasons ?? []);
    public string TimeText => Created_At.LocalDateTime.ToString("g");
}
public record DownloadEventsResult(string Folder, DownloadEvent[] Events, int Count);
public record DownloadAnalytics(int Total, int Today, int Executables, int High_Risk, int Quarantined, long Total_Bytes);
