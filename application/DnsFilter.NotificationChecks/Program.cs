using DnsFilter.Avalonia.Notifications;
using System.Xml.Linq;

var root = Path.Combine(Path.GetTempPath(), "dns-filter-notification-checks-" + Guid.NewGuid());
var file = Path.Combine(root, "state.json");
var state = NotificationState.Load(file);
state.MarkDelivered("due-task-1");
state.Enabled = false; state.Save();
var restored = NotificationState.Load(file);
if (!restored.WasDelivered("due-task-1") || restored.Enabled) throw new Exception("Delivery state did not survive a restart.");
state.Delivered["old"] = DateTimeOffset.UtcNow.AddDays(-8); state.Save();
if (NotificationState.Load(file).WasDelivered("old")) throw new Exception("Old delivery state was not pruned.");
var text = "Math <script> & \"revision\" مراجعة";
var xml = XElement.Parse(WindowsNotifier.BuildXml(text, text));
if (xml.Descendants("text").Any(node => node.Value != text) || xml.Descendants("script").Any()) throw new Exception("Task text was not escaped safely.");
if (xml.Descendants("image").Single().Attribute("placement")?.Value != "appLogoOverride") throw new Exception("Missing DNS Filter notification icon.");
Console.WriteLine("Notification checks passed: persistent deduplication, preferences, pruning, Unicode/XML safety, and branded icon.");
