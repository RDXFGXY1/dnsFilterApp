$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
[Console]::InputEncoding=[Text.Encoding]::UTF8
[Console]::OutputEncoding=[Text.Encoding]::UTF8
try {
    $request=[Console]::In.ReadToEnd()|ConvertFrom-Json
    [Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime]|Out-Null
    [Windows.Data.Xml.Dom.XmlDocument,Windows.Data.Xml.Dom.XmlDocument,ContentType=WindowsRuntime]|Out-Null
    $xml=New-Object Windows.Data.Xml.Dom.XmlDocument
    $xml.LoadXml([string]$request.xml)
    $notifier=[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('DNSFilter.Desktop')
    # Some Windows builds expose an empty setting through the WinRT projection.
    # In that case let Show report delivery errors instead of treating null as disabled.
    if($null -ne $notifier.Setting -and $notifier.Setting -ne [Windows.UI.Notifications.NotificationSetting]::Enabled){throw ('Windows notifications are disabled: '+$notifier.Setting)}
    $toast=[Windows.UI.Notifications.ToastNotification]::new($xml)
    $toast.Tag=[string]$request.tag
    $toast.Group='deadlines'
    $notifier.Show($toast)
    [Console]::Out.WriteLine('accepted')
} catch {
    [Console]::Error.WriteLine($_.Exception.Message)
    exit 1
}
