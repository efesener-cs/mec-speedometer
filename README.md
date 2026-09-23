# MEC Speedometer for Linux

A read-only Go implementation of the original Mirror's Edge Catalyst speedometer's memory-reading method. The current first release displays Faith's horizontal speed in a terminal; it does not write game memory or implement a desktop overlay yet.

## Build and run

```sh
go build -o mec-speedometer .
./mec-speedometer
```

Run the game first. By default the program looks for `MirrorsEdgeCatalyst.exe`, reads the original pointer chain, and samples every 100 ms. Stop it with Ctrl+C.

```sh
./mec-speedometer -unit km/h -decimals 1
./mec-speedometer -pid 12345 -interval 50ms
./mec-speedometer -base-offset 0x023DD6F8
```

Supported units are `m/s` (default), `km/h`, and `mph`. The `-process` and `-module` flags can override the game executable/module names. `-base-offset` selects the module-relative pointer location; it may need updating for a different game build.

## Linux permissions

Linux may deny `/proc/<pid>/mem` access because of ptrace restrictions (including Yama), process ownership, or container/sandbox policy. Run the game and tool as the same user first. If access remains denied, consult your distribution's ptrace policy; this program does not request elevated privileges or change system settings.

## Compatibility

The pointer chain and `0x023DD6F8` offset come from the bundled original Windows project. They have not yet been verified against the current Linux/Proton game build. A module mapping or pointer-chain error is reported while the tool waits/retries. Speed units follow the original raw game value, assumed to be metres per second.

The original Windows source and bundled assets are kept in `original/` as reference material.
