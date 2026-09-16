# Read the owner of this process's actual Terminal Services session. A loaded
# profile or an interactive logon type alone does not establish session ownership.
Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Security.Principal;
public static class HEICDesktopSession {
 [DllImport("wtsapi32.dll", CharSet = CharSet.Unicode, ExactSpelling = true, SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool WTSQuerySessionInformationW(IntPtr server, int session, int kind, out IntPtr buffer, out int bytes);
 [DllImport("wtsapi32.dll")]
 static extern void WTSFreeMemory(IntPtr buffer);
 static string Query(int session, int kind) {
  IntPtr buffer;
  int bytes;
  if (!WTSQuerySessionInformationW(IntPtr.Zero, session, kind, out buffer, out bytes))
   throw new Win32Exception(Marshal.GetLastWin32Error());
  try {
   if (bytes < 2) throw new InvalidOperationException("Session has no owner.");
   return Marshal.PtrToStringUni(buffer);
  } finally { WTSFreeMemory(buffer); }
 }
 public static SecurityIdentifier Owner(int session) {
  if (session == 0) throw new InvalidOperationException("MSIX qualification requires a desktop session.");
  string user = Query(session, 5); // WTSUserName
  string domain = Query(session, 7); // WTSDomainName
  if (String.IsNullOrEmpty(user) || String.IsNullOrEmpty(domain))
   throw new InvalidOperationException("Session has no named owner.");
  return (SecurityIdentifier)new NTAccount(domain, user).Translate(typeof(SecurityIdentifier));
 }
}
'@

function Get-HEICDesktopIdentity {
    $session = [System.Diagnostics.Process]::GetCurrentProcess().SessionId
    $owner = [HEICDesktopSession]::Owner($session)
    $current = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    try {
        if ($current.User.Value -ne $owner.Value) { throw 'MSIX launch account must own its desktop session.' }
        [pscustomobject]@{ SessionID = $session; UserSID = $current.User.Value; Name = $current.Name }
    } finally { $current.Dispose() }
}
