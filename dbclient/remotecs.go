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
	clients := rcs.p2p.GetAllClients()
	if len(clients) == 0 {
		return rcs.localCS.GetMany(ctx, hashes, found)
	}

	return rcs.localCS.GetMany(ctx, hashes, found)
}

func (rcs *RemoteChunkStore) Has(ctx context.Context, h hash.Hash) (bool, error) {
	fmt.Println("calling Has")
	clients := rcs.p2p.GetAllClients()
	if len(clients) == 0 {
		return rcs.localCS.Has(ctx, h)
	}

	return rcs.localCS.Has(ctx, h)
}

func (rcs *RemoteChunkStore) HasMany(ctx context.Context, hashes hash.HashSet) (hash.HashSet, error) {
	fmt.Println("calling HasMany")
	clients := rcs.p2p.GetAllClients()
	if len(clients) == 0 {
		return rcs.localCS.HasMany(ctx, hashes)
	}

	return rcs.localCS.HasMany(ctx, hashes)
}

func (rcs *RemoteChunkStore) Put(ctx context.Context, c chunks.Chunk, getAddrs chunks.GetAddrsCb) error {
	fmt.Println("calling Put")
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) Version() string {
	fmt.Println("calling Version")
	clients := rcs.p2p.GetAllClients()

	if len(clients) == 0 {
		return rcs.localCS.Version()
	}

	responses := map[string]*remotesapi.GetRepoMetadataResponse{}
	for id, client := range clients {
		resp, err := client.GetRepoMetadata(context.Background(), &remotesapi.GetRepoMetadataRequest{})
		if err != nil {
			fmt.Printf("Error getting repo metadata from client %s: %s, skipping\n", id, err)
		}
		responses[id] = resp
	}
	return rcs.localCS.Version()
}

func (rcs *RemoteChunkStore) Rebase(ctx context.Context) error {
	fmt.Println("calling Rebase")
	return fmt.Errorf("not supported")
}

// func (rcs *RemoteChunkStore) loadRoot(ctx context.Context) error {
// 	clients := rcs.p2p.GetAllClients()
// 	if len(clients) == 0 {
// 		return fmt.Errorf("failed to load root, no clients")
// 	}

// 	id, token := rcs.getRepoId()
// 	req := &remotesapi.RootRequest{RepoId: id, RepoToken: token, RepoPath: "/"}
// 	responses := map[string]*remotesapi.RootResponse{}
// 	for id, client := range clients {
// 		resp, err := client.Root(context.Background(), req)
// 		if err != nil {
// 			fmt.Printf("Error getting repo metadata from client %s: %s, skipping\n", id, err)
// 		}
// 		responses[id] = resp
// 	}

// 	if resp.RepoToken != "" {
// 		dcs.repoToken.Store(resp.RepoToken)
// 	}
// 	dcs.root = hash.New(resp.RootHash)
// 	return nil
// }

func (rcs *RemoteChunkStore) Root(ctx context.Context) (hash.Hash, error) {
	fmt.Println("calling Root")
	clients := rcs.p2p.GetAllClients()
	if len(clients) == 0 {
		return rcs.localCS.Root(ctx)
	}

	for _, client := range clients {
		resp, err := client.Root(context.Background(), &remotesapi.RootRequest{})
		if err != nil {
			return hash.Hash{}, err
		}
		return hash.New(resp.RootHash), nil
	}

	return hash.Hash{}, fmt.Errorf("no clients")
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

//
// TableFileStore implementation
//

func (rcs *RemoteChunkStore) Sources(ctx context.Context) (hash.Hash, []chunks.TableFile, []chunks.TableFile, error) {
	return hash.Hash{}, nil, nil, fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) Size(ctx context.Context) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) WriteTableFile(ctx context.Context, fileId string, numChunks int, contentHash []byte, getRd func() (io.ReadCloser, uint64, error)) error {
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) AddTableFilesToManifest(ctx context.Context, fileIdToNumChunks map[string]int) error {
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) PruneTableFiles(ctx context.Context) error {
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) SetRootChunk(ctx context.Context, root, previous hash.Hash) error {
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) SupportedOperations() chunks.TableFileStoreOps {
	tfs, ok := rcs.localCS.(chunks.TableFileStore)
	if !ok {
		return chunks.TableFileStoreOps{}
	}

	return tfs.SupportedOperations()
}
