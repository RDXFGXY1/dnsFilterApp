using System.Text.Json;
using System.Text.Json.Serialization;

namespace DnsFilter.Avalonia.Notifications;

internal sealed record DeadlineItem(
    [property: JsonPropertyName("key")] string Key,
    [property: JsonPropertyName("title")] string Title,
    [property: JsonPropertyName("body")] string Body,
    [property: JsonPropertyName("kind")] string Kind,
    [property: JsonPropertyName("due_at")] DateTimeOffset DueAt);

internal sealed class NotificationState
{
    private readonly object sync = new();
    private string storagePath = FilePath;
    public bool Enabled { get; set; } = true;
    public bool StartWithWindows { get; set; } = true;
    public Dictionary<string, DateTimeOffset> Delivered { get; set; } = new();
    internal static string FilePath => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "DNSFilter", "desktop-notifications.json");
    internal static NotificationState Load(string? path = null)
    {
        path ??= FilePath;
        NotificationState state;
        try { state = JsonSerializer.Deserialize<NotificationState>(File.ReadAllText(path)) ?? new(); }
        catch (Exception error) when (error is IOException or JsonException or UnauthorizedAccessException) { state = new(); }
        state.storagePath = path;
        return state;
    }
    internal void Save()
    {
        lock (sync) SaveLocked();
    }
    internal bool WasDelivered(string key)
    {
        lock (sync) { Delivered ??= new(); return Delivered.ContainsKey(key); }
    }
    internal void MarkDelivered(string key)
    {
        lock (sync) { Delivered ??= new(); Delivered[key] = DateTimeOffset.UtcNow; SaveLocked(); }
    }
    private void SaveLocked()
    {
        // Candidate deadlines have a 24-hour catch-up window. Keep deduplication
        // longer than that so a desktop/service restart cannot replay old alerts.
        Delivered ??= new();
        var cutoff = DateTimeOffset.UtcNow.AddDays(-7);
        foreach (var key in Delivered.Where(item => item.Value < cutoff).Select(item => item.Key).ToArray()) Delivered.Remove(key);
        Directory.CreateDirectory(Path.GetDirectoryName(storagePath)!);
        var temporary = storagePath + ".tmp";
        File.WriteAllText(temporary, JsonSerializer.Serialize(this));
        File.Move(temporary, storagePath, true);
    }
}
