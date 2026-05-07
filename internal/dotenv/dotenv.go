package dotenv

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Load reads .env and then .env.local using github.com/joho/godotenv rules:
// variables already present in the process environment are not overwritten.
//
// It tries, in order:
//  1. The directory containing this process executable (after resolving symlinks)
//  2. The current working directory
//
// That way production can keep .env next to the binary even if WorkingDirectory
// is misconfigured; local dev still works when the binary lives in a temp dir
// but the shell cwd is the repo root.
func Load() {
	loadDir := func(dir string) {
		if dir == "" {
			return
		}
		_ = godotenv.Load(filepath.Join(dir, ".env"))
		_ = godotenv.Load(filepath.Join(dir, ".env.local"))
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		loadDir(filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		loadDir(wd)
	}
}
