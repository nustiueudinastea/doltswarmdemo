package dbclient

import (
	"context"
	"fmt"

	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/hash"
	"github.com/protosio/distributeddolt/p2p"
)

func NewRemoteChunkStore(p2p *p2p.P2P, localCS chunks.ChunkStore) (*RemoteChunkStore, error) {
	return &RemoteChunkStore{p2p: p2p, localCS: localCS}, nil
}

type RemoteChunkStore struct {
	p2p     *p2p.P2P
	localCS chunks.ChunkStore
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
	clients := rcs.p2p.GetAllClients()

	if len(clients) == 0 {
		return rcs.localCS.Version()
	}

	responses := map[string]*remotesapi.GetRepoMetadataResponse{}
	for id, client := range clients {
		resp, err := client.ChunkStoreServiceClient.GetRepoMetadata(context.Background(), &remotesapi.GetRepoMetadataRequest{})
		if err != nil {
			fmt.Printf("Error getting repo metadata from client %s: %s, skipping\n", id, err)
		}
		responses[id] = resp
	}
	return rcs.localCS.Version()
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
