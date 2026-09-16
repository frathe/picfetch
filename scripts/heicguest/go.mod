module github.com/frathe/picfetch/scripts/heicguest

go 1.27.1

require (
 github.com/frathe/picfetch v0.0.0
 github.com/gen2brain/h265 v0.2.2
)

replace github.com/frathe/picfetch => ../..

replace github.com/gen2brain/h265 => ../../third_party/h265
