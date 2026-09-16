# Image fixtures

`test_exif.heic` tests production rejection of unsupported HEIC images and the
candidate sandboxed helper's ordinary metadata/container-rotation behavior.
It is a synthetic TestCam/Model123 fixture, not a camera photograph. It contains
image data, not decoder code. It was copied from
[gen2brain/heic](https://github.com/gen2brain/heic/tree/v0.7.1/testdata),
whose [MIT license](https://github.com/gen2brain/heic/blob/v0.7.1/LICENSE)
is retained below. The HEIC decoder is not a PicFetch dependency.

```text
MIT License

Copyright (c) 2024 gen2brain

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
