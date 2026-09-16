# CI-only alternate-user process creation using an explicitly authenticated
# primary token. The installed test still verifies the actual account/groups.
Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Security;
using System.Text;
public static class HEICStandardUserLogon {
 [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
 struct StartupInfo {
  public int Size;
  public string Reserved, Desktop, Title;
  public uint X, Y, Width, Height, XChars, YChars, Fill, Flags;
  public ushort Show, ReservedSize;
  public IntPtr ReservedBytes, Input, Output, Error;
 }
 [StructLayout(LayoutKind.Sequential)]
 struct ProcessInfo {
  public IntPtr Process, Thread;
  public uint ProcessId, ThreadId;
 }
 [DllImport("advapi32.dll", CharSet = CharSet.Unicode, ExactSpelling = true, SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool LogonUserW(string user, string domain, IntPtr password, uint type, uint provider, out IntPtr token);
 [DllImport("advapi32.dll", CharSet = CharSet.Unicode, ExactSpelling = true, SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool CreateProcessWithTokenW(IntPtr token, uint logonFlags, string application,
  StringBuilder command, uint flags, IntPtr environment, string directory, ref StartupInfo startup, out ProcessInfo process);
 [DllImport("kernel32.dll", SetLastError = true)]
 static extern uint WaitForSingleObject(IntPtr handle, uint milliseconds);
 [DllImport("kernel32.dll", SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool GetExitCodeProcess(IntPtr process, out uint code);
 [DllImport("kernel32.dll", SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool TerminateProcess(IntPtr process, uint code);
 [DllImport("kernel32.dll")]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool CloseHandle(IntPtr handle);
 public static uint Run(string user, string domain, SecureString password, string application, string arguments, string directory) {
  IntPtr token = IntPtr.Zero;
  IntPtr secret = Marshal.SecureStringToGlobalAllocUnicode(password);
  try {
   // LOGON32_LOGON_INTERACTIVE, LOGON32_PROVIDER_DEFAULT. This authenticates
   // the Users-only account; it does not derive a filtered administrator token.
   if (!LogonUserW(user, domain, secret, 2, 0, out token))
    throw new Win32Exception(Marshal.GetLastWin32Error(), "Standard-user authentication failed.");
  } finally { Marshal.ZeroFreeGlobalAllocUnicode(secret); }
  try {
   var startup = new StartupInfo { Size = Marshal.SizeOf(typeof(StartupInfo)) };
   var command = new StringBuilder("\"" + application + "\" " + arguments);
   ProcessInfo process;
   // LOGON_WITH_PROFILE; Windows supplies the account's environment and grants
   // that account access to the inherited desktop when Desktop is null.
   if (!CreateProcessWithTokenW(token, 1, application, command, 0, IntPtr.Zero, directory, ref startup, out process))
    throw new Win32Exception(Marshal.GetLastWin32Error(), "Standard-user process creation failed.");
   try {
    uint wait = WaitForSingleObject(process.Process, 600000);
    if (wait != 0) {
     int error = Marshal.GetLastWin32Error();
     if (!TerminateProcess(process.Process, 1)) throw new Win32Exception(Marshal.GetLastWin32Error());
     WaitForSingleObject(process.Process, 10000);
     if (wait == 258) throw new TimeoutException("Standard-user qualification exceeded ten minutes.");
     throw new Win32Exception(error, "Cannot wait for standard-user qualification.");
    }
    uint code;
    if (!GetExitCodeProcess(process.Process, out code)) throw new Win32Exception(Marshal.GetLastWin32Error());
    return code;
   } finally { CloseHandle(process.Thread); CloseHandle(process.Process); }
  } finally { CloseHandle(token); }
 }
}
'@
