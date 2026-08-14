# Streamy

A library that reads RGBA32 image data from a file using libav. And it loops. At least it's better than my old FFmpeg pipe solution.

## Notes

I have not yet determined if this works properly on little-endian systems. It probably depends on how the returned array is used, but I'd like to make sure. I know sdl should handle it, so ImageSort should be fine, but raylib...

## Legal notes

While the source code of this library is licensed under the zlib License, this library links with FFmpeg's libraries, which may be licensed under the GPL depending on compile options. As such, the compiled version may be licensed under the GPL. If you wish to avoid this, consider using an LGPL-compliant version of the libav libraries.
