//go:build !linux && !windows

package fsbridge

import (
	"fmt"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/database"
)

// MountOptions configures the FUSE mount behavior.
type MountOptions struct {
	MountPoint   string
	ReadOnly     bool
	AllowOther   bool
	Debug        bool
	ShowChunks   bool
	ShowMetadata bool
}

// FuseMount is a stub FUSE mount for unsupported platforms.
type FuseMount struct {
	opts MountOptions
}

// NewFuseMount creates a stub mount that will error at runtime.
func NewFuseMount(_ *database.DB, _ config.StorageSource, opts MountOptions) *FuseMount {
	return &FuseMount{opts: opts}
}

// Mount returns an error on unsupported platforms.
func (m *FuseMount) Mount() error {
	return fmt.Errorf("FUSE mount is not supported on this platform; use 'stratafs serve' instead")
}

// Unmount is a no-op on unsupported platforms.
func (m *FuseMount) Unmount() error {
	return nil
}

// IsMounted always returns false on unsupported platforms.
func (m *FuseMount) IsMounted() bool {
	return false
}
