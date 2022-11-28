package fs

import (
	"context"
	"runtime/debug"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type FusePanicLogger struct {
	Fs fuseutil.FileSystem
}

func LogPanic(err *error) {
	if e := recover(); e != nil {
		log.Errorf("stacktrace from panic: %v \n"+string(debug.Stack()), e)
		if *err == nil {
			*err = fuse.EIO
		}
	}
}

func (fs FusePanicLogger) StatFS(ctx context.Context, op *fuseops.StatFSOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.StatFS(ctx, op)
}
func (fs FusePanicLogger) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.LookUpInode(ctx, op)
}
func (fs FusePanicLogger) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.GetInodeAttributes(ctx, op)
}
func (fs FusePanicLogger) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.SetInodeAttributes(ctx, op)
}
func (fs FusePanicLogger) Fallocate(ctx context.Context, op *fuseops.FallocateOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.Fallocate(ctx, op)
}
func (fs FusePanicLogger) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ForgetInode(ctx, op)
}
func (fs FusePanicLogger) MkDir(ctx context.Context, op *fuseops.MkDirOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.MkDir(ctx, op)
}
func (fs FusePanicLogger) MkNode(ctx context.Context, op *fuseops.MkNodeOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.MkNode(ctx, op)
}
func (fs FusePanicLogger) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.CreateFile(ctx, op)
}
func (fs FusePanicLogger) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.CreateLink(ctx, op)
}
func (fs FusePanicLogger) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.CreateSymlink(ctx, op)
}
func (fs FusePanicLogger) Rename(ctx context.Context, op *fuseops.RenameOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.Rename(ctx, op)
}
func (fs FusePanicLogger) RmDir(ctx context.Context, op *fuseops.RmDirOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.RmDir(ctx, op)
}
func (fs FusePanicLogger) Unlink(ctx context.Context, op *fuseops.UnlinkOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.Unlink(ctx, op)
}
func (fs FusePanicLogger) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.OpenDir(ctx, op)
}
func (fs FusePanicLogger) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ReadDir(ctx, op)
}
func (fs FusePanicLogger) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ReleaseDirHandle(ctx, op)
}
func (fs FusePanicLogger) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.OpenFile(ctx, op)
}
func (fs FusePanicLogger) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ReadFile(ctx, op)
}
func (fs FusePanicLogger) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.WriteFile(ctx, op)
}
func (fs FusePanicLogger) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.SyncFile(ctx, op)
}
func (fs FusePanicLogger) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.FlushFile(ctx, op)
}
func (fs FusePanicLogger) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ReleaseFileHandle(ctx, op)
}
func (fs FusePanicLogger) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ReadSymlink(ctx, op)
}
func (fs FusePanicLogger) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.RemoveXattr(ctx, op)
}
func (fs FusePanicLogger) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.GetXattr(ctx, op)
}
func (fs FusePanicLogger) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.ListXattr(ctx, op)
}
func (fs FusePanicLogger) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) (err error) {
	defer LogPanic(&err)
	return fs.Fs.SetXattr(ctx, op)
}

func (fs FusePanicLogger) Destroy() {
	fs.Fs.Destroy()
}
