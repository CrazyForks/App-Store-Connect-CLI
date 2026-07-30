//go:build !darwin && !linux

package rootfs

import "os"

func copyReplacementMetadata(destination, _ *os.File, info os.FileInfo) error {
	return destination.Chmod(info.Mode().Perm())
}
