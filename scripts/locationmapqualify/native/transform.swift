// Pure pixel registration, shared by the native observer and portable tests.
// Coordinates are in the observer's four-pixel luminance samples.
struct VisualFrame {
    let width: Int
    let height: Int
    let luminance: [UInt8]

    fileprivate func pixel(_ x: Double, _ y: Double) -> Double {
        let ix = Int(x), iy = Int(y), fx = x - Double(ix), fy = y - Double(iy)
        let top = Double(luminance[iy * width + ix]) * (1 - fx) + Double(luminance[iy * width + ix + 1]) * fx
        let bottom = Double(luminance[(iy + 1) * width + ix]) * (1 - fx) + Double(luminance[(iy + 1) * width + ix + 1]) * fx
        return top * (1 - fy) + bottom * fy
    }
}

struct VisualTransform: Encodable {
    let method = "patch-grid-v1"
    let scale: Double
    let dx: Double
    let dy: Double
    let matches: Int
    let tested: Int
}

func identifyTransform(before: VisualFrame, after: VisualFrame, kind: String, key: UInt16, shift: Bool) -> VisualTransform? {
    let requestedScale: Double
    switch (kind, key, shift) {
    case ("pan", 0x7b, true), ("pan", 0x7c, true): requestedScale = 1
    case ("zoom", 0x45, false): requestedScale = 2
    case ("zoom", 0x4e, false): requestedScale = 0.5
    default: return nil
    }
    guard before.width == after.width, before.height == after.height,
          before.width >= 100, before.width <= 1024, before.height >= 80, before.height <= 1024,
          before.luminance.count == before.width * before.height,
          after.luminance.count == after.width * after.height else { return nil }
    let registration = Registration(before: before, after: after)
    guard registration.patches.count >= 8 else { return nil }
    let candidates = registration.candidates()
    guard let best = candidates.first, best.coverage >= 0.65, best.matches >= 8,
          best.spread, best.scale == requestedScale else { return nil }
    if kind == "pan" && (key == 0x7b ? best.dx <= 0 : best.dx >= 0) { return nil }
    // Competing displacements or scales must be clearly worse. This also
    // rejects unchanged frames and periodic/checkerboard map placeholders.
    for other in candidates.dropFirst() {
        if other.coverage >= best.coverage - 0.08 && other.matches >= 8 { return nil }
    }
    return VisualTransform(scale: best.scale, dx: best.dx * 4, dy: best.dy * 4,
                           matches: best.matches, tested: best.tested)
}

private struct Patch {
    let x: Double
    let y: Double
    let values: [Double]
}

private struct Score {
    let scale: Double
    let dx: Double
    let dy: Double
    let matches: Int
    let tested: Int
    let error: Double
    let spread: Bool
    var coverage: Double { tested == 0 ? 0 : Double(matches) / Double(tested) }

    func near(_ other: Score) -> Bool {
        scale == other.scale && abs(dx - other.dx) <= 3 && abs(dy - other.dy) <= 3
    }

    func better(than other: Score) -> Bool {
        if coverage != other.coverage { return coverage > other.coverage }
        return error < other.error
    }
}

private struct Registration {
    let after: VisualFrame
    let patches: [Patch]
    let minX: Double, maxX: Double, minY: Double, maxY: Double

    init(before: VisualFrame, after: VisualFrame) {
        self.after = after
        minX = Double(before.width) * 0.1
        maxX = Double(before.width) * 0.9
        minY = Double(before.height) * 0.2
        maxY = Double(before.height) * 0.8
        var patches = [Patch]()
        // Distributed 5x5 patches, not a single thumbnail or changing tile.
        for y in stride(from: Int(minY) + 4, to: Int(maxY) - 4, by: 12) {
            for x in stride(from: Int(minX) + 4, to: Int(maxX) - 4, by: 12) {
                var values = [Double]()
                for py in -2...2 {
                    for px in -2...2 { values.append(Double(before.luminance[(y + py) * before.width + x + px])) }
                }
                if values.max()! - values.min()! >= 45 {
                    patches.append(Patch(x: Double(x), y: Double(y), values: values))
                }
            }
        }
        self.patches = patches
    }

    func score(scale: Double, dx: Double, dy: Double, coarse: Bool) -> Score {
        var tested = 0, matches = 0, totalError = 0.0
        var left = maxX, right = minX, top = maxY, bottom = minY
        for patch in patches {
            let x = patch.x * scale + dx, y = patch.y * scale + dy
            guard x - 2 * scale >= minX, x + 2 * scale < maxX,
                  y - 2 * scale >= minY, y + 2 * scale < maxY else { continue }
            tested += 1
            var error = 0.0, samples = 0
            for py in stride(from: -2, through: 2, by: coarse ? 2 : 1) {
                for px in stride(from: -2, through: 2, by: coarse ? 2 : 1) {
                    error += abs(patch.values[(py + 2) * 5 + px + 2]
                        - after.pixel(x + Double(px) * scale, y + Double(py) * scale))
                    samples += 1
                }
            }
            error /= Double(samples)
            if error <= 12 {
                matches += 1
                totalError += error
                left = min(left, x); right = max(right, x)
                top = min(top, y); bottom = max(bottom, y)
            }
        }
        return Score(scale: scale, dx: dx, dy: dy, matches: matches, tested: tested,
                     error: matches == 0 ? 255 : totalError / Double(matches),
                     spread: right - left >= (maxX - minX) * 0.35 && bottom - top >= (maxY - minY) * 0.35)
    }

    func candidates() -> [Score] {
        var coarse = [Score]()
        // Search both directions and scales, independently of the requested key.
        // Capture is fixed at 1200x800. These bounds tolerate window chrome and
        // capture scaling around the 60 logical-pixel pan and centered zoom.
        coarse.append(score(scale: 1, dx: 0, dy: 0, coarse: false))
        for dx in stride(from: -40, through: 40, by: 2) where abs(dx) >= 6 {
            for dy in -2...2 { coarse.append(score(scale: 1, dx: Double(dx), dy: Double(dy), coarse: true)) }
        }
        for scale in [0.5, 2.0] {
            let centerDX = (1 - scale) * Double(after.width) / 2
            let centerDY = (1 - scale) * Double(after.height) / 2
            for dx in stride(from: -12, through: 12, by: 2) {
                for dy in stride(from: -20, through: 20, by: 2) {
                    coarse.append(score(scale: scale, dx: centerDX + Double(dx), dy: centerDY + Double(dy), coarse: true))
                }
            }
        }
        coarse.sort { $0.better(than: $1) }
        var peaks = [Score]()
        for candidate in coarse where candidate.coverage >= 0.45 && candidate.matches >= 8 {
            if !peaks.contains(where: { $0.near(candidate) }) { peaks.append(candidate) }
            if peaks.count == 12 { break }
        }
        var refined = [Score]()
        for peak in peaks {
            var best = score(scale: peak.scale, dx: peak.dx, dy: peak.dy, coarse: false)
            if peak.scale != 1 || peak.dx != 0 {
                for x in -3...3 {
                    for y in -3...3 {
                        let candidate = score(scale: peak.scale, dx: peak.dx + Double(x) / 2,
                                              dy: peak.dy + Double(y) / 2, coarse: false)
                        if candidate.better(than: best) { best = candidate }
                    }
                }
            }
            refined.append(best)
        }
        refined.sort { $0.better(than: $1) }
        var distinct = [Score]()
        for candidate in refined where !distinct.contains(where: { $0.near(candidate) }) {
            distinct.append(candidate)
        }
        return distinct
    }
}
