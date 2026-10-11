# Trane in the terminal

Run from the repository root without linking or starting the desktop app:

```sh
make trane
```

A shaded, stylized 3D Trane head makes one smooth three-second turn above the
app's help text, then reveals an ASCII PicFetch wordmark and exits. This is the
same renderer used by `picfetch --help`. Only ASCII characters form the picture;
ANSI sequences move the cursor. Ctrl+C skips the animation and prints plain help.

```sh
./bin/picfetch --help
./bin/picfetch --help > /tmp/picfetch-help.txt
```

Use a dark terminal with a light monospace font, preferably at least 100 columns
by 55 rows. Artwork scales to the available space. If the complete wrapped help,
wordmark and at least ten rows of artwork will not fit, plain help prints
immediately. Piped output and `TERM=dumb` also print ordinary help immediately.
Resizing to a smaller terminal ends animation cleanly, preserving the full help.

Rotation uses elapsed time at up to 30 FPS, with smooth acceleration and braking.
The camera stays fixed; depth-tested ellipsoids form the head, ears, muzzle, eyes
and tongue. Frame 360 equals frame 0; the underlying turn is seamlessly loopable.
Normal completion leaves the portrait, wordmark and help in terminal scrollback.

This is original procedural geometry inspired by the retained Trane artwork,
not a recovered mesh. It uses Go and the existing `golang.org/x/term` dependency,
starts no render workers, and returns before desktop startup or storage access.
