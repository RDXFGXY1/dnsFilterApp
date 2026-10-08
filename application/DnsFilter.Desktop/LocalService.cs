using System.Diagnostics;
using System.IO;

namespace DnsFilter.Desktop;

public static class LocalService
{
    public static bool TryStart(out string message)
    {
        var directory = new DirectoryInfo(AppContext.BaseDirectory);
        for (var depth = 0; directory is not null && depth < 6; depth++, directory = directory.Parent)
        {
            var installed = Path.Combine(directory.FullName, "dns-filter.exe");
            var directDevelopment = Path.Combine(directory.FullName, "dns-filter-gui-server.exe");
            var nestedInstalled = Path.Combine(directory.FullName, "build", "dns-filter.exe");
            var nestedDevelopment = Path.Combine(directory.FullName, "build", "dns-filter-gui-server.exe");
            var executable = File.Exists(installed) ? installed
                : File.Exists(directDevelopment) ? directDevelopment
                : File.Exists(nestedInstalled) ? nestedInstalled
                : File.Exists(nestedDevelopment) ? nestedDevelopment
                : null;
            if (executable is null) continue;

            var workingDirectory = Path.GetFileName(directory.FullName).Equals("build", StringComparison.OrdinalIgnoreCase)
                ? directory.Parent?.FullName ?? directory.FullName
                : directory.FullName;
            Process.Start(new ProcessStartInfo(executable)
            {
                WorkingDirectory = workingDirectory,
                UseShellExecute = true,
                WindowStyle = ProcessWindowStyle.Hidden
            });
            message = "Starting the local protection service…";
            return true;
        }
        message = "The Go service was not found. Keep this application inside the project's build folder.";
        return false;
    }
}
