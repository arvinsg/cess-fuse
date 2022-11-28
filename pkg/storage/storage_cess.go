package storage

// TODO
type CessConfig struct {
}

// TODO
type CessStorage struct {
}

// TODO
func NewCessStorage(cfg *CessConfig) (ObjectBackend, error) {
	return &CessStorage{}, nil
}

// TODO
func (cs *CessStorage) Init(key string) error {
	return ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) Capabilities() *Capabilities {
	return nil
}

// TODO
func (cs *CessStorage) Bucket() string {
	return ""
}

// TODO
func (cs *CessStorage) HeadBlob(param *HeadBlobInput) (*HeadBlobOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) ListBlobs(param *ListBlobsInput) (*ListBlobsOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) DeleteBlob(param *DeleteBlobInput) (*DeleteBlobOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) DeleteBlobs(param *DeleteBlobsInput) (*DeleteBlobsOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) RenameBlob(param *RenameBlobInput) (*RenameBlobOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) CopyBlob(param *CopyBlobInput) (*CopyBlobOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) GetBlob(param *GetBlobInput) (*GetBlobOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) PutBlob(param *PutBlobInput) (*PutBlobOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) MultipartBlobBegin(param *MultipartBlobBeginInput) (*MultipartBlobCommitInput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) MultipartBlobAdd(param *MultipartBlobAddInput) (*MultipartBlobAddOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) MultipartBlobAbort(param *MultipartBlobCommitInput) (*MultipartBlobAbortOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) MultipartBlobCommit(param *MultipartBlobCommitInput) (*MultipartBlobCommitOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) MultipartExpire(param *MultipartExpireInput) (*MultipartExpireOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) RemoveBucket(param *RemoveBucketInput) (*RemoveBucketOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) MakeBucket(param *MakeBucketInput) (*MakeBucketOutput, error) {
	return nil, ErrUnsupportedMethod
}

// TODO
func (cs *CessStorage) Delegate() interface{} {
	return nil
}
