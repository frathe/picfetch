import Foundation

// Synthetic camera scenes exercise the observer boundary, not application state.
// No screen recording, input posting, permissions or native performance claims.
@main struct TransformTests {
    static func main() {
        let before = scene()
        let right = scene(dx: 15)
        let left = scene(dx: -15)
        let zoomIn = scene(scale: 2, dx: -150, dy: -104)
        let zoomOut = scene(scale: 0.5, dx: 75, dy: 52)
        expect(before, right, kind: "pan", key: 0x7b, scale: 1, dx: 60, dy: 0)
        expect(before, left, kind: "pan", key: 0x7c, scale: 1, dx: -60, dy: 0)
        expect(before, zoomIn, kind: "zoom", key: 0x45, scale: 2, dx: -600, dy: -416)
        expect(before, zoomOut, kind: "zoom", key: 0x4e, scale: 0.5, dx: 300, dy: 208)
        for (name, frame) in [("stationary", before), ("opposite pan", left),
                              ("wrong scale", zoomIn), ("unrelated repaint", scene(dx: 93, dy: 37))] {
            reject(before, frame, kind: "pan", key: 0x7b, name: name)
        }
        reject(before, zoomOut, kind: "zoom", key: 0x45, name: "opposite zoom")
        reject(before, right, kind: "zoom", key: 0x45, name: "pan instead of zoom")
        reject(before, scene(scale: 1.2, dx: -30, dy: -20), kind: "zoom", key: 0x45, name: "wrong zoom scale")
        precondition(identifyTransform(before: before, after: right, kind: "pan", key: 0x7b, shift: false) == nil,
                     "Selection arrows cannot qualify as pan input")
        let blank = VisualFrame(width: 300, height: 200, luminance: Array(repeating: 128, count: 60_000))
        reject(blank, blank, kind: "pan", key: 0x7b, name: "no texture")
        var replacedTile = before.luminance
        for y in 60..<100 {
            for x in 60..<100 { replacedTile[y * 300 + x] = right.luminance[y * 300 + x] }
        }
        reject(before, VisualFrame(width: 300, height: 200, luminance: replacedTile),
               kind: "pan", key: 0x7b, name: "one arriving tile")
        var movingWithTile = right.luminance
        for y in 60..<100 {
            for x in 60..<100 { movingWithTile[y * 300 + x] = 128 }
        }
        expect(before, VisualFrame(width: 300, height: 200, luminance: movingWithTile),
               kind: "pan", key: 0x7b, scale: 1, dx: 60, dy: 0)
        let periodic = VisualFrame(width: 300, height: 200,
            luminance: (0..<60_000).map { (($0 % 300) / 6 + ($0 / 300) / 6) % 2 == 0 ? 40 : 210 })
        let shiftedPeriodic = VisualFrame(width: 300, height: 200,
            luminance: (0..<60_000).map { ((($0 % 300) + 3) / 6 + ($0 / 300) / 6) % 2 == 0 ? 40 : 210 })
        reject(periodic, shiftedPeriodic, kind: "pan", key: 0x7b, name: "ambiguous repeated pattern")
        reject(periodic, shiftedPeriodic, kind: "pan", key: 0x7c, name: "ambiguous reversed pattern")
        reject(before, VisualFrame(width: 300, height: 200, luminance: []),
               kind: "pan", key: 0x7b, name: "incomplete frame")
        print("Native Location Map visual transform tests passed")
    }

    static func reject(_ before: VisualFrame, _ after: VisualFrame, kind: String, key: UInt16, name: String) {
        if identifyTransform(before: before, after: after, kind: kind, key: key, shift: kind == "pan") != nil {
            fatalError("Accepted \(name) as \(kind)")
        }
    }

    static func expect(_ before: VisualFrame, _ after: VisualFrame, kind: String, key: UInt16,
                       scale: Double, dx: Double, dy: Double) {
        guard let match = identifyTransform(before: before, after: after, kind: kind, key: key, shift: kind == "pan") else {
            fatalError("Requested \(kind) \(key) was not identified")
        }
        precondition(match.scale == scale && abs(match.dx - dx) <= 4 && abs(match.dy - dy) <= 4,
                     "Wrong visual transform: \(match)")
    }

    static func scene(scale: Double = 1, dx: Double = 0, dy: Double = 0) -> VisualFrame {
        var pixels = [UInt8]()
        for y in 0..<200 {
            for x in 0..<300 {
                let sx = (Double(x) - dx) / scale, sy = (Double(y) - dy) / scale
                // Unequal frequencies and a mixed term avoid repeated tiles.
                let value = 128 + 36 * sin(sx * 0.21) + 31 * cos(sy * 0.27)
                    + 29 * sin(sx * 0.13 + sy * 0.19) + 25 * cos(sx * sy * 0.0017)
                pixels.append(UInt8(max(0, min(255, value))))
            }
        }
        return VisualFrame(width: 300, height: 200, luminance: pixels)
    }
}
