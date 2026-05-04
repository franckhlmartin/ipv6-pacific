package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"os/user"
)

// applyDataOwnership chowns everything under dataDir when COLLECTOR_DATA_USER is set.
// Use when the collector runs as root but the web server reads DATA_DIR as a normal user (e.g. franck).
// COLLECTOR_DATA_GROUP is optional; if empty, the user's primary group is used.
func applyDataOwnership(dataDir string) {
	name := strings.TrimSpace(os.Getenv("COLLECTOR_DATA_USER"))
	if name == "" {
		return
	}
	u, err := user.Lookup(name)
	if err != nil {
		log.Printf("[collector] COLLECTOR_DATA_USER=%q: %v", name, err)
		return
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		log.Printf("[collector] COLLECTOR_DATA_USER uid: %v", err)
		return
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		log.Printf("[collector] COLLECTOR_DATA_USER gid: %v", err)
		return
	}
	if g := strings.TrimSpace(os.Getenv("COLLECTOR_DATA_GROUP")); g != "" {
		grp, err := user.LookupGroup(g)
		if err != nil {
			log.Printf("[collector] COLLECTOR_DATA_GROUP=%q: %v", g, err)
			return
		}
		gid, err = strconv.Atoi(grp.Gid)
		if err != nil {
			log.Printf("[collector] COLLECTOR_DATA_GROUP gid: %v", err)
			return
		}
	}

	err = filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(path, uid, gid)
	})
	if err != nil {
		log.Printf("[collector] chown %s to uid=%d gid=%d: %v", dataDir, uid, gid, err)
		return
	}
	log.Printf("[collector] ownership under %s set to %s (uid=%d gid=%d)", dataDir, name, uid, gid)
}
