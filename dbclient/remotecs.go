package dbclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync/atomic"

	"github.com/cenkalti/backoff"
	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
	"github.com/dolthub/dolt/go/libraries/doltcore/remotestorage"
	"github.com/dolthub/dolt/go/store/atomicerr"
	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/hash"
	"github.com/dolthub/dolt/go/store/nbs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	getLocsBatchSize    = 256
	chunkAggDistance    = 8 * 1024
	maxHasManyBatchSize = 16 * 1024
)

func NewRemoteChunkStore(client Client, peerID string, nbfVersion string) (*RemoteChunkStore, error) {
	rcs := &RemoteChunkStore{
		client:      client,
		peerID:      peerID,
		cache:       newMapChunkCache(),
		httpFetcher: &http.Client{},
		concurrency: ConcurrencyParams{
			ConcurrentSmallFetches: 64,
			ConcurrentLargeFetches: 2,
			LargeFetchSize:         2 * 1024 * 1024,
		},
		nbfVersion: nbfVersion,
	}

	metadata, err := client.GetRepoMetadata(context.Background(), &remotesapi.GetRepoMetadataRequest{
		RepoId:   rcs.getRepoId(),
		RepoPath: "",
		ClientRepoFormat: &remotesapi.ClientRepoFormat{
			NbfVersion: nbfVersion,
			NbsVersion: nbs.StorageVersion,
		},
	})
	if err != nil {
		return nil, err
	}

	rcs.repoSize = metadata.StorageSize

	err = rcs.loadRoot(context.Background())
	if err != nil {
		return nil, err
	}

	return rcs, nil
}

type HTTPFetcher interface {
	Do(req *http.Request) (*http.Response, error)
}

type ConcurrencyParams struct {
	ConcurrentSmallFetches int
	ConcurrentLargeFetches int
	LargeFetchSize         int
}

type RemoteChunkStore struct {
	client      Client
	peerID      string
	cache       remotestorage.ChunkCache
	httpFetcher HTTPFetcher
	concurrency ConcurrencyParams
	nbfVersion  string
	repoSize    uint64
	root        hash.Hash
}

func (rcs *RemoteChunkStore) Get(ctx context.Context, h hash.Hash) (chunks.Chunk, error) {
	fmt.Println("calling Get")

	hashes := hash.HashSet{h: struct{}{}}
	var found *chunks.Chunk
	err := rcs.GetMany(ctx, hashes, func(_ context.Context, c *chunks.Chunk) { found = c })
	if err != nil {
		return chunks.EmptyChunk, err
	}
	if found != nil {
		return *found, nil
	} else {
		return chunks.EmptyChunk, nil
	}
}

func (rcs *RemoteChunkStore) GetMany(ctx context.Context, hashes hash.HashSet, found func(context.Context, *chunks.Chunk)) error {
	fmt.Println("calling GetMany")
	ae := atomicerr.New()
	decompressedSize := uint64(0)
	err := rcs.GetManyCompressed(ctx, hashes, func(ctx context.Context, cc nbs.CompressedChunk) {
		if ae.IsSet() {
			return
		}
		c, err := cc.ToChunk()
		if ae.SetIfErrAndCheck(err) {
			return
		}
		atomic.AddUint64(&decompressedSize, uint64(len(c.Data())))
		found(ctx, &c)
	})
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int64("decompressed_bytes", int64(decompressedSize)))
	if err != nil {
		return err
	}
	if err = ae.Get(); err != nil {
		return err
	}
	return nil
}

func (rcs *RemoteChunkStore) GetManyCompressed(ctx context.Context, hashes hash.HashSet, found func(context.Context, nbs.CompressedChunk)) error {
	hashToChunk := rcs.cache.Get(hashes)

	notCached := make([]hash.Hash, 0, len(hashes))
	for h := range hashes {
		c := hashToChunk[h]

		if c.IsEmpty() {
			notCached = append(notCached, h)
		} else {
			found(ctx, c)
		}
	}

	if len(notCached) > 0 {
		err := rcs.readChunksAndCache(ctx, hashes, notCached, found)

		if err != nil {
			return err
		}
	}

	return nil
}

func (rcs *RemoteChunkStore) getDLLocs(ctx context.Context, hashes []hash.Hash) (dlLocations, error) {

	res := dlLocations{
		ranges:    make(map[string]*remotestorage.GetRange),
		refreshes: make(map[string]*locationRefresh),
	}

	// channel for receiving results from go routines making grpc calls to get download locations for chunks
	resCh := make(chan []*remotesapi.DownloadLoc)

	eg, ctx := errgroup.WithContext(ctx)

	// go routine for receiving the results of the grpc calls and aggregating the results into resourceToUrlAndRanges
	eg.Go(func() error {
		for {
			select {
			case locs, ok := <-resCh:
				if !ok {
					return nil
				}
				for _, loc := range locs {
					res.Add(loc)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	// go routine for batching the get location requests, streaming the requests and streaming the responses.
	eg.Go(func() error {
		var reqs []*remotesapi.GetDownloadLocsRequest
		hashesBytes := remotestorage.HashesToSlices(hashes)
		batchItr(len(hashesBytes), getLocsBatchSize, func(st, end int) (stop bool) {
			batch := hashesBytes[st:end]
			req := &remotesapi.GetDownloadLocsRequest{RepoPath: rcs.peerID, ChunkHashes: batch}
			reqs = append(reqs, req)
			return false
		})
		op := func() error {
			seg, ctx := errgroup.WithContext(ctx)
			stream, err := rcs.client.StreamDownloadLocations(ctx)
			if err != nil {
				return remotestorage.NewRpcError(err, "StreamDownloadLocations", rcs.peerID, nil)
			}
			completedReqs := 0
			// Write requests
			seg.Go(func() error {
				for i := range reqs {
					if err := stream.Send(reqs[i]); err != nil {
						return remotestorage.NewRpcError(err, "StreamDownloadLocations", rcs.peerID, reqs[i])
					}
				}
				return stream.CloseSend()
			})
			// Read responses
			seg.Go(func() error {
				for {
					resp, err := stream.Recv()
					if err != nil {
						if err == io.EOF {
							return nil
						}
						var r *remotesapi.GetDownloadLocsRequest
						if completedReqs < len(reqs) {
							r = reqs[completedReqs]
						}
						return remotestorage.NewRpcError(err, "StreamDownloadLocations", rcs.peerID, r)
					}
					select {
					case resCh <- resp.Locs:
						completedReqs += 1
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			})
			err = seg.Wait()
			reqs = reqs[completedReqs:]
			if len(reqs) == 0 {
				close(resCh)
			}
			return processGrpcErr(err)
		}
		return backoff.Retry(op, grpcBackOff(ctx))
	})

	if err := eg.Wait(); err != nil {
		return dlLocations{}, err
	}
	return res, nil
}

func (rcs *RemoteChunkStore) readChunksAndCache(ctx context.Context, hashes hash.HashSet, notCached []hash.Hash, found func(context.Context, nbs.CompressedChunk)) error {
	// get the locations where the chunks can be downloaded from
	dlLocs, err := rcs.getDLLocs(ctx, notCached)
	if err != nil {
		return err
	}

	// channel to receive chunks on
	chunkChan := make(chan nbs.CompressedChunk, 128)

	toSend := make(map[hash.Hash]struct{}, len(notCached))
	for _, h := range notCached {
		toSend[h] = struct{}{}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	// start a go routine to receive the downloaded chunks on
	eg.Go(func() error {
		for {
			select {
			case chunk, ok := <-chunkChan:
				if !ok {
					return nil
				}
				if rcs.cache.PutChunk(chunk) {
					return fmt.Errorf("cache full")
				}
				h := chunk.Hash()

				if _, send := toSend[h]; send {
					found(egCtx, chunk)
				}
			case <-egCtx.Done():
				return nil
			}
		}
	})

	// download the chunks and close the channel after
	eg.Go(func() error {
		defer close(chunkChan)
		return rcs.downloadChunks(egCtx, dlLocs, chunkChan)
	})

	// wait for all the results to finish processing
	return eg.Wait()
}

func (rcs *RemoteChunkStore) downloadChunks(ctx context.Context, dlLocs dlLocations, chunkChan chan nbs.CompressedChunk) error {
	resourceGets := dlLocs.ranges

	gets := aggregateDownloads(chunkAggDistance, resourceGets)

	sort.Slice(gets, func(i, j int) bool {
		return gets[j].RangeLen() < gets[i].RangeLen()
	})

	toUrl := func(ctx context.Context, lastError error, resourcePath string) (string, error) {
		return dlLocs.refreshes[resourcePath].GetURL(ctx, lastError, rcs.client)
	}

	stats := remotestorage.StatsFactory()

	eg, ctx := errgroup.WithContext(ctx)

	// loop over all the gets that need to be downloaded and create a work function for each
	work := make([]func() error, len(gets))
	largeCutoff := -1
	for i, get := range gets {
		work[i] = get.GetDownloadFunc(ctx, stats, rcs.httpFetcher, chunkChan, toUrl)
		if get.RangeLen() >= uint64(rcs.concurrency.LargeFetchSize) {
			largeCutoff = i
		}
	}

	// execute the work
	eg.Go(func() error {
		return concurrentExec(work[0:largeCutoff+1], rcs.concurrency.ConcurrentLargeFetches)
	})
	eg.Go(func() error {
		return concurrentExec(work[largeCutoff+1:], rcs.concurrency.ConcurrentSmallFetches)
	})

	defer func() {
		remotestorage.StatsFlusher(stats)
	}()
	return eg.Wait()
}

func (rcs *RemoteChunkStore) Has(ctx context.Context, h hash.Hash) (bool, error) {
	fmt.Println("calling Has")
	hashes := hash.HashSet{h: struct{}{}}
	absent, err := rcs.HasMany(ctx, hashes)

	if err != nil {
		return false, err
	}

	return len(absent) == 0, nil
}

func (rcs *RemoteChunkStore) HasMany(ctx context.Context, hashes hash.HashSet) (hash.HashSet, error) {
	fmt.Println("calling HasMany")

	notCached := rcs.cache.Has(hashes)

	if len(notCached) == 0 {
		return notCached, nil
	}

	// convert the set to a slice of hashes and a corresponding slice of the byte encoding for those hashes
	hashSl, byteSl := remotestorage.HashSetToSlices(notCached)

	absent := make(hash.HashSet)
	var found []nbs.CompressedChunk
	var err error

	batchItr(len(hashSl), maxHasManyBatchSize, func(st, end int) (stop bool) {
		// slice the slices into a batch of hashes
		currHashSl := hashSl[st:end]
		currByteSl := byteSl[st:end]

		// send a request to the remote api to determine which chunks the remote api already has
		req := &remotesapi.HasChunksRequest{Hashes: currByteSl, RepoPath: rcs.peerID}
		var resp *remotesapi.HasChunksResponse
		resp, err = rcs.client.HasChunks(ctx, req)
		if err != nil {
			err = remotestorage.NewRpcError(err, "HasChunks", rcs.peerID, req)
			return true
		}

		numAbsent := len(resp.Absent)
		sort.Slice(resp.Absent, func(i, j int) bool {
			return resp.Absent[i] < resp.Absent[j]
		})

		// loop over every hash in the current batch, and if they are absent from the remote host add them to the
		// absent set, otherwise append them to the found slice
		for i, j := 0, 0; i < len(currHashSl); i++ {
			currHash := currHashSl[i]

			nextAbsent := -1
			if j < numAbsent {
				nextAbsent = int(resp.Absent[j])
			}

			if i == nextAbsent {
				absent[currHash] = struct{}{}
				j++
			} else {
				c := nbs.ChunkToCompressedChunk(chunks.NewChunkWithHash(currHash, []byte{}))
				found = append(found, c)
			}
		}

		return false
	})

	if err != nil {
		return nil, err
	}

	if len(found)+len(absent) != len(notCached) {
		panic("not all chunks were accounted for")
	}

	if len(found) > 0 {
		if rcs.cache.Put(found) {
			return hash.HashSet{}, remotestorage.ErrCacheCapacityExceeded
		}
	}

	return absent, nil
}

func (rcs *RemoteChunkStore) Put(ctx context.Context, c chunks.Chunk, getAddrs chunks.GetAddrsCb) error {
	fmt.Println("calling Put")
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) Version() string {
	fmt.Println("calling Version: ", rcs.nbfVersion)
	return rcs.nbfVersion
}

func (rcs *RemoteChunkStore) Rebase(ctx context.Context) error {
	fmt.Println("calling Rebase")
	return fmt.Errorf("not supported")
}

func (rcs *RemoteChunkStore) loadRoot(ctx context.Context) error {
	req := &remotesapi.RootRequest{RepoPath: rcs.peerID}
	resp, err := rcs.client.Root(ctx, req)
	if err != nil {
		return remotestorage.NewRpcError(err, "Root", rcs.peerID, req)
	}
	rcs.root = hash.New(resp.RootHash)
	return nil
}

func (rcs *RemoteChunkStore) Root(ctx context.Context) (hash.Hash, error) {
	fmt.Println("calling Root")
	return rcs.root, nil
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

func (rcs *RemoteChunkStore) getRepoId() *remotesapi.RepoId {
	return &remotesapi.RepoId{Org: rcs.peerID, RepoName: "protos"}
}

//
// TableFileStore implementation
//

func (rcs *RemoteChunkStore) Sources(ctx context.Context) (hash.Hash, []chunks.TableFile, []chunks.TableFile, error) {
	fmt.Println("calling Sources")
	id := rcs.getRepoId()
	req := &remotesapi.ListTableFilesRequest{RepoId: id, RepoPath: "", RepoToken: ""}
	resp, err := rcs.client.ListTableFiles(ctx, req)
	if err != nil {
		return hash.Hash{}, nil, nil, fmt.Errorf("failed to list table files: %w", err)
	}
	sourceFiles := getTableFiles(rcs.client, resp.TableFileInfo)
	// TODO: remove this
	for _, nfo := range resp.TableFileInfo {
		fmt.Println(nfo)
	}
	appendixFiles := getTableFiles(rcs.client, resp.AppendixTableFileInfo)
	return hash.New(resp.RootHash), sourceFiles, appendixFiles, nil
}

func getTableFiles(client Client, infoList []*remotesapi.TableFileInfo) []chunks.TableFile {
	tableFiles := make([]chunks.TableFile, 0)
	for _, nfo := range infoList {
		tableFiles = append(tableFiles, RemoteTableFile{client, nfo})
	}
	return tableFiles
}

func (rcs *RemoteChunkStore) Size(ctx context.Context) (uint64, error) {
	fmt.Println("calling Size")
	return rcs.repoSize, nil
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
	return chunks.TableFileStoreOps{
		CanRead:  true,
		CanWrite: false,
		CanPrune: false,
		CanGC:    false,
	}
}
