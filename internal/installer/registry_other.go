//go:build !windows

package installer

import "errors"

var errNoRegistry = errors.New("the registry exists only on Windows")

func setRegistryDefault(key, value string) error { return errNoRegistry }

func deleteRegistryKey(key string) error { return errNoRegistry }
