// misc helpers
package main

import (
	"os"
	"strings"
)

func FullPath(name string) string {
	return os.ExpandEnv(strings.Replace(name, "~", os.Getenv("HOME"), 1))
}
