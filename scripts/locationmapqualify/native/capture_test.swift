// Pure observer policy checks: no screen capture, input injection or permissions.
@main struct CapturePolicyTests {
    static func main() {
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
        precondition(isResponseFrame(kind: "pan", current: 20, before: 10, closed: 30))
        precondition(!isResponseFrame(kind: "zoom", current: 10, before: 10, closed: 30))
        print("Native Location Map response-frame policy passed")
    }
}
