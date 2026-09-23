# MEC Speedometer for Linux

A read-only Go implementation of the original Mirror's Edge Catalyst speedometer's memory-reading method. It displays Faith's horizontal speed in a small, transparent, always-on-top X11 window and can wrap the Steam game command. It never writes game memory.

## Build and run

```sh
go build -o mec-speedometer .
./mec-speedometer
```

Run the game first. By default the program looks for a process/module name beginning with `MirrorsEdgeCata`, reads the original pointer chain, and samples every 100 ms. The prefix match accepts the truncated `Cata` spelling and the full `Catalyst` name. Stop it with Ctrl+C.

```sh
./mec-speedometer -unit km/h -decimals 1
./mec-speedometer -pid 12345 -interval 50ms
./mec-speedometer -base-offset 0x023DD6F8
./mec-speedometer -terminal
```

Supported units are `m/s` (default), `km/h`, and `mph`. The `-process` and `-module` flags can override the game executable/module names. They accept a prefix, so `-module MirrorsEdgeCata.exe` matches `MirrorsEdgeCatalyst.exe`. `-base-offset` selects the module-relative pointer location; it may need updating for a different game build.

## Steam launch options

Build the binary at a stable path, then open the game's **Properties → General → Launch Options** and enter:

```text
/home/your-user/Projects/mec-speedometer/mec-speedometer --steam-launch -- %command%
```

Replace the path with the absolute path to this executable. Steam's `%command%` is started as a child process; the speedometer follows the matching game process, displays the overlay, and exits when the game exits. To use the terminal instead, add `--terminal` before `--`.

The overlay requires CGO, GTK 3 development files, and X11/XWayland access. GTK is explicitly pinned to its X11 backend so it can follow and overlay the XWayland game window even in a Wayland desktop session. The speed window is click-through and does not take keyboard focus. If XWayland or its display authorization is unavailable, the app falls back to terminal output.

## Linux permissions

Linux may deny `/proc/<pid>/mem` access because of ptrace restrictions (including Yama), process ownership, or container/sandbox policy. Run the game and tool as the same user first. If access remains denied, consult your distribution's ptrace policy; this program does not request elevated privileges or change system settings.

## Compatibility

The pointer chain and `0x023DD6F8` offset come from the bundled original Windows project. They have not yet been verified against the current Linux/Proton game build. A module mapping or pointer-chain error is reported while the tool waits/retries. Speed units follow the original raw game value, assumed to be metres per second. The overlay tracks the game window through its X11 `_NET_WM_PID` property; window managers or Wine builds that omit this property may leave it at its default position.

The original Windows source and bundled assets are kept in `original/` as reference material.
