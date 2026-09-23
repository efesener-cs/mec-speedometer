//go:build linux

package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type gameReader struct {
	pid     int
	exe     string
	mem     *os.File
	address uint64
}

func findGame(requestedPID int, processName string) (int, error) {
	if requestedPID > 0 {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", requestedPID)); err != nil {
			return 0, err
		}
		return requestedPID, nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		exe, _ := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		cmdline, _ := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		comm, _ := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		base := filepath.Base(strings.TrimSuffix(exe, " (deleted)"))
		if strings.EqualFold(base, processName) || strings.EqualFold(strings.TrimSpace(string(comm)), processName) || strings.Contains(strings.ToLower(string(cmdline)), strings.ToLower(processName)) {
			return pid, nil
		}
	}
	return 0, errNoProcess
}

func attach(pid int, module string, pointerOffset uint64) (*gameReader, error) {
	proc := fmt.Sprintf("/proc/%d", pid)
	exe, err := os.Readlink(filepath.Join(proc, "exe"))
	if err != nil {
		return nil, fmt.Errorf("read executable link: %w", err)
	}
	if strings.HasSuffix(exe, " (deleted)") {
		exe = strings.TrimSuffix(exe, " (deleted)")
	}
	maps, err := os.ReadFile(filepath.Join(proc, "maps"))
	if err != nil {
		return nil, fmt.Errorf("read process mappings: %w", err)
	}
	base, err := moduleBase(string(maps), module)
	if err != nil {
		return nil, err
	}
	mem, err := os.Open(filepath.Join(proc, "mem"))
	if err != nil {
		return nil, fmt.Errorf("open process memory (check ptrace/Yama permissions): %w", err)
	}
	return &gameReader{pid: pid, exe: exe, mem: mem, address: base + pointerOffset}, nil
}

func moduleBase(maps, module string) (uint64, error) {
	for _, line := range strings.Split(maps, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		name := strings.TrimSuffix(fields[5], " (deleted)")
		if !strings.EqualFold(filepath.Base(name), module) {
			continue
		}
		bounds := strings.Split(fields[0], "-")
		if len(bounds) != 2 {
			continue
		}
		start, e1 := strconv.ParseUint(bounds[0], 16, 64)
		offset, e2 := strconv.ParseUint(fields[2], 16, 64)
		if e1 == nil && e2 == nil && start >= offset {
			return start - offset, nil
		}
	}
	return 0, fmt.Errorf("module %q not found in process mappings", module)
}

func (r *gameReader) readUint64(address uint64) (uint64, error) {
	var data [8]byte
	n, err := r.mem.ReadAt(data[:], int64(address))
	if err != nil {
		return 0, fmt.Errorf("read pointer at 0x%x: %w", address, err)
	}
	if n != len(data) {
		return 0, fmt.Errorf("short pointer read at 0x%x", address)
	}
	return binary.LittleEndian.Uint64(data[:]), nil
}

func (r *gameReader) speed() (float32, error) {
	address, err := r.readUint64(r.address)
	if err != nil {
		return 0, err
	}
	if address == 0 || address > uint64(1<<63-1)-0x1e8 {
		return 0, fmt.Errorf("invalid pointer 0x%x", address)
	}
	address, err = r.readUint64(address + 0x1e8)
	if err != nil {
		return 0, err
	}
	if address == 0 || address > uint64(1<<63-1)-0x10 {
		return 0, fmt.Errorf("invalid pointer 0x%x", address)
	}
	address, err = r.readUint64(address + 0x10)
	if err != nil {
		return 0, err
	}
	if address == 0 || address > uint64(1<<63-1)-0x438 {
		return 0, fmt.Errorf("invalid pointer 0x%x", address)
	}
	var data [4]byte
	n, err := r.mem.ReadAt(data[:], int64(address+0x438))
	if err != nil {
		return 0, fmt.Errorf("read speed at 0x%x: %w", address+0x438, err)
	}
	if n != len(data) {
		return 0, fmt.Errorf("short speed read at 0x%x", address+0x438)
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(data[:])), nil
}

func (r *gameReader) alive() bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", r.pid))
	return err == nil
}
func (r *gameReader) close() {
	if r.mem != nil {
		_ = r.mem.Close()
	}
}
