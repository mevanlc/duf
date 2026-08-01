package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dereferenceMountPointPatterns adds the backing mount point of every symlink
// matched by the supplied filesystem patterns.
func dereferenceMountPointPatterns(mounts []Mount, patterns map[string]struct{}) (map[string]struct{}, error) {
	var mountPoints []string
	for pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid mount point pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			info, err := os.Lstat(match)
			if err != nil {
				return nil, fmt.Errorf("inspect mount point match %q: %w", match, err)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				continue
			}

			matchedMounts, err := findMounts(mounts, match)
			if err != nil {
				return nil, fmt.Errorf("dereference mount point match %q: %w", match, err)
			}
			for _, mount := range matchedMounts {
				mountPoints = append(mountPoints, mount.Mountpoint)
			}
		}
	}

	for _, mountPoint := range mountPoints {
		patterns[mountPoint] = struct{}{}
	}

	return patterns, nil
}

func findMounts(mounts []Mount, path string) ([]Mount, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(path)
	if err != nil {
		return nil, err
	}

	var m []Mount
	for _, v := range mounts {
		if path == v.Device {
			return []Mount{v}, nil
		}

		if strings.HasPrefix(path, v.Mountpoint) {
			var nm []Mount

			// keep all entries that are as close or closer to the target
			for _, mv := range m {
				if len(mv.Mountpoint) >= len(v.Mountpoint) {
					nm = append(nm, mv)
				}
			}
			m = nm

			// add entry only if we didn't already find something closer
			if len(nm) == 0 || len(v.Mountpoint) >= len(nm[0].Mountpoint) {
				m = append(m, v)
			}
		}
	}

	return m, nil
}

func deviceType(m Mount) string {
	if isNetworkFs(m) {
		return networkDevice
	}
	if isSpecialFs(m) {
		return specialDevice
	}
	if isFuseFs(m) {
		return fuseDevice
	}

	return localDevice
}

// remote: [ "nfs", "smbfs", "cifs", "ncpfs", "afs", "coda", "ftpfs", "mfs", "sshfs", "fuse.sshfs", "nfs4" ]
// special: [ "tmpfs", "devpts", "devtmpfs", "proc", "sysfs", "usbfs", "devfs", "fdescfs", "linprocfs" ]
