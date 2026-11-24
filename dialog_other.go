//go:build !darwin

package main

// Dialog boxes are only used by the macOS app; elsewhere the installer runs
// in a console window.
func useDialogs([]string) bool             { return false }
func dialogAsk(string) (string, bool)      { return "", false }
func dialogFinish(output string, code int) {}
