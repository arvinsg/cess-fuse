package fs

import (
	"context"
	"fmt"
	"time"

	"github.com/arvinsg/cess-fuse/pkg/storage"
	"github.com/arvinsg/cess-fuse/pkg/utils"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/sirupsen/logrus"
)

func MountFS(ctx context.Context, cloud storage.ObjectBackend, flags *Flags) (*FileSystem, *fuse.MountedFileSystem, error) {
	// Mount the file system.
	mountCfg := &fuse.MountConfig{
		FSName:                  "CESS",
		Options:                 flags.MountOptions,
		ErrorLogger:             utils.GetStdLogger(utils.NewLogger("fuse"), logrus.ErrorLevel),
		DisableWritebackCaching: true,
	}

	if flags.DebugFuse {
		fuseLog := utils.GetLogger("fuse")
		fuseLog.Level = logrus.DebugLevel
		log.Level = logrus.DebugLevel
		mountCfg.DebugLogger = utils.GetStdLogger(fuseLog, logrus.DebugLevel)
	}

	fs := NewFileSystem(ctx, cloud, flags)
	if fs == nil {
		return nil, nil, fmt.Errorf("initialization file system failed")
	}

	server := fuseutil.NewFileSystemServer(FusePanicLogger{fs})
	mfs, err := fuse.Mount(flags.MountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("mount fail,  err: %v", err)
		return nil, nil, err
	}

	return fs, mfs, nil
}

func TryUnmountFS(mountPoint string) (err error) {
	for i := 0; i < 10; i++ {
		err = fuse.Unmount(mountPoint)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return err
}
