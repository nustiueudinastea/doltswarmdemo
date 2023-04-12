package dbclient

import (
	"context"
	"fmt"
	"io"

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
	fmt.Println("calling Get")
	clients := rcs.p2p.GetAllClients()
	if len(clients) == 0 {
		return rcs.localCS.Get(ctx, h)
	}

	return rcs.localCS.Get(ctx, h)
}

func (rcs *RemoteChunkStore) GetMany(ctx context.Context, hashes hash.HashSet, found func(context.Context, *chunks.Chunk)) error {
	fmt.Println("calling GetMany")
	return nil
}

func (rcs *RemoteChunkStore) Has(ctx context.Context, h hash.Hash) (bool, error) {
	fmt.Println("calling Has")
	return false, nil
}

func (rcs *RemoteChunkStore) HasMany(ctx context.Context, hashes hash.HashSet) (hash.HashSet, error) {
	fmt.Println("calling HasMany")
	return make(hash.HashSet), nil
}

func (rcs *RemoteChunkStore) Put(ctx context.Context, c chunks.Chunk, getAddrs chunks.GetAddrsCb) error {
	fmt.Println("calling Put")
	return nil
}

func (rcs *RemoteChunkStore) Version() string {
	fmt.Println("calling Version")
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
	fmt.Println("calling Rebase")
	return nil
}

func (rcs *RemoteChunkStore) Root(ctx context.Context) (hash.Hash, error) {
	fmt.Println("calling Root")
	clients := rcs.p2p.GetAllClients()
	if len(clients) == 0 {
		return rcs.localCS.Root(ctx)
	}

	return rcs.localCS.Root(ctx)
}

func (rcs *RemoteChunkStore) Commit(ctx context.Context, current, last hash.Hash) (bool, error) {
	fmt.Println("calling Commit")
	return false, fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) Stats() interface{} {
	fmt.Println("calling Stats")
	return nil
}

func (rcs *RemoteChunkStore) StatsSummary() string {
	fmt.Println("calling StatsSummary")
	return "Unsupported"
}

func (rcs *RemoteChunkStore) Close() error {
	fmt.Println("calling Close")
	return nil
}

// TableFileStore implementation

func (rcs *RemoteChunkStore) Sources(ctx context.Context) (hash.Hash, []chunks.TableFile, []chunks.TableFile, error) {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return hash.Hash{}, nil, nil, fmt.Errorf("local chunk store does not implement TableFileStore")
	}

	return tfs.Sources(ctx)
}

func (rcs *RemoteChunkStore) Size(ctx context.Context) (uint64, error) {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return 0, fmt.Errorf("local chunk store does not implement TableFileStore")
	}

	return tfs.Size(ctx)
}

func (rcs *RemoteChunkStore) WriteTableFile(ctx context.Context, fileId string, numChunks int, contentHash []byte, getRd func() (io.ReadCloser, uint64, error)) error {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return fmt.Errorf("local chunk store does not implement TableFileStore")
	}

	return tfs.WriteTableFile(ctx, fileId, numChunks, contentHash, getRd)
}

func (rcs *RemoteChunkStore) AddTableFilesToManifest(ctx context.Context, fileIdToNumChunks map[string]int) error {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return fmt.Errorf("local chunk store does not implement TableFileStore")
	}

	return tfs.AddTableFilesToManifest(ctx, fileIdToNumChunks)
}

func (rcs *RemoteChunkStore) PruneTableFiles(ctx context.Context) error {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return fmt.Errorf("local chunk store does not implement TableFileStore")
	}

	return tfs.PruneTableFiles(ctx)
}

func (rcs *RemoteChunkStore) SetRootChunk(ctx context.Context, root, previous hash.Hash) error {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return fmt.Errorf("local chunk store does not implement TableFileStore")
	}

	return tfs.SetRootChunk(ctx, root, previous)
}

func (rcs *RemoteChunkStore) SupportedOperations() chunks.TableFileStoreOps {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return chunks.TableFileStoreOps{}
	}

	return tfs.SupportedOperations()

}
