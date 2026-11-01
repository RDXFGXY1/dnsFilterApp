using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;
using System.Text;
using Microsoft.Win32;

namespace DnsFilter.Avalonia.Notifications;

internal static class WindowsIntegration
{
    internal const string AppId = "DNSFilter.Desktop";
    internal static string Executable => Environment.ProcessPath ?? throw new InvalidOperationException("Cannot locate DNS Filter desktop.");
    internal static string IconPath => Path.Combine(AppContext.BaseDirectory, "Assets", "dns-filter.png");
    internal static void RegisterIdentity()
    {
        if (!OperatingSystem.IsWindows()) return;
        Marshal.ThrowExceptionForHR(SetCurrentProcessExplicitAppUserModelID(AppId));
        using (var key = Registry.CurrentUser.CreateSubKey(@"Software\Classes\AppUserModelId\" + AppId))
        {
            key.SetValue("DisplayName", "DNS Filter");
            key.SetValue("IconUri", IconPath);
        }
        using (var protocol = Registry.CurrentUser.CreateSubKey(@"Software\Classes\dnsfilter-app"))
        {
            protocol.SetValue("", "URL:DNS Filter");
            protocol.SetValue("URL Protocol", "");
            using var command = protocol.CreateSubKey(@"shell\open\command");
            // Deliberately do not interpolate a URI into the shell command.
            command.SetValue("", $"\"{Executable}\" --open-discipline");
        }
        CreateShortcut();
    }
    internal static bool StartsWithWindows()
    {
        if (!OperatingSystem.IsWindows()) return false;
        using var key = Registry.CurrentUser.OpenSubKey(@"Software\Microsoft\Windows\CurrentVersion\Run");
        return string.Equals(key?.GetValue("DNSFilterDesktop") as string, $"\"{Executable}\" --tray", StringComparison.OrdinalIgnoreCase);
    }
    internal static void SetStartsWithWindows(bool enabled)
    {
        if (!OperatingSystem.IsWindows()) return;
        using var key = Registry.CurrentUser.CreateSubKey(@"Software\Microsoft\Windows\CurrentVersion\Run");
        if (enabled) key.SetValue("DNSFilterDesktop", $"\"{Executable}\" --tray");
        else key.DeleteValue("DNSFilterDesktop", false);
    }
    internal static void OpenDiscipline() => Process.Start(new ProcessStartInfo("http://127.0.0.1:8080/open?view=discipline") { UseShellExecute = true });
    private static void CreateShortcut()
    {
        if (!OperatingSystem.IsWindows()) return;
        var folder = Environment.GetFolderPath(Environment.SpecialFolder.Programs);
        Directory.CreateDirectory(folder);
        object shell = Activator.CreateInstance(Type.GetTypeFromCLSID(new Guid("00021401-0000-0000-C000-000000000046"))!)!;
        try
        {
            var link = (IShellLinkW)shell;
            link.SetPath(Executable);
            link.SetArguments("");
            link.SetDescription("DNS Filter — local protection and deadline notifications");
            link.SetWorkingDirectory(AppContext.BaseDirectory);
            link.SetIconLocation(Path.Combine(AppContext.BaseDirectory, "Assets", "dns-filter.ico"), 0);
            var store = (IPropertyStore)shell;
            var key = new PropertyKey { FormatId = new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"), PropertyId = 5 };
            var value = new PropVariant { Type = 31, Value = Marshal.StringToCoTaskMemUni(AppId) };
            try { Marshal.ThrowExceptionForHR(store.SetValue(ref key, ref value)); Marshal.ThrowExceptionForHR(store.Commit()); }
            finally { Marshal.FreeCoTaskMem(value.Value); }
            ((IPersistFile)shell).Save(Path.Combine(folder, "DNS Filter Desktop.lnk"), true);
        }
        finally { Marshal.FinalReleaseComObject(shell); }
    }
    [DllImport("shell32.dll", CharSet = CharSet.Unicode)] private static extern int SetCurrentProcessExplicitAppUserModelID(string appId);
    [StructLayout(LayoutKind.Sequential)] private struct PropertyKey { public Guid FormatId; public uint PropertyId; }
    [StructLayout(LayoutKind.Explicit, Size = 24)] private struct PropVariant { [FieldOffset(0)] public ushort Type; [FieldOffset(8)] public IntPtr Value; }
    [ComImport, Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    private interface IPropertyStore
    {
        [PreserveSig] int GetCount(out uint count);
        [PreserveSig] int GetAt(uint index, out PropertyKey key);
        [PreserveSig] int GetValue(ref PropertyKey key, out PropVariant value);
        [PreserveSig] int SetValue(ref PropertyKey key, ref PropVariant value);
        [PreserveSig] int Commit();
    }
    [ComImport, Guid("000214F9-0000-0000-C000-000000000046"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    private interface IShellLinkW
    {
        void GetPath([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder path, int max, IntPtr data, uint flags);
        void GetIDList(out IntPtr list); void SetIDList(IntPtr list);
        void GetDescription([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder text, int max);
        void SetDescription([MarshalAs(UnmanagedType.LPWStr)] string text);
        void GetWorkingDirectory([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder path, int max);
        void SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string path);
        void GetArguments([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder args, int max);
        void SetArguments([MarshalAs(UnmanagedType.LPWStr)] string args);
        void GetHotkey(out short value); void SetHotkey(short value);
        void GetShowCmd(out int value); void SetShowCmd(int value);
        void GetIconLocation([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder path, int max, out int index);
        void SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string path, int index);
        void SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string path, uint reserved);
        void Resolve(IntPtr hwnd, uint flags);
        void SetPath([MarshalAs(UnmanagedType.LPWStr)] string path);
    }
}
