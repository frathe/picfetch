import Foundation
#if canImport(AppKit)
import AppKit
import ScreenCaptureKit
import CoreMedia
import CoreVideo
import CoreImage
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers
#endif

struct CaptureFailure: Error, CustomStringConvertible {
    let description: String
}

struct Command: Decodable {
    let kind: String
    let key: UInt16
    let shift: Bool
    let name: String
}

struct Observation: Encodable {
    var kind: String
    var input_ns: UInt64 = 0
    var visible_ns: UInt64 = 0
    var before = ""
    var after = ""
    var skipped = false
    var error = ""
    var closed_viewer = false
    var identified = false
    var transform: VisualTransform?
}

// Exit timing must identify the viewer captured before map entry. Scanning and
// tile delivery can change arbitrary map pixels after Escape was posted.
func isResponseFrame(kind: String, current: UInt64, before: UInt64, closed: UInt64?, identified: Bool = false) -> Bool {
    guard current != before else { return false }
    if kind == "cancel" || kind == "close" { return closed == current }
    if kind == "pan" || kind == "zoom" { return identified }
    return true
}

// Both input and WindowServer display times use Mach absolute ticks, converted
// through this one timebase. Media PTS and CGEvent.timestamp are not substituted.
#if canImport(AppKit)
func nanoseconds(_ ticks: UInt64) -> UInt64 {
    var base = mach_timebase_info_data_t()
    mach_timebase_info(&base)
    return ticks / UInt64(base.denom) * UInt64(base.numer)
        + ticks % UInt64(base.denom) * UInt64(base.numer) / UInt64(base.denom)
}

// Mutable capture/input state is confined to queue; public execute only enqueues.
final class Observer: NSObject, SCStreamOutput, SCStreamDelegate, @unchecked Sendable {
    let queue = DispatchQueue(label: "picfetch.native.capture")
    let pid: pid_t
    let directory: URL
    let context = CIContext(options: [.cacheIntermediates: false])
    var latest: CVPixelBuffer?
    var latestHash: UInt64 = 0
    var changedAt: UInt64 = 0
    var closedViewerHash: UInt64?
    struct Pending {
        let command: Command
        let before: CVPixelBuffer
        let frame: VisualFrame
        let hash: UInt64
        let input: UInt64
        let continuation: CheckedContinuation<Observation, Never>
    }
    var pending: Pending?

    init(pid: pid_t, directory: URL) {
        self.pid = pid
        self.directory = directory
    }

    // Sample the map body, excluding title/progress bars, pointer and toast area.
    // Screen pixels, not UI state or a rendered test canvas, establish a change.
    func hash(_ pixels: CVPixelBuffer) -> UInt64? {
        guard CVPixelBufferLockBaseAddress(pixels, .readOnly) == kCVReturnSuccess else { return nil }
        defer { CVPixelBufferUnlockBaseAddress(pixels, .readOnly) }
        guard let address = CVPixelBufferGetBaseAddress(pixels) else { return nil }
        let width = CVPixelBufferGetWidth(pixels), height = CVPixelBufferGetHeight(pixels)
        let strideBytes = CVPixelBufferGetBytesPerRow(pixels)
        let bytes = address.assumingMemoryBound(to: UInt8.self)
        var value: UInt64 = 14695981039346656037
        for y in stride(from: height / 5, to: height * 4 / 5, by: 3) {
            for x in stride(from: width / 10, to: width * 9 / 10, by: 3) {
                let offset = y * strideBytes + x * 4
                for channel in 0..<3 { value = (value ^ UInt64(bytes[offset + channel])) &* 1099511628211 }
            }
        }
        return value
    }

    // Average each 4x4 block before registration. Retain no application facts;
    // both baseline and response are independent WindowServer screen pixels.
    func visualFrame(_ pixels: CVPixelBuffer) -> VisualFrame? {
        guard CVPixelBufferLockBaseAddress(pixels, .readOnly) == kCVReturnSuccess else { return nil }
        defer { CVPixelBufferUnlockBaseAddress(pixels, .readOnly) }
        guard let address = CVPixelBufferGetBaseAddress(pixels) else { return nil }
        let width = CVPixelBufferGetWidth(pixels) / 4, height = CVPixelBufferGetHeight(pixels) / 4
        let rowBytes = CVPixelBufferGetBytesPerRow(pixels)
        let bytes = address.assumingMemoryBound(to: UInt8.self)
        var values = [UInt8]()
        values.reserveCapacity(width * height)
        for y in 0..<height {
            for x in 0..<width {
                var sum = 0
                for py in 0..<4 {
                    for px in 0..<4 {
                        let offset = (y * 4 + py) * rowBytes + (x * 4 + px) * 4
                        sum += Int(bytes[offset]) * 29 + Int(bytes[offset + 1]) * 150 + Int(bytes[offset + 2]) * 77
                    }
                }
                values.append(UInt8(sum / (16 * 256)))
            }
        }
        return VisualFrame(width: width, height: height, luminance: values)
    }

    func stream(_ stream: SCStream, didOutputSampleBuffer sampleBuffer: CMSampleBuffer, of outputType: SCStreamOutputType) {
        guard outputType == .screen,
              let attachments = CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, createIfNecessary: false) as? [[SCStreamFrameInfo: Any]],
              let information = attachments.first,
              let rawStatus = information[.status] as? Int,
              SCFrameStatus(rawValue: rawStatus) == .complete,
              let displayTicks = information[.displayTime] as? UInt64,
              let pixels = CMSampleBufferGetImageBuffer(sampleBuffer) else { return }
        guard let currentHash = hash(pixels) else { return }
        if currentHash != latestHash { changedAt = mach_absolute_time() }
        latest = pixels
        latestHash = currentHash
        guard let work = pending, displayTicks > work.input else { return }
        let command = work.command
        var transform: VisualTransform?
        if (command.kind == "pan" || command.kind == "zoom"), currentHash != work.hash,
           let current = visualFrame(pixels) {
            transform = identifyTransform(before: work.frame, after: current, kind: command.kind,
                                          key: command.key, shift: command.shift)
        }
        guard isResponseFrame(kind: command.kind, current: currentHash, before: work.hash,
                              closed: closedViewerHash, identified: transform != nil) else { return }
        pending = nil
        var observation = Observation(kind: command.kind, input_ns: nanoseconds(work.input), visible_ns: nanoseconds(displayTicks))
        observation.closed_viewer = command.kind == "cancel" || command.kind == "close"
        observation.identified = transform != nil
        observation.transform = transform
        do {
            // Entry feedback is an orchestration boundary, not a latency sample.
            // Avoid PNG encoding before the immediate scan-cancellation trial.
            if command.kind != "open" {
                observation.before = command.name + "-before.png"
                observation.after = command.name + "-after.png"
                try save(work.before, name: observation.before)
                try save(pixels, name: observation.after)
            }
        } catch {
            observation.error = String(describing: error)
        }
        work.continuation.resume(returning: observation)
    }

    func stream(_ stream: SCStream, didStopWithError error: Error) {
        queue.async {
            if let work = self.pending {
                self.pending = nil
                work.continuation.resume(returning: Observation(kind: work.command.kind, input_ns: nanoseconds(work.input), error: String(describing: error)))
            }
        }
    }

    func save(_ pixels: CVPixelBuffer, name: String) throws {
        let image = CIImage(cvPixelBuffer: pixels)
        guard let cgImage = context.createCGImage(image, from: image.extent),
              let output = CGImageDestinationCreateWithURL(directory.appendingPathComponent(name) as CFURL, UTType.png.identifier as CFString, 1, nil) else {
            throw CaptureFailure(description: "cannot create screen artifact")
        }
        CGImageDestinationAddImage(output, cgImage, nil)
        if !CGImageDestinationFinalize(output) { throw CaptureFailure(description: "cannot finish screen artifact") }
    }

    func execute(_ command: Command) async -> Observation {
        await withCheckedContinuation { continuation in
            queue.async { self.admit(command, continuation, deadline: nanoseconds(mach_absolute_time()) + 10_000_000_000) }
        }
    }

    func admit(_ command: Command, _ continuation: CheckedContinuation<Observation, Never>, deadline: UInt64) {
        let now = nanoseconds(mach_absolute_time())
        let needsStableFrame = command.kind == "pan" || command.kind == "zoom" || command.kind == "open"
        if latest == nil || (needsStableFrame && now - nanoseconds(changedAt) < 250_000_000) {
            if now >= deadline {
                continuation.resume(returning: Observation(kind: command.kind, skipped: true, error: "no stable native frame before input"))
            } else {
                queue.asyncAfter(deadline: .now() + .milliseconds(20)) { self.admit(command, continuation, deadline: deadline) }
            }
            return
        }
        guard pending == nil, let before = latest, let frame = visualFrame(before),
              let down = CGEvent(keyboardEventSource: nil, virtualKey: command.key, keyDown: true),
              let up = CGEvent(keyboardEventSource: nil, virtualKey: command.key, keyDown: false) else {
            continuation.resume(returning: Observation(kind: command.kind, skipped: true, error: "cannot admit native input"))
            return
        }
        if command.shift { down.flags = .maskShift; up.flags = .maskShift }
        if command.kind == "open" { closedViewerHash = latestHash }
        let input = mach_absolute_time()
        pending = Pending(command: command, before: before, frame: frame, hash: latestHash, input: input, continuation: continuation)
        if command.shift, let modifier = CGEvent(keyboardEventSource: nil, virtualKey: 0x38, keyDown: true) {
            modifier.type = .flagsChanged
            modifier.flags = .maskShift
            modifier.postToPid(pid)
        }
        down.postToPid(pid)
        up.postToPid(pid)
        if command.shift, let modifier = CGEvent(keyboardEventSource: nil, virtualKey: 0x38, keyDown: false) {
            modifier.type = .flagsChanged
            modifier.flags = []
            modifier.postToPid(pid)
        }
        queue.asyncAfter(deadline: .now() + .seconds(3)) {
            guard let work = self.pending, work.input == input else { return }
            self.pending = nil
            var observation = Observation(kind: command.kind, input_ns: nanoseconds(input), error: "no identified native response within 3s")
            // Failed attempts retain pixels too; never pick a later frame and
            // invent a successful timestamp when registration cannot identify it.
            do {
                observation.before = command.name + "-before.png"
                try self.save(work.before, name: observation.before)
                if let latest = self.latest {
                    observation.after = command.name + "-after.png"
                    try self.save(latest, name: observation.after)
                }
            } catch { observation.error += "; " + String(describing: error) }
            work.continuation.resume(returning: observation)
        }
    }
}

#if !CAPTURE_TEST
@main struct CaptureMain {
    static func main() async {
        do {
            guard CommandLine.arguments.count == 3, let pid = Int32(CommandLine.arguments[1]) else {
                throw CaptureFailure(description: "usage: location-map-capture PID EVIDENCE_DIR")
            }
            guard CGPreflightScreenCaptureAccess(), CGPreflightPostEventAccess() else {
                throw CaptureFailure(description: "native capture/input permission is absent; grant Screen Recording and Accessibility permissions manually before retrying")
            }
            var target: SCWindow?
            for _ in 0..<100 {
                let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: true)
                target = content.windows.first { $0.owningApplication?.processID == pid && $0.frame.width > 100 && $0.frame.height > 100 }
                if target != nil { break }
                try await Task.sleep(nanoseconds: 100_000_000)
            }
            guard let target else { throw CaptureFailure(description: "launched PicFetch window was not found") }
            let observer = Observer(pid: pid, directory: URL(fileURLWithPath: CommandLine.arguments[2]))
            let configuration = SCStreamConfiguration()
            configuration.width = 1200
            configuration.height = 800
            configuration.minimumFrameInterval = CMTime(value: 1, timescale: 120)
            configuration.pixelFormat = kCVPixelFormatType_32BGRA
            configuration.showsCursor = false
            configuration.queueDepth = 3
            let stream = SCStream(filter: SCContentFilter(desktopIndependentWindow: target), configuration: configuration, delegate: observer)
            try stream.addStreamOutput(observer, type: .screen, sampleHandlerQueue: observer.queue)
            try await stream.startCapture()
            while let line = readLine() {
                let command = try JSONDecoder().decode(Command.self, from: Data(line.utf8))
                guard !command.name.isEmpty, command.name.allSatisfy({ $0.isASCII && ($0.isLetter || $0.isNumber || $0 == "-") }) else {
                    throw CaptureFailure(description: "invalid artifact name")
                }
                guard NSWorkspace.shared.frontmostApplication?.processIdentifier == pid else {
                    throw CaptureFailure(description: "PicFetch is not the foreground application; native observations cannot qualify an obscured window")
                }
                let result = await observer.execute(command)
                var output = try JSONEncoder().encode(result)
                output.append(10)
                FileHandle.standardOutput.write(output)
            }
            try await stream.stopCapture()
        } catch {
            FileHandle.standardError.write(Data((String(describing: error) + "\n").utf8))
            exit(1)
        }
    }
}
#endif
#endif
