package fs

import (
	"mime"
	"os"
	"strings"
	"time"

	"github.com/arvinsg/cess-fuse/pkg/utils"
)

var (
	log     = utils.GetLogger("main")
	fuseLog = utils.GetLogger("fuse")
)

type Flags struct {
	// File system
	MountOptions      map[string]string
	MountPoint        string
	MountPointArg     string
	MountPointCreated string

	Cache    []string
	DirMode  os.FileMode
	FileMode os.FileMode
	Uid      uint32
	Gid      uint32

	// Common Backend Flags
	UseContentType bool
	Endpoint       string

	// Tuning
	ExplicitDir  bool
	StatCacheTTL time.Duration
	TypeCacheTTL time.Duration
	HTTPTimeout  time.Duration

	// Debugging
	DebugFuse  bool
	Foreground bool
}

func (c *Flags) GetMimeType(fileName string) (retMime *string) {
	if !c.UseContentType {
		return nil
	}
	dotPosition := strings.LastIndex(fileName, ".")
	if dotPosition == -1 {
		return nil
	}
	mimeType := mime.TypeByExtension(fileName[dotPosition:])
	if mimeType == "" {
		return nil
	}
	semicolonPosition := strings.LastIndex(mimeType, ";")
	if semicolonPosition == -1 {
		return &mimeType
	}
	s := mimeType[:semicolonPosition]
	retMime = &s

	return retMime
}

func (c *Flags) Cleanup() {
	if c.MountPointCreated != "" && c.MountPointCreated != c.MountPointArg {
		err := os.Remove(c.MountPointCreated)
		if err != nil {
			log.Errorf("rmdir %v = %v", c.MountPointCreated, err)
		}
	}
}
