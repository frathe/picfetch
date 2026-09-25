import Foundation
import AppKit
import ScreenCaptureKit
import CoreMedia
import CoreVideo
import CoreImage
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers

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
}

// Both input and WindowServer display times use Mach absolute ticks, converted
// through this one timebase. Media PTS and CGEvent.timestamp are not substituted.
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
    var pending: (Command, CVPixelBuffer, UInt64, UInt64, CheckedContinuation<Observation, Never>)?

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
        guard let (command, before, beforeHash, inputTicks, continuation) = pending,
              currentHash != beforeHash, displayTicks > inputTicks else { return }
        pending = nil
        var observation = Observation(kind: command.kind, input_ns: nanoseconds(inputTicks), visible_ns: nanoseconds(displayTicks))
        do {
            // Entry feedback is an orchestration boundary, not a latency sample.
            // Avoid PNG encoding before the immediate scan-cancellation trial.
            if command.kind != "open" {
                observation.before = command.name + "-before.png"
                observation.after = command.name + "-after.png"
                try save(before, name: observation.before)
                try save(pixels, name: observation.after)
            }
        } catch {
            observation.error = String(describing: error)
        }
        continuation.resume(returning: observation)
    }

    func stream(_ stream: SCStream, didStopWithError error: Error) {
        queue.async {
            if let (command, _, _, input, continuation) = self.pending {
                self.pending = nil
                continuation.resume(returning: Observation(kind: command.kind, input_ns: nanoseconds(input), error: String(describing: error)))
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
        let needsStableFrame = command.kind == "pan" || command.kind == "zoom"
        if latest == nil || (needsStableFrame && now - nanoseconds(changedAt) < 250_000_000) {
            if now >= deadline {
                continuation.resume(returning: Observation(kind: command.kind, skipped: true, error: "no stable native frame before input"))
            } else {
                queue.asyncAfter(deadline: .now() + .milliseconds(20)) { self.admit(command, continuation, deadline: deadline) }
            }
            return
        }
        guard pending == nil, let before = latest,
              let down = CGEvent(keyboardEventSource: nil, virtualKey: command.key, keyDown: true),
              let up = CGEvent(keyboardEventSource: nil, virtualKey: command.key, keyDown: false) else {
            continuation.resume(returning: Observation(kind: command.kind, skipped: true, error: "cannot admit native input"))
            return
        }
        if command.shift { down.flags = .maskShift; up.flags = .maskShift }
        let input = mach_absolute_time()
        pending = (command, before, latestHash, input, continuation)
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
            guard let (_, _, _, current, callback) = self.pending, current == input else { return }
            self.pending = nil
            callback.resume(returning: Observation(kind: command.kind, input_ns: nanoseconds(input), error: "no changed native frame after input"))
        }
    }
}

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
