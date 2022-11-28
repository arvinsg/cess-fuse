package storage

import (
	"fmt"
	"io"
	"sync"
	"time"
)

/// Implementations of all the functions here are expected to be
/// concurrency-safe, except for
///
/// Init() is called exactly once before any other functions are
/// called.
type ObjectBackend interface {
	Init(key string) error

	Capabilities() *Capabilities
	// typically this would return bucket/prefix
	Bucket() string
	HeadBlob(param *HeadBlobInput) (*HeadBlobOutput, error)
	ListBlobs(param *ListBlobsInput) (*ListBlobsOutput, error)
	DeleteBlob(param *DeleteBlobInput) (*DeleteBlobOutput, error)
	DeleteBlobs(param *DeleteBlobsInput) (*DeleteBlobsOutput, error)
	RenameBlob(param *RenameBlobInput) (*RenameBlobOutput, error)
	CopyBlob(param *CopyBlobInput) (*CopyBlobOutput, error)
	GetBlob(param *GetBlobInput) (*GetBlobOutput, error)
	PutBlob(param *PutBlobInput) (*PutBlobOutput, error)
	MultipartBlobBegin(param *MultipartBlobBeginInput) (*MultipartBlobCommitInput, error)
	MultipartBlobAdd(param *MultipartBlobAddInput) (*MultipartBlobAddOutput, error)
	MultipartBlobAbort(param *MultipartBlobCommitInput) (*MultipartBlobAbortOutput, error)
	MultipartBlobCommit(param *MultipartBlobCommitInput) (*MultipartBlobCommitOutput, error)
	MultipartExpire(param *MultipartExpireInput) (*MultipartExpireOutput, error)
	RemoveBucket(param *RemoveBucketInput) (*RemoveBucketOutput, error)
	MakeBucket(param *MakeBucketInput) (*MakeBucketOutput, error)
	Delegate() interface{}
}

type Delegator interface {
	Delegate() interface{}
}

type Capabilities struct {
	NoParallelMultipart bool
	MaxMultipartSize    uint64
	// indicates that the blob store has native support for directories
	DirBlob bool
	Name    string

	Base *CapacityBase
}

type CapacityBase struct {
	BlockSize       uint32
	Blocks          uint64
	BlocksFree      uint64
	BlocksAvailable uint64
	IoSize          uint32
	Inodes          uint64
	InodesFree      uint64
}

type HeadBlobInput struct {
	Key string
}

type BlobItemOutput struct {
	Key          *string
	ETag         *string
	LastModified *time.Time
	Size         uint64
	StorageClass *string
}

type HeadBlobOutput struct {
	BlobItemOutput

	ContentType *string
	Metadata    map[string]*string
	IsDirBlob   bool

	RequestId string
}

type ListBlobsInput struct {
	Prefix            *string
	Delimiter         *string
	MaxKeys           *uint32
	StartAfter        *string // XXX: not supported by Azure
	ContinuationToken *string
}

type BlobPrefixOutput struct {
	Prefix *string
}

type ListBlobsOutput struct {
	Prefixes              []BlobPrefixOutput
	Items                 []BlobItemOutput
	NextContinuationToken *string
	IsTruncated           bool

	RequestId string
}

type DeleteBlobInput struct {
	Key string
}

type DeleteBlobOutput struct {
	RequestId string
}

type DeleteBlobsInput struct {
	Items []string
}

type DeleteBlobsOutput struct {
	RequestId string
}

type RenameBlobInput struct {
	Source      string
	Destination string
}

type RenameBlobOutput struct {
	RequestId string
}

type CopyBlobInput struct {
	Source      string
	Destination string

	Size         *uint64
	ETag         *string            // if non-nil, do conditional copy
	Metadata     map[string]*string // if nil, copy from Source
	StorageClass *string            // if nil, copy from Source
}

type CopyBlobOutput struct {
	RequestId string
}

type GetBlobInput struct {
	Key     string
	Start   uint64
	Count   uint64
	IfMatch *string
}

type GetBlobOutput struct {
	HeadBlobOutput

	Body io.ReadCloser

	RequestId string
}

type PutBlobInput struct {
	Key         string
	Metadata    map[string]*string
	ContentType *string
	DirBlob     bool

	Body io.ReadSeeker
	Size *uint64
}

type PutBlobOutput struct {
	ETag         *string
	LastModified *time.Time
	StorageClass *string

	RequestId string
}

type MultipartBlobBeginInput struct {
	Key         string
	Metadata    map[string]*string
	ContentType *string
}

type MultipartBlobCommitInput struct {
	Key *string

	Metadata map[string]*string
	UploadId *string
	Parts    []*string
	NumParts uint32

	// for GCS
	backendData interface{}
}

type MultipartBlobAddInput struct {
	Commit     *MultipartBlobCommitInput
	PartNumber uint32

	Body io.ReadSeeker

	Size   uint64 // GCS wants to know part size
	Last   bool   // GCS needs to know if this part is the last one
	Offset uint64 // ADLv2 needs to know offset
}

type MultipartBlobAddOutput struct {
	RequestId string
}

type MultipartBlobCommitOutput struct {
	ETag         *string
	LastModified *time.Time
	StorageClass *string

	RequestId string
}

type MultipartBlobAbortOutput struct {
	RequestId string
}

type MultipartExpireInput struct {
}

type MultipartExpireOutput struct {
	RequestId string
}

type RemoveBucketInput struct {
}

type RemoveBucketOutput struct {
	RequestId string
}

type MakeBucketInput struct {
}

type MakeBucketOutput struct {
	RequestId string
}

type sortBlobPrefixOutput []BlobPrefixOutput

func (p sortBlobPrefixOutput) Len() int {
	return len(p)
}

func (p sortBlobPrefixOutput) Less(i, j int) bool {
	return *p[i].Prefix < *p[j].Prefix
}

func (p sortBlobPrefixOutput) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type sortBlobItemOutput []BlobItemOutput

func (p sortBlobItemOutput) Len() int {
	return len(p)
}

func (p sortBlobItemOutput) Less(i, j int) bool {
	return *p[i].Key < *p[j].Key
}

func (p sortBlobItemOutput) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (b BlobItemOutput) String() string {
	return fmt.Sprintf("%v: %v", *b.Key, b.Size)
}

func (b BlobPrefixOutput) String() string {
	return fmt.Sprintf("%v", *b.Prefix)
}

type ReadSeekerCloser struct {
	io.ReadSeeker
}

func (r *ReadSeekerCloser) Close() error {
	if closer, ok := r.ReadSeeker.(io.Closer); ok {
		return closer.Close()
	} else {
		return nil
	}
}

/////////////////////////////////////////////
type ObjectBackendInitWrapper struct {
	ObjectBackend
	init    sync.Once
	initKey string
	initErr error
}

func (s *ObjectBackendInitWrapper) Init(key string) error {
	s.init.Do(func() {
		s.initErr = s.ObjectBackend.Init(s.initKey)
		if s.initErr != nil {
			s.ObjectBackend = ObjectBackendInitError{
				s.initErr,
				*s.ObjectBackend.Capabilities(),
			}
		}
	})
	return s.initErr
}

func (s *ObjectBackendInitWrapper) Capabilities() *Capabilities {
	return s.ObjectBackend.Capabilities()
}

func (s *ObjectBackendInitWrapper) Bucket() string {
	return s.ObjectBackend.Bucket()
}

func (s *ObjectBackendInitWrapper) HeadBlob(param *HeadBlobInput) (*HeadBlobOutput, error) {
	s.Init("")
	return s.ObjectBackend.HeadBlob(param)
}

func (s *ObjectBackendInitWrapper) ListBlobs(param *ListBlobsInput) (*ListBlobsOutput, error) {
	s.Init("")
	return s.ObjectBackend.ListBlobs(param)
}

func (s *ObjectBackendInitWrapper) DeleteBlob(param *DeleteBlobInput) (*DeleteBlobOutput, error) {
	s.Init("")
	return s.ObjectBackend.DeleteBlob(param)
}

func (s *ObjectBackendInitWrapper) DeleteBlobs(param *DeleteBlobsInput) (*DeleteBlobsOutput, error) {
	s.Init("")
	return s.ObjectBackend.DeleteBlobs(param)
}

func (s *ObjectBackendInitWrapper) RenameBlob(param *RenameBlobInput) (*RenameBlobOutput, error) {
	s.Init("")
	return s.ObjectBackend.RenameBlob(param)
}

func (s *ObjectBackendInitWrapper) CopyBlob(param *CopyBlobInput) (*CopyBlobOutput, error) {
	s.Init("")
	return s.ObjectBackend.CopyBlob(param)
}

func (s *ObjectBackendInitWrapper) GetBlob(param *GetBlobInput) (*GetBlobOutput, error) {
	s.Init("")
	return s.ObjectBackend.GetBlob(param)
}

func (s *ObjectBackendInitWrapper) PutBlob(param *PutBlobInput) (*PutBlobOutput, error) {
	s.Init("")
	return s.ObjectBackend.PutBlob(param)
}

func (s *ObjectBackendInitWrapper) MultipartBlobBegin(param *MultipartBlobBeginInput) (*MultipartBlobCommitInput, error) {
	s.Init("")
	return s.ObjectBackend.MultipartBlobBegin(param)
}

func (s *ObjectBackendInitWrapper) MultipartBlobAdd(param *MultipartBlobAddInput) (*MultipartBlobAddOutput, error) {
	s.Init("")
	return s.ObjectBackend.MultipartBlobAdd(param)
}

func (s *ObjectBackendInitWrapper) MultipartBlobAbort(param *MultipartBlobCommitInput) (*MultipartBlobAbortOutput, error) {
	s.Init("")
	return s.ObjectBackend.MultipartBlobAbort(param)
}

func (s *ObjectBackendInitWrapper) MultipartBlobCommit(param *MultipartBlobCommitInput) (*MultipartBlobCommitOutput, error) {
	s.Init("")
	return s.ObjectBackend.MultipartBlobCommit(param)
}

func (s *ObjectBackendInitWrapper) MultipartExpire(param *MultipartExpireInput) (*MultipartExpireOutput, error) {
	s.Init("")
	return s.ObjectBackend.MultipartExpire(param)
}

func (s *ObjectBackendInitWrapper) RemoveBucket(param *RemoveBucketInput) (*RemoveBucketOutput, error) {
	s.Init("")
	return s.ObjectBackend.RemoveBucket(param)
}

func (s *ObjectBackendInitWrapper) MakeBucket(param *MakeBucketInput) (*MakeBucketOutput, error) {
	s.Init("")
	return s.ObjectBackend.MakeBucket(param)
}

type ObjectBackendInitError struct {
	error
	cap Capabilities
}

func (oe ObjectBackendInitError) Init(key string) error {
	return oe
}

func (oe ObjectBackendInitError) Delegate() interface{} {
	return oe
}

func (oe ObjectBackendInitError) Capabilities() *Capabilities {
	return &oe.cap
}

func (oe ObjectBackendInitError) Bucket() string {
	return ""
}

func (oe ObjectBackendInitError) HeadBlob(param *HeadBlobInput) (*HeadBlobOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) ListBlobs(param *ListBlobsInput) (*ListBlobsOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) DeleteBlob(param *DeleteBlobInput) (*DeleteBlobOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) DeleteBlobs(param *DeleteBlobsInput) (*DeleteBlobsOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) RenameBlob(param *RenameBlobInput) (*RenameBlobOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) CopyBlob(param *CopyBlobInput) (*CopyBlobOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) GetBlob(param *GetBlobInput) (*GetBlobOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) PutBlob(param *PutBlobInput) (*PutBlobOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) MultipartBlobBegin(param *MultipartBlobBeginInput) (*MultipartBlobCommitInput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) MultipartBlobAdd(param *MultipartBlobAddInput) (*MultipartBlobAddOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) MultipartBlobAbort(param *MultipartBlobCommitInput) (*MultipartBlobAbortOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) MultipartBlobCommit(param *MultipartBlobCommitInput) (*MultipartBlobCommitOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) MultipartExpire(param *MultipartExpireInput) (*MultipartExpireOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) RemoveBucket(param *RemoveBucketInput) (*RemoveBucketOutput, error) {
	return nil, oe
}

func (oe ObjectBackendInitError) MakeBucket(param *MakeBucketInput) (*MakeBucketOutput, error) {
	return nil, oe
}
