// Pure observer policy checks: no screen capture, input injection or permissions.
@main struct CapturePolicyTests {
    static func main() {
        precondition(isResponseWithinDeadline(inputNS: 10, displayNS: 3_000_000_009),
                     "A captured response before the deadline remains eligible")
        for displayNS: UInt64 in [9, 10, 3_000_000_010, 3_000_000_011, UInt64.max] {
            precondition(!isResponseWithinDeadline(inputNS: 10, displayNS: displayNS),
                         "A stale or expired captured frame must not beat a queued timeout")
        }
        precondition(isResponseWithinDeadline(inputNS: UInt64.max - 1, displayNS: UInt64.max),
                     "Deadline checks must not overflow when comparing timestamps")
        for kind in ["cancel", "close"] {
            precondition(!isResponseFrame(kind: kind, current: 20, before: 10, closed: 30),
                         "An in-flight map frame must not complete exit timing")
            precondition(isResponseFrame(kind: kind, current: 30, before: 10, closed: 30),
                         "The observed closed viewer must complete exit timing")
            precondition(!isResponseFrame(kind: kind, current: 30, before: 10, closed: nil),
                         "Missing viewer baseline must fail closed")
            precondition(!isResponseFrame(kind: kind, current: 30, before: 30, closed: 30),
                         "An unchanged frame is not an observed response")
        }
        precondition(!isResponseFrame(kind: "pan", current: 20, before: 10, closed: 30),
                     "Unidentified changed pixels must not qualify pan timing")
        precondition(!isResponseFrame(kind: "zoom", current: 20, before: 10, closed: 30),
                     "Unidentified changed pixels must not qualify zoom timing")
        precondition(!isResponseFrame(kind: "zoom", current: 10, before: 10, closed: 30))
        for kind in ["pan", "zoom"] {
            precondition(isResponseFrame(kind: kind, current: 20, before: 10, closed: nil, identified: true),
                         "A verified changed transform must complete gesture timing")
            precondition(!isResponseFrame(kind: kind, current: 10, before: 10, closed: nil, identified: true),
                         "An unchanged frame must never complete gesture timing")
        }
        print("Native Location Map response-frame policy passed")
    }
}
