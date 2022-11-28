package fs

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arvinsg/cess-fuse/pkg/storage"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// FileSystem is a Filey System written in Go. All the backend data is
// stored on CESS as is. It's a Filey System instead of a File System
// because it makes minimal effort at being POSIX compliant.

type FileSystem struct {
	fuseutil.NotImplementedFileSystem

	flags *Flags

	cloud storage.ObjectBackend

	umask uint32

	rootAttrs InodeAttributes

	bufferPool *BufferPool

	// A lock protecting the state of the file system struct itself (distinct
	// from per-inode locks). Make sure to see the notes on lock ordering above.
	mu sync.RWMutex

	// The next inode ID to hand out. We assume that this will never overflow,
	// since even if we were handing out inode IDs at 4 GHz, it would still take
	// over a century to do so.
	//
	// GUARDED_BY(mu)
	nextInodeID fuseops.InodeID

	// The collection of live inodes, keyed by inode ID. No ID less than
	// fuseops.RootInodeID is ever used.
	//
	// INVARIANT: For all keys k, fuseops.RootInodeID <= k < nextInodeID
	// INVARIANT: For all keys k, inodes[k].ID() == k
	// INVARIANT: inodes[fuseops.RootInodeID] is missing or of type inode.DirInode
	// INVARIANT: For all v, if IsDirName(v.Name()) then v is inode.DirInode
	//
	// GUARDED_BY(mu)
	inodes map[fuseops.InodeID]*Inode

	nextHandleID fuseops.HandleID
	dirHandles   map[fuseops.HandleID]*DirHandle

	fileHandles map[fuseops.HandleID]*FileHandle

	replicators *Ticket
	restorers   *Ticket

	forgotCnt uint32
}

func NewFileSystem(ctx context.Context, cloud storage.ObjectBackend, flags *Flags) *FileSystem {
	// Set up the basic struct.
	fs := &FileSystem{
		flags: flags,
		umask: 0122,
		cloud: cloud,
	}

	now := time.Now()
	fs.rootAttrs = InodeAttributes{
		Size:  4096,
		Mtime: now,
	}

	fs.bufferPool = BufferPool{}.Init()
	fs.nextInodeID = fuseops.RootInodeID + 1
	fs.inodes = make(map[fuseops.InodeID]*Inode)
	root := NewInode(fs, nil, PString(""))
	root.Id = fuseops.RootInodeID
	root.ToDir()
	root.dir.cloud = cloud
	root.Attributes.Mtime = fs.rootAttrs.Mtime

	fs.inodes[fuseops.RootInodeID] = root
	fs.addDotAndDotDot(root)

	fs.nextHandleID = 1
	fs.dirHandles = make(map[fuseops.HandleID]*DirHandle)

	fs.fileHandles = make(map[fuseops.HandleID]*FileHandle)

	fs.replicators = Ticket{Total: 16}.Init()
	fs.restorers = Ticket{Total: 20}.Init()

	return fs
}

// from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func RandStringBytesMaskImprSrc(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	src := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func (fs *FileSystem) SigUsr1() {
	fs.mu.RLock()

	log.Infof("forgot %v inodes", fs.forgotCnt)
	log.Infof("%v inodes", len(fs.inodes))
	fs.mu.RUnlock()
	debug.FreeOSMemory()
}

// Find the given inode. Panic if it doesn't exist.
//
// RLOCKS_REQUIRED(fs.mu)
func (fs *FileSystem) getInodeOrDie(id fuseops.InodeID) (inode *Inode) {
	inode = fs.inodes[id]
	if inode == nil {
		panic(fmt.Sprintf("Unknown inode: %v", id))
	}

	return
}

type Mount struct {
	// Mount Point relative to FileSystem's root mount.
	name    string
	cloud   storage.ObjectBackend
	prefix  string
	mounted bool
}

func (fs *FileSystem) mount(mp *Inode, b *Mount) {
	if b.mounted {
		return
	}

	name := strings.Trim(b.name, "/")

	// create path for the mount. AttrTime is set to TIME_MAX so
	// they will never expire and be removed. But DirTime is not
	// so we will still consult the underlining cloud for listing
	// (which will then be merged with the cached result)

	for {
		idx := strings.Index(name, "/")
		if idx == -1 {
			break
		}
		dirName := name[0:idx]
		name = name[idx+1:]

		mp.mu.Lock()
		dirInode := mp.findChildUnlocked(dirName)
		if dirInode == nil {
			fs.mu.Lock()

			dirInode = NewInode(fs, mp, &dirName)
			dirInode.ToDir()
			dirInode.AttrTime = TIMEMAX

			fs.insertInode(mp, dirInode)
			fs.mu.Unlock()
		}
		mp.mu.Unlock()
		mp = dirInode
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	prev := mp.findChildUnlocked(name)
	if prev == nil {
		mountInode := NewInode(fs, mp, &name)
		mountInode.ToDir()
		mountInode.dir.cloud = b.cloud
		mountInode.dir.mountPrefix = b.prefix
		mountInode.AttrTime = TIMEMAX

		fs.mu.Lock()
		defer fs.mu.Unlock()

		fs.insertInode(mp, mountInode)
		prev = mountInode
	} else {
		if !prev.isDir() {
			panic(fmt.Sprintf("inode %v is not a directory", *prev.FullName()))
		}

		// This inode might have some cached data from a parent mount.
		// Clear this cache by resetting the DirTime.
		// Note: resetDirTimeRec should be called without holding the lock.
		prev.resetDirTimeRec()
		prev.mu.Lock()
		defer prev.mu.Unlock()
		prev.dir.cloud = b.cloud
		prev.dir.mountPrefix = b.prefix
		prev.AttrTime = TIMEMAX

	}
	fuseLog.Infof("mounted /%v", *prev.FullName())
	b.mounted = true
}

func (fs *FileSystem) MountAll(mounts []*Mount) {
	fs.mu.RLock()
	root := fs.getInodeOrDie(fuseops.RootInodeID)
	fs.mu.RUnlock()

	for _, m := range mounts {
		fs.mount(root, m)
	}
}

func (fs *FileSystem) Mount(mount *Mount) {
	fs.mu.RLock()
	root := fs.getInodeOrDie(fuseops.RootInodeID)
	fs.mu.RUnlock()
	fs.mount(root, mount)
}

func (fs *FileSystem) Unmount(mountPoint string) {
	fs.mu.RLock()
	mp := fs.getInodeOrDie(fuseops.RootInodeID)
	fs.mu.RUnlock()

	fuseLog.Infof("Attempting to unmount %v", mountPoint)
	path := strings.Split(strings.Trim(mountPoint, "/"), "/")
	for _, localName := range path {
		dirInode := mp.findChild(localName)
		if dirInode == nil || !dirInode.isDir() {
			fuseLog.Errorf("Failed to find directory:%v while unmounting %v. "+
				"Ignoring the unmount operation.", localName, mountPoint)
			return
		}
		mp = dirInode
	}
	mp.ResetForUnmount()
	return
}

func (fs *FileSystem) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	cap := fs.cloud.Capabilities().CapacityBase
	if cap == nil {
		const BLOCK_SIZE = 4096
		const TOTAL_SPACE = 1 * 1024 * 1024 * 1024 * 1024 * 1024 // 1PB
		const TOTAL_BLOCKS = TOTAL_SPACE / BLOCK_SIZE
		const INODES = 1 * 1000 * 1000 * 1000 // 1 billion
		op.BlockSize = BLOCK_SIZE
		op.Blocks = TOTAL_BLOCKS
		op.BlocksFree = TOTAL_BLOCKS
		op.BlocksAvailable = TOTAL_BLOCKS
		op.IoSize = 1 * 1024 * 1024 // 1MB
		op.Inodes = INODES
		op.InodesFree = INODES
		return nil
	}

	op.BlockSize = cap.BlockSize
	op.Blocks = cap.Blocks
	op.BlocksFree = cap.BlocksFree
	op.BlocksAvailable = cap.BlocksAvailable
	op.IoSize = cap.IoSize
	op.Inodes = cap.Inodes
	op.InodesFree = cap.InodesFree
	return nil
}

func (fs *FileSystem) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) (err error) {
	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	attr, err := inode.GetAttributes()
	if err == nil {
		op.Attributes = *attr
		op.AttributesExpiration = time.Now().Add(fs.flags.StatCacheTTL)
	}

	return
}

func (fs *FileSystem) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) (err error) {
	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	value, err := inode.GetXattr(op.Name)
	if err != nil {
		return
	}

	op.BytesRead = len(value)

	if len(op.Dst) != 0 {
		if len(op.Dst) < op.BytesRead {
			return syscall.ERANGE
		}

		copy(op.Dst, value)
	}
	return
}

func (fs *FileSystem) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) (err error) {
	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	xattrs, err := inode.ListXattr()

	ncopied := 0

	for _, name := range xattrs {
		buf := op.Dst[ncopied:]
		nlen := len(name) + 1

		if nlen <= len(buf) {
			copy(buf, name)
			ncopied += nlen
			buf[nlen-1] = '\x00'
		}

		op.BytesRead += nlen
	}

	if len(op.Dst) != 0 && ncopied < op.BytesRead {
		err = syscall.ERANGE
	}

	return
}

func (fs *FileSystem) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) (err error) {
	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	err = inode.RemoveXattr(op.Name)

	return
}

func (fs *FileSystem) SetXattr(ctx context.Context,
	op *fuseops.SetXattrOp) (err error) {
	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	err = inode.SetXattr(op.Name, op.Value, op.Flags)
	return
}

func mapHttpError(status int) error {
	switch status {
	case 400:
		return fuse.EINVAL
	case 401:
		return syscall.EACCES
	case 403:
		return syscall.EACCES
	case 404:
		return fuse.ENOENT
	case 405:
		return syscall.ENOTSUP
	case http.StatusConflict:
		return syscall.EINTR
	case 429:
		return syscall.EAGAIN
	case 500:
		return syscall.EAGAIN
	default:
		return nil
	}
}

func mapStorageError(err error) error {
	if err == nil {
		return nil
	}

	if err == storage.ErrNoSuchKey {
		return syscall.ENXIO
	}
	if err == storage.ErrFileAlreadyExists {
		return fuse.EEXIST
	}
	return err
}

// note that this is NOT the same as url.PathEscape in golang 1.8,
// as this preserves / and url.PathEscape converts / to %2F
func pathEscape(path string) string {
	u := url.URL{Path: path}
	return u.EscapedPath()
}

func (fs *FileSystem) allocateInodeId() (id fuseops.InodeID) {
	id = fs.nextInodeID
	fs.nextInodeID++
	return
}

func expired(cache time.Time, ttl time.Duration) bool {
	now := time.Now()
	if cache.After(now) {
		return false
	}
	return !cache.Add(ttl).After(now)
}

func (fs *FileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) (err error) {

	var inode *Inode
	var ok bool
	defer func() { fuseLog.Debugf("<-- LookUpInode %v %v %v", op.Parent, op.Name, err) }()

	fs.mu.RLock()
	parent := fs.getInodeOrDie(op.Parent)
	fs.mu.RUnlock()

	parent.mu.Lock()
	inode = parent.findChildUnlocked(op.Name)
	if inode != nil {
		ok = true
		inode.Ref()

		if expired(inode.AttrTime, fs.flags.StatCacheTTL) {
			ok = false
			if atomic.LoadInt32(&inode.fileHandles) != 0 {
				// we have an open file handle, object
				// in S3 may not represent the true
				// state of the file anyway, so just
				// return what we know which is
				// potentially more accurate
				ok = true
			} else {
				inode.logFuse("lookup expired")
			}
		}
	} else {
		ok = false
	}
	parent.mu.Unlock()

	if !ok {
		var newInode *Inode

		newInode, err = parent.LookUp(op.Name)
		if err == fuse.ENOENT && inode != nil && inode.isDir() {
			// we may not be able to look up an implicit
			// dir if all the children are removed, so we
			// just pretend this dir is still around
			err = nil
		} else if err != nil {
			if inode != nil {
				// just kidding! pretend we didn't up the ref
				fs.mu.Lock()
				defer fs.mu.Unlock()

				stale := inode.DeRef(1)
				if stale {
					delete(fs.inodes, inode.Id)
					parent.removeChild(inode)
				}
			}
			return err
		}

		if inode == nil {
			parent.mu.Lock()
			// check again if it's there, could have been
			// added by another lookup or readdir
			inode = parent.findChildUnlocked(op.Name)
			if inode == nil {
				fs.mu.Lock()
				inode = newInode
				fs.insertInode(parent, inode)
				fs.mu.Unlock()
			}
			parent.mu.Unlock()
		} else {
			inode.mu.Lock()

			if newInode != nil {
				// if only size changed, kernel seems to
				// automatically drop cache
				if !inode.Attributes.Equal(newInode.Attributes) {
					inode.logFuse("invalidate cache because attributes changed", inode.Attributes, newInode.Attributes)
					inode.invalidateCache = true
				} else if inode.knownETag != nil &&
					newInode.knownETag != nil &&
					*inode.knownETag != *newInode.knownETag {
					// if this is a new file (ie:
					// inode.knownETag is nil),
					// then prefer to read our own
					// write then reading updated
					// data
					inode.logFuse("invalidate cache because etag changed", *inode.knownETag, *newInode.knownETag)
					inode.invalidateCache = true
				}

				if newInode.Attributes.Mtime.IsZero() {
					// this can happen if it's an
					// implicit dir, use the last
					// known value
					newInode.Attributes.Mtime = inode.Attributes.Mtime
				}
				inode.Attributes = newInode.Attributes
				inode.knownETag = newInode.knownETag
			}
			inode.AttrTime = time.Now()

			inode.mu.Unlock()
		}
	}

	op.Entry.Child = inode.Id
	op.Entry.Attributes = inode.InflateAttributes()
	op.Entry.AttributesExpiration = time.Now().Add(fs.flags.StatCacheTTL)
	op.Entry.EntryExpiration = time.Now().Add(fs.flags.TypeCacheTTL)

	return
}

// LOCKS_REQUIRED(fs.mu)
// LOCKS_REQUIRED(parent.mu)
func (fs *FileSystem) insertInode(parent *Inode, inode *Inode) {
	addInode := false
	if *inode.Name == "." {
		inode.Id = parent.Id
	} else if *inode.Name == ".." {
		inode.Id = fuseops.InodeID(fuseops.RootInodeID)
		if parent.Parent != nil {
			inode.Id = parent.Parent.Id
		}
	} else {
		if inode.Id != 0 {
			panic(fmt.Sprintf("inode id is set: %v %v", *inode.Name, inode.Id))
		}
		inode.Id = fs.allocateInodeId()
		addInode = true
	}
	parent.insertChildUnlocked(inode)
	if addInode {
		fs.inodes[inode.Id] = inode

		// if we are inserting a new directory, also create
		// the child . and ..
		if inode.isDir() {
			fs.addDotAndDotDot(inode)
		}
	}
}

func (fs *FileSystem) addDotAndDotDot(dir *Inode) {
	dot := NewInode(fs, dir, PString("."))
	dot.ToDir()
	dot.AttrTime = TIMEMAX
	fs.insertInode(dir, dot)

	dot = NewInode(fs, dir, PString(".."))
	dot.ToDir()
	dot.AttrTime = TIMEMAX
	fs.insertInode(dir, dot)
}

func (fs *FileSystem) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) (err error) {

	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	if inode.Parent != nil {
		inode.Parent.mu.Lock()
		defer inode.Parent.mu.Unlock()
	}
	stale := inode.DeRef(op.N)

	if stale {
		fs.mu.Lock()
		defer fs.mu.Unlock()

		delete(fs.inodes, op.Inode)
		fs.forgotCnt += 1

		if inode.Parent != nil {
			inode.Parent.removeChildUnlocked(inode)
		}
	}

	return
}

func (fs *FileSystem) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) (err error) {
	fs.mu.Lock()

	handleID := fs.nextHandleID
	fs.nextHandleID++

	in := fs.getInodeOrDie(op.Inode)
	fs.mu.Unlock()

	// XXX/is this a dir?
	dh := in.OpenDir()

	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.dirHandles[handleID] = dh
	op.Handle = handleID

	return
}

func makeDirEntry(en *DirHandleEntry) fuseutil.Dirent {
	return fuseutil.Dirent{
		Name:   en.Name,
		Type:   en.Type,
		Inode:  en.Inode,
		Offset: en.Offset,
	}
}

func (fs *FileSystem) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) (err error) {

	// Find the handle.
	fs.mu.RLock()
	dh := fs.dirHandles[op.Handle]
	fs.mu.RUnlock()

	if dh == nil {
		panic(fmt.Sprintf("can't find dh=%v", op.Handle))
	}

	inode := dh.inode
	inode.logFuse("ReadDir", op.Offset)

	dh.mu.Lock()
	defer dh.mu.Unlock()

	for i := op.Offset; ; i++ {
		e, err := dh.ReadDir(i)
		if err != nil {
			return err
		}
		if e == nil {
			break
		}

		if e.Inode == 0 {
			panic(fmt.Sprintf("unset inode %v", e.Name))
		}

		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], makeDirEntry(e))
		if n == 0 {
			break
		}

		dh.inode.logFuse("<-- ReadDir", e.Name, e.Offset)

		op.BytesRead += n
	}

	return
}

func (fs *FileSystem) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) (err error) {

	fs.mu.Lock()
	defer fs.mu.Unlock()

	dh := fs.dirHandles[op.Handle]
	dh.CloseDir()

	fuseLog.Debugln("ReleaseDirHandle", *dh.inode.FullName())

	delete(fs.dirHandles, op.Handle)

	return
}

func (fs *FileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) (err error) {
	fs.mu.RLock()
	in := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	fh, err := in.OpenFile(op.Metadata)
	if err != nil {
		return
	}

	fs.mu.Lock()

	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.fileHandles[handleID] = fh
	fs.mu.Unlock()

	op.Handle = handleID

	in.mu.Lock()
	defer in.mu.Unlock()

	// this flag appears to tell the kernel if this open should
	// use the page cache or not. "use" here means:
	//
	// read will read from cache
	// write will populate cache
	//
	// because we have one flag to control both behaviors, if an
	// object is updated out-of-band and we need to invalidate
	// cache, and we write to this object locally, subsequent read
	// will not read from cache
	//
	// see tests TestReadNewFileWithExternalChangesFuse and
	// TestReadMyOwnWrite*Fuse
	op.KeepPageCache = !in.invalidateCache
	fh.keepPageCache = op.KeepPageCache
	in.invalidateCache = false

	return
}

func (fs *FileSystem) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) (err error) {

	fs.mu.RLock()
	fh := fs.fileHandles[op.Handle]
	fs.mu.RUnlock()

	op.BytesRead, err = fh.ReadFile(op.Offset, op.Dst)

	return
}

func (fs *FileSystem) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) (err error) {

	// intentionally ignored, so that write()/sync()/write() works
	// see https://github.com/kahing/FileSystem/issues/154
	return
}

func (fs *FileSystem) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {

	fs.mu.RLock()
	fh := fs.fileHandles[op.Handle]
	fs.mu.RUnlock()

	// If the file handle has a tgid, then flush the file only if the
	// incoming request's tgid matches the tgid in the file handle.
	// This check helps us with scenarios like https://github.com/kahing/FileSystem/issues/273
	// Also see FileSystem_test.go:TestClientForkExec.
	if fh.Tgid != nil {
		tgid, err := GetTgid(op.Metadata.Pid)
		if err != nil {
			fh.inode.logFuse("<-- FlushFile",
				fmt.Sprintf("Failed to retrieve tgid from op.Metadata.Pid. FlushFileOp:%#v, err:%v",
					op, err))
			return fuse.EIO
		}
		if *fh.Tgid != *tgid {
			fh.inode.logFuse("<-- FlushFile",
				"Operation ignored",
				fmt.Sprintf("fh.Pid:%v != tgid:%v, op:%#v", *fh.Tgid, *tgid, op))
			return nil
		}
	}

	err = fh.FlushFile()
	if err != nil {
		// if we returned success from creat() earlier
		// linux may think this file exists even when it doesn't,
		// until TypeCacheTTL is over
		// TODO: figure out a way to make the kernel forget this inode
		// see TestWriteAnonymousFuse
		fs.mu.RLock()
		inode := fs.getInodeOrDie(op.Inode)
		fs.mu.RUnlock()

		if inode.KnownSize == nil {
			inode.AttrTime = time.Time{}
		}

	}
	fh.inode.logFuse("<-- FlushFile", err, op.Handle, op.Inode)
	return
}

func (fs *FileSystem) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fh := fs.fileHandles[op.Handle]
	fh.Release()

	fuseLog.Debugln("ReleaseFileHandle", *fh.inode.FullName(), op.Handle, fh.inode.Id)

	delete(fs.fileHandles, op.Handle)

	// try to compact heap
	//fs.bufferPool.MaybeGC()
	return
}

func (fs *FileSystem) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) (err error) {

	fs.mu.RLock()
	parent := fs.getInodeOrDie(op.Parent)
	fs.mu.RUnlock()

	inode, fh := parent.Create(op.Name, op.Metadata)

	parent.mu.Lock()

	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.insertInode(parent, inode)

	parent.mu.Unlock()

	op.Entry.Child = inode.Id
	op.Entry.Attributes = inode.InflateAttributes()
	op.Entry.AttributesExpiration = time.Now().Add(fs.flags.StatCacheTTL)
	op.Entry.EntryExpiration = time.Now().Add(fs.flags.TypeCacheTTL)

	// Allocate a handle.
	handleID := fs.nextHandleID
	fs.nextHandleID++

	fs.fileHandles[handleID] = fh

	op.Handle = handleID

	inode.logFuse("<-- CreateFile")

	return
}

func (fs *FileSystem) MkDir(ctx context.Context, op *fuseops.MkDirOp) (err error) {

	fs.mu.RLock()
	parent := fs.getInodeOrDie(op.Parent)
	fs.mu.RUnlock()

	// ignore op.Mode for now
	inode, err := parent.MkDir(op.Name)
	if err != nil {
		return err
	}

	parent.mu.Lock()

	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.insertInode(parent, inode)

	parent.mu.Unlock()

	op.Entry.Child = inode.Id
	op.Entry.Attributes = inode.InflateAttributes()
	op.Entry.AttributesExpiration = time.Now().Add(fs.flags.StatCacheTTL)
	op.Entry.EntryExpiration = time.Now().Add(fs.flags.TypeCacheTTL)

	return
}

func (fs *FileSystem) RmDir(ctx context.Context, op *fuseops.RmDirOp) (err error) {

	fs.mu.RLock()
	parent := fs.getInodeOrDie(op.Parent)
	fs.mu.RUnlock()

	err = parent.RmDir(op.Name)
	parent.logFuse("<-- RmDir", op.Name, err)
	return
}

func (fs *FileSystem) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) (err error) {

	fs.mu.RLock()
	inode := fs.getInodeOrDie(op.Inode)
	fs.mu.RUnlock()

	attr, err := inode.GetAttributes()
	if err == nil {
		op.Attributes = *attr
		op.AttributesExpiration = time.Now().Add(fs.flags.StatCacheTTL)
	}
	return
}

func (fs *FileSystem) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) (err error) {
	fs.mu.RLock()
	fh, ok := fs.fileHandles[op.Handle]
	if !ok {
		panic(fmt.Sprintf("WriteFile: can't find handle %v", op.Handle))
	}
	fs.mu.RUnlock()

	err = fh.WriteFile(op.Offset, op.Data)

	return
}

func (fs *FileSystem) Unlink(ctx context.Context, op *fuseops.UnlinkOp) (err error) {
	fs.mu.RLock()
	parent := fs.getInodeOrDie(op.Parent)
	fs.mu.RUnlock()

	err = parent.Unlink(op.Name)
	return
}

// rename("from", "to") causes the kernel to send lookup of "from" and
// "to" prior to sending rename to us
func (fs *FileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) (err error) {
	fs.mu.RLock()
	parent := fs.getInodeOrDie(op.OldParent)
	newParent := fs.getInodeOrDie(op.NewParent)
	fs.mu.RUnlock()

	// XXX don't hold the lock the entire time
	if op.OldParent == op.NewParent {
		parent.mu.Lock()
		defer parent.mu.Unlock()
	} else {
		// lock ordering to prevent deadlock
		if op.OldParent < op.NewParent {
			parent.mu.Lock()
			newParent.mu.Lock()
		} else {
			newParent.mu.Lock()
			parent.mu.Lock()
		}
		defer parent.mu.Unlock()
		defer newParent.mu.Unlock()
	}

	err = parent.Rename(op.OldName, newParent, op.NewName)
	if err != nil {
		if err == fuse.ENOENT {
			// if the source doesn't exist, it could be
			// because this is a new file and we haven't
			// flushed it yet, pretend that's ok because
			// when we flush we will handle the rename
			inode := parent.findChildUnlocked(op.OldName)
			if inode != nil && atomic.LoadInt32(&inode.fileHandles) != 0 {
				err = nil
			}
		}
	}
	if err == nil {
		inode := parent.findChildUnlocked(op.OldName)
		if inode != nil {
			inode.mu.Lock()
			defer inode.mu.Unlock()

			parent.removeChildUnlocked(inode)

			newNode := newParent.findChildUnlocked(op.NewName)
			if newNode != nil {
				// this file's been overwritten, it's
				// been detached but we can't delete
				// it just yet, because the kernel
				// will still send forget ops to us
				newParent.removeChildUnlocked(newNode)
				newNode.Parent = nil
			}

			inode.Name = &op.NewName
			inode.Parent = newParent
			newParent.insertChildUnlocked(inode)
		}
	}
	return
}
