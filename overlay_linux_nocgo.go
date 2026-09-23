//go:build linux && !cgo

package main

type speedOverlay struct{}

func newSpeedOverlay() *speedOverlay     { return nil }
func (*speedOverlay) update(int, string) {}
func (*speedOverlay) close()             {}
