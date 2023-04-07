package dbclient

import (
	"context"
	"fmt"

	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/hash"
	"github.com/protosio/distributeddolt/p2p"
)

func NewRemoteChunkStore(p2p *p2p.P2P) (*RemoteChunkStore, error) {
	return &RemoteChunkStore{p2p: p2p}, nil
}

type RemoteChunkStore struct {
	p2p *p2p.P2P
}

func (rcs *RemoteChunkStore) Get(ctx context.Context, h hash.Hash) (chunks.Chunk, error) {
	return chunks.Chunk{}, nil
}

func (rcs *RemoteChunkStore) GetMany(ctx context.Context, hashes hash.HashSet, found func(context.Context, *chunks.Chunk)) error {
	return nil
}

func (rcs *RemoteChunkStore) Has(ctx context.Context, h hash.Hash) (bool, error) {
	return false, nil
}

func (rcs *RemoteChunkStore) HasMany(ctx context.Context, hashes hash.HashSet) (hash.HashSet, error) {
	return make(hash.HashSet), nil
}

func (rcs *RemoteChunkStore) Put(ctx context.Context, c chunks.Chunk, getAddrs chunks.GetAddrsCb) error {
	return nil
}

func (rcs *RemoteChunkStore) Version() string {
	return "0"
}

func (rcs *RemoteChunkStore) Rebase(ctx context.Context) error {
	return nil
}

func (rcs *RemoteChunkStore) Root(ctx context.Context) (hash.Hash, error) {
	return hash.Hash{}, nil
}

func (rcs *RemoteChunkStore) Commit(ctx context.Context, current, last hash.Hash) (bool, error) {
	return false, fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) Stats() interface{} {
	return nil
}

func (rcs *RemoteChunkStore) StatsSummary() string {
	return "Unsupported"
}

func (rcs *RemoteChunkStore) Close() error {
	return nil
}
