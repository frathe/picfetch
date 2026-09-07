import dbus, dbus.service, dbus.mainloop.glib
from gi.repository import GLib
import json, os, pathlib, subprocess, tempfile, threading, urllib.parse

dbus.mainloop.glib.DBusGMainLoop(set_as_default=True)
bus = dbus.SessionBus()
name = dbus.service.BusName("org.freedesktop.portal.Desktop", bus)
loop = GLib.MainLoop()
selection = []
requests = []
results = []
failure = []
class Request(dbus.service.Object):
    @dbus.service.signal("org.freedesktop.portal.Request", signature="ua{sv}")
    def Response(self, code, result): pass
class Portal(dbus.service.Object):
    @dbus.service.method("org.freedesktop.DBus.Properties", in_signature="s", out_signature="a{sv}")
    def GetAll(self, interface):
        print("portal GetAll", interface, flush=True)
        return {"version": dbus.UInt32(3)}
    @dbus.service.method("org.freedesktop.DBus.Properties", in_signature="ss", out_signature="v")
    def Get(self, interface, property): return dbus.UInt32(3)
    def respond(self, sender, options):
        print("portal selection", selection, flush=True)
        handle = "/org/freedesktop/portal/desktop/request/" + sender[1:].replace(".", "_") + "/" + str(options["handle_token"])
        request = Request(bus, handle)
        requests.append(request)
        uris = dbus.Array(["file://" + urllib.parse.quote(p) for p in selection], signature="s")
        def publish():
            message = dbus.lowlevel.SignalMessage(handle, "org.freedesktop.portal.Request", "Response")
            message.set_destination(sender)
            message.append(dbus.UInt32(0 if selection else 1), {"uris": uris}, signature="ua{sv}")
            bus.send_message(message)
            return False
        GLib.idle_add(publish)
        return dbus.ObjectPath(handle)
    @dbus.service.method("org.freedesktop.portal.FileChooser", in_signature="ssa{sv}", out_signature="o", sender_keyword="sender")
    def OpenFile(self, parent, title, options, sender=None): return self.respond(sender, options)
    @dbus.service.method("org.freedesktop.portal.FileChooser", in_signature="ssa{sv}", out_signature="o", sender_keyword="sender")
    def SaveFile(self, parent, title, options, sender=None): return self.respond(sender, options)
portal = Portal(bus, "/org/freedesktop/portal/desktop")
def run():
    global selection
    try:
        root = pathlib.Path(tempfile.mkdtemp(prefix="picfetch-portal-"))
        paths = [str(root / n) for n in ["line\nnext.jpg", "tail\r", "tail\n", " spaced ", "café 東京 😀.png"]]
        for p in paths: pathlib.Path(p).write_bytes(b"native transport fixture")
        separator = "//picfetch-file//"
        for multiple, selected in [(False, [p]) for p in paths] + [(True, paths), (False, []), (True, [])]:
            selection = selected
            args = ["zenity", "--file-selection", "--title=NativeProtocolProbe"]
            if multiple: args += ["--multiple", "--separator=" + separator]
            else: args += ["--save", "--confirm-overwrite", "--filename=" + (selected[0] if selected else str(root / "cancel.jpg"))]
            with tempfile.TemporaryFile() as output, tempfile.TemporaryFile() as errors:
                proc = subprocess.Popen(args, stdout=output, stderr=errors, env=dict(os.environ, GTK_USE_PORTAL="1", GDK_DEBUG="portals", GSK_RENDERER="cairo"))
                try: code = proc.wait(timeout=30)
                except subprocess.TimeoutExpired:
                    errors.seek(0); print("native stderr", errors.read(), flush=True)
                    output.seek(0); print("native stdout", output.read(), flush=True)
                    raise
                finally:
                    if proc.poll() is None: proc.kill(); proc.wait()
                output.seek(0); out = output.read()
                errors.seek(0); err = errors.read()
            expected = (separator.join(selected) + "\n").encode() if selected else b""
            assert code == (0 if selected else 1) and out == expected, (code, out, expected, err)
            results.append({"multiple": multiple, "paths": selected, "output": out.decode(), "exit_code": code})
            print("native Zenity case passed", multiple, repr(selected), flush=True)
        pathlib.Path("/probe/native-results.json").write_text(json.dumps(results))
    except BaseException as error:
        failure.append(error)
    finally: GLib.idle_add(loop.quit)
worker = threading.Thread(target=run)
worker.start(); loop.run(); worker.join()
if failure: raise failure[0]
