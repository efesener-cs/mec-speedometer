// mec-speedometer reads Faith's horizontal speed from a running Linux game process.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const defaultModule = "MirrorsEdgeCatalyst.exe"

func main() {
	processName := flag.String("process", defaultModule, "game executable name to find under /proc")
	pid := flag.Int("pid", 0, "attach to this process ID instead of searching by name")
	module := flag.String("module", defaultModule, "module name whose mapping contains the pointer offset")
	baseOffset := flag.Uint64("base-offset", 0x023DD6F8, "module-relative address of the speed pointer (hex accepted)")
	interval := flag.Duration("interval", 100*time.Millisecond, "sample interval")
	decimals := flag.Int("decimals", 2, "digits after the decimal point (0-6)")
	unit := flag.String("unit", "m/s", "display unit: m/s, km/h, or mph")
	flag.Parse()

	if *interval < 10*time.Millisecond {
		fatal("interval must be at least 10ms")
	}
	if *decimals < 0 || *decimals > 6 {
		fatal("decimals must be between 0 and 6")
	}
	factor, suffix, ok := unitFactor(*unit)
	if !ok {
		fatal("unit must be m/s, km/h, or mph")
	}
	if *pid < 0 {
		fatal("pid cannot be negative")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Println("MEC Speedometer — read-only process memory; Ctrl+C to quit")
	var lastPID int
	var reader *gameReader
	for ctx.Err() == nil {
		if reader == nil || !reader.alive() {
			if reader != nil {
				reader.close()
				reader = nil
			}
			p, err := findGame(*pid, *processName)
			if err != nil {
				fmt.Printf("\rWaiting for %s... (%s)\033[K", *processName, err)
				if !sleep(ctx, time.Second) {
					break
				}
				continue
			}
			r, err := attach(p, *module, uint64(*baseOffset))
			if err != nil {
				fmt.Printf("\rFound PID %d, cannot read it: %v\033[K", p, err)
				if !sleep(ctx, time.Second) {
					break
				}
				continue
			}
			reader = r
			lastPID = p
			fmt.Printf("\nAttached to PID %d (%s)\n", p, filepath.Base(r.exe))
		}

		speed, err := reader.speed()
		if err != nil {
			fmt.Printf("\rPID %d: memory read failed: %v\033[K", lastPID, err)
			reader.close()
			reader = nil
		} else {
			if math.IsNaN(float64(speed)) || math.IsInf(float64(speed), 0) || math.Abs(float64(speed)) > 10000 {
				fmt.Printf("\rPID %d: invalid speed value\033[K", lastPID)
			} else {
				fmt.Printf("\r%s %s\033[K", formatSpeed(float64(speed)*factor, *decimals), suffix)
			}
		}
		if !sleep(ctx, *interval) {
			break
		}
	}
	if reader != nil {
		reader.close()
	}
	fmt.Println("\nStopped.")
}

func fatal(message string) { fmt.Fprintln(os.Stderr, "error:", message); os.Exit(2) }

func unitFactor(unit string) (float64, string, bool) {
	switch strings.ToLower(unit) {
	case "m/s", "ms":
		return 1, "m/s", true
	case "km/h", "kmh":
		return 3.6, "km/h", true
	case "mph":
		return 2.2369362921, "mph", true
	default:
		return 0, "", false
	}
}

func formatSpeed(value float64, decimals int) string { return fmt.Sprintf("%.*f", decimals, value) }

func sleep(ctx context.Context, duration time.Duration) bool {
	t := time.NewTimer(duration)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

var errNoProcess = errors.New("process not found")
