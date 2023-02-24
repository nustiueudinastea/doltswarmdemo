package main

import (
	"context"
	"fmt"

	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/hash"
	"github.com/dolthub/dolt/go/store/types"
)

type ProtosChunkStore struct {
	logger chunks.DebugLogger
	root   hash.Hash
}

func NewProtosChunkStore(ctx context.Context, nbf *types.NomsBinFormat) (*ProtosChunkStore, error) {

	cs := &ProtosChunkStore{}
	err := cs.loadRoot(ctx)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

func (pcs *ProtosChunkStore) SetLogger(logger chunks.DebugLogger) {
	pcs.logger = logger
}

func (pcs *ProtosChunkStore) logf(fmt string, args ...interface{}) {
	if pcs.logger != nil {
		pcs.logger.Logf(fmt, args...)
	}
}

type cacheStats struct {
	Hits uint32
}

func (s cacheStats) CacheHits() uint32 {
	return s.Hits
}

type CacheStats interface {
	CacheHits() uint32
}

// Get the Chunk for the value of the hash in the store. If the hash is absent from the store EmptyChunk is returned.
func (pcs *ProtosChunkStore) Get(ctx context.Context, h hash.Hash) (chunks.Chunk, error) {
	hashes := hash.HashSet{h: struct{}{}}
	var found *chunks.Chunk
	err := pcs.GetMany(ctx, hashes, func(_ context.Context, c *chunks.Chunk) { found = c })
	if err != nil {
		return chunks.EmptyChunk, err
	}
	if found != nil {
		return *found, nil
	} else {
		return chunks.EmptyChunk, nil
	}
}

func (pcs *ProtosChunkStore) GetMany(ctx context.Context, hashes hash.HashSet, found func(context.Context, *chunks.Chunk)) error {
	return nil
}

// Returns true iff the value at the address |h| is contained in the
// store
func (pcs *ProtosChunkStore) Has(ctx context.Context, h hash.Hash) (bool, error) {
	hashes := hash.HashSet{h: struct{}{}}
	absent, err := pcs.HasMany(ctx, hashes)

	if err != nil {
		return false, err
	}

	return len(absent) == 0, nil
}

const maxHasManyBatchSize = 16 * 1024

// Returns a new HashSet containing any members of |hashes| that are
// absent from the store.
func (pcs *ProtosChunkStore) HasMany(ctx context.Context, hashes hash.HashSet) (hash.HashSet, error) {
	absent := make(hash.HashSet)
	return absent, nil
}

// Put caches c. Upon return, c must be visible to
// subsequent Get and Has calls, but must not be persistent until a call
// to Flush(). Put may be called concurrently with other calls to Put(),
// Get(), GetMany(), Has() and HasMany().
func (pcs *ProtosChunkStore) Put(ctx context.Context, c chunks.Chunk, getAddrs chunks.GetAddrsCb) error {
	return nil
}

// Returns the NomsVersion with which this ChunkSource is compatible.
func (pcs *ProtosChunkStore) Version() string {
	return "1"
}

// Rebase brings this ChunkStore into sync with the persistent storage's
// current root.
func (pcs *ProtosChunkStore) Rebase(ctx context.Context) error {
	return nil
}

// Root returns the root of the database as of the time the ChunkStore
// was opened or the most recent call to Rebase.
func (pcs *ProtosChunkStore) Root(ctx context.Context) (hash.Hash, error) {
	return pcs.root, nil
}

func (pcs *ProtosChunkStore) loadRoot(ctx context.Context) error {
	pcs.root = hash.New([]byte{})
	return nil
}

// Commit atomically attempts to persist all novel Chunks and update the
// persisted root hash from last to current (or keeps it the same).
// If last doesn't match the root in persistent storage, returns false.
func (pcs *ProtosChunkStore) Commit(ctx context.Context, current, last hash.Hash) (bool, error) {
	return false, nil
}

// Stats may return some kind of struct that reports statistics about the
// ChunkStore instance. The type is implementation-dependent, and impls
// may return nil
func (pcs *ProtosChunkStore) Stats() interface{} {
	return cacheStats{}
}

// StatsSummary may return a string containing summarized statistics for
// this ChunkStore. It must return "Unsupported" if this operation is not
// supported.
func (pcs *ProtosChunkStore) StatsSummary() string {
	return fmt.Sprintf("CacheHits: %v", pcs.Stats().(CacheStats).CacheHits())
}

// Close tears down any resources in use by the implementation. After
// Close(), the ChunkStore may not be used again. It is NOT SAFE to call
// Close() concurrently with any other ChunkStore method; behavior is
// undefined and probably crashy.
func (pcs *ProtosChunkStore) Close() error {
	return nil
}

func (pcs *ProtosChunkStore) SupportedOperations() chunks.TableFileStoreOps {
	return chunks.TableFileStoreOps{
		CanRead:  true,
		CanWrite: true,
		CanPrune: false,
		CanGC:    false,
	}
}

// SetRootChunk changes the root chunk hash from the previous value to the new root.
func (pcs *ProtosChunkStore) SetRootChunk(ctx context.Context, root, previous hash.Hash) error {
	panic("Not Implemented")
}
