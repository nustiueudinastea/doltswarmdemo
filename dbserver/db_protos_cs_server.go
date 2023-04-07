// Copyright 2019 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dbserver

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
	"github.com/dolthub/dolt/go/libraries/doltcore/remotestorage"
	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/types"
)

var ErrUnimplemented = errors.New("unimplemented")

const RepoPathField = "repo_path"

type RemoteChunkStore struct {
	HttpHost   string
	httpScheme string

	csCache DBCache
	bucket  string
	lgr     *logrus.Entry
	remotesapi.UnimplementedChunkStoreServiceServer
}

func NewProtosChunkStore(lgr *logrus.Entry, csCache DBCache) *RemoteChunkStore {
	return &RemoteChunkStore{
		csCache: csCache,
		bucket:  "",
		lgr: lgr.WithFields(logrus.Fields{
			"service": "ChunkStoreService",
		}),
	}
}

type repoRequest interface {
	GetRepoId() *remotesapi.RepoId
	GetRepoPath() string
}

func getRepoPath(req repoRequest) string {
	if req.GetRepoPath() != "" {
		return req.GetRepoPath()
	}
	if repoId := req.GetRepoId(); repoId != nil {
		return repoId.Org + "/" + repoId.RepoName
	}
	panic("unexpected empty repo_path and nil repo_id")
}

func (rs *RemoteChunkStore) HasChunks(ctx context.Context, req *remotesapi.HasChunksRequest) (*remotesapi.HasChunksResponse, error) {
	logger := getReqLogger(rs.lgr, "HasChunks")
	if err := ValidateHasChunksRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	repoPath := getRepoPath(req)
	logger = logger.WithField(RepoPathField, repoPath)
	defer func() { logger.Info("finished") }()

	cs, err := rs.getStore(logger, repoPath)
	if err != nil {
		return nil, err
	}

	hashes, hashToIndex := remotestorage.ParseByteSlices(req.Hashes)

	absent, err := cs.HasMany(ctx, hashes)
	if err != nil {
		logger.WithError(err).Error("error calling HasMany")
		return nil, status.Error(codes.Internal, "HasMany failure:"+err.Error())
	}

	indices := make([]int32, len(absent))

	n := 0
	for h := range absent {
		indices[n] = int32(hashToIndex[h])
		n++
	}

	resp := &remotesapi.HasChunksResponse{
		Absent: indices,
	}

	logger = logger.WithFields(logrus.Fields{
		"num_requested": len(hashToIndex),
		"num_absent":    len(indices),
	})

	return resp, nil
}

func (rs *RemoteChunkStore) GetDownloadLocations(ctx context.Context, req *remotesapi.GetDownloadLocsRequest) (*remotesapi.GetDownloadLocsResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "HTTP download locations are not supported.")
}

func (rs *RemoteChunkStore) StreamDownloadLocations(stream remotesapi.ChunkStoreService_StreamDownloadLocationsServer) error {
	ologger := getReqLogger(rs.lgr, "StreamDownloadLocations")
	numMessages := 0
	numHashes := 0
	numUrls := 0
	numRanges := 0
	defer func() {
		ologger.WithFields(logrus.Fields{
			"num_messages":  numMessages,
			"num_requested": numHashes,
			"num_urls":      numUrls,
			"num_ranges":    numRanges,
		}).Info("finished")
	}()
	logger := ologger

	md, _ := metadata.FromIncomingContext(stream.Context())

	var repoPath string
	var cs RemoteSrvStore
	var prefix string
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		numMessages += 1

		if err := ValidateGetDownloadLocsRequest(req); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}

		nextPath := getRepoPath(req)
		if nextPath != repoPath {
			repoPath = nextPath
			logger = ologger.WithField(RepoPathField, repoPath)
			cs, err = rs.getStore(logger, repoPath)
			if err != nil {
				return err
			}
			prefix, err = rs.getRelativeStorePath(cs)
			if err != nil {
				logger.WithError(err).Error("error getting file store path for chunk store")
				return err
			}
		}

		hashes, _ := remotestorage.ParseByteSlices(req.ChunkHashes)
		if err != nil {
			return err
		}
		numHashes += len(hashes)
		locations, err := cs.GetChunkLocationsWithPaths(hashes)
		if err != nil {
			logger.WithError(err).Error("error getting chunk locations for hashes")
			return err
		}

		var locs []*remotesapi.DownloadLoc
		for loc, hashToRange := range locations {
			if len(hashToRange) == 0 {
				continue
			}

			numUrls += 1
			numRanges += len(hashToRange)

			var ranges []*remotesapi.RangeChunk
			for h, r := range hashToRange {
				hCpy := h
				ranges = append(ranges, &remotesapi.RangeChunk{Hash: hCpy[:], Offset: r.Offset, Length: r.Length})
			}

			url := rs.getDownloadUrl(md, prefix+"/"+loc)
			preurl := url.String()
			logger.WithFields(logrus.Fields{
				"url":        preurl,
				"ranges":     ranges,
				"sealed_url": url.String(),
			}).Trace("generated sealed url")

			getRange := &remotesapi.HttpGetRange{Url: url.String(), Ranges: ranges}
			locs = append(locs, &remotesapi.DownloadLoc{Location: &remotesapi.DownloadLoc_HttpGetRange{HttpGetRange: getRange}})
		}

		if err := stream.Send(&remotesapi.GetDownloadLocsResponse{Locs: locs}); err != nil {
			return err
		}
	}
}

func (rs *RemoteChunkStore) getHost(md metadata.MD) string {
	host := rs.HttpHost
	if strings.HasPrefix(rs.HttpHost, ":") {
		hosts := md.Get(":authority")
		if len(hosts) > 0 {
			host = strings.Split(hosts[0], ":")[0] + rs.HttpHost
		}
	} else if rs.HttpHost == "" {
		hosts := md.Get(":authority")
		if len(hosts) > 0 {
			host = hosts[0]
		}
	}
	return host
}

func (rs *RemoteChunkStore) getDownloadUrl(md metadata.MD, path string) *url.URL {
	host := rs.getHost(md)
	return &url.URL{
		Scheme: rs.httpScheme,
		Host:   host,
		Path:   path,
	}
}

func parseTableFileDetails(req *remotesapi.GetUploadLocsRequest) []*remotesapi.TableFileDetails {
	tfd := req.GetTableFileDetails()

	if len(tfd) == 0 {
		_, hashToIdx := remotestorage.ParseByteSlices(req.TableFileHashes)

		tfd = make([]*remotesapi.TableFileDetails, len(hashToIdx))
		for h, i := range hashToIdx {
			tfd[i] = &remotesapi.TableFileDetails{
				Id:            h[:],
				ContentLength: 0,
				ContentHash:   nil,
			}
		}
	}

	return tfd
}

func (rs *RemoteChunkStore) GetUploadLocations(ctx context.Context, req *remotesapi.GetUploadLocsRequest) (*remotesapi.GetUploadLocsResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "this server only provides read-only access")
}

func (rs *RemoteChunkStore) Rebase(ctx context.Context, req *remotesapi.RebaseRequest) (*remotesapi.RebaseResponse, error) {
	logger := getReqLogger(rs.lgr, "Rebase")
	if err := ValidateRebaseRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	repoPath := getRepoPath(req)
	logger = logger.WithField(RepoPathField, repoPath)
	defer func() { logger.Info("finished") }()

	_, err := rs.getStore(logger, repoPath)
	if err != nil {
		return nil, err
	}

	return &remotesapi.RebaseResponse{}, nil
}

func (rs *RemoteChunkStore) Root(ctx context.Context, req *remotesapi.RootRequest) (*remotesapi.RootResponse, error) {
	logger := getReqLogger(rs.lgr, "Root")
	if err := ValidateRootRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	repoPath := getRepoPath(req)
	logger = logger.WithField(RepoPathField, repoPath)
	defer func() { logger.Info("finished") }()

	cs, err := rs.getStore(logger, repoPath)
	if err != nil {
		return nil, err
	}

	h, err := cs.Root(ctx)
	if err != nil {
		logger.WithError(err).Error("error calling Root on chunk store.")
		return nil, status.Error(codes.Internal, "Failed to get root")
	}

	return &remotesapi.RootResponse{RootHash: h[:]}, nil
}

func (rs *RemoteChunkStore) Commit(ctx context.Context, req *remotesapi.CommitRequest) (*remotesapi.CommitResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "this server only provides read-only access")
}

func (rs *RemoteChunkStore) GetRepoMetadata(ctx context.Context, req *remotesapi.GetRepoMetadataRequest) (*remotesapi.GetRepoMetadataResponse, error) {
	logger := getReqLogger(rs.lgr, "GetRepoMetadata")
	if err := ValidateGetRepoMetadataRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	repoPath := getRepoPath(req)
	logger = logger.WithField(RepoPathField, repoPath)
	defer func() { logger.Info("finished") }()

	cs, err := rs.getOrCreateStore(logger, repoPath, req.ClientRepoFormat.NbfVersion)
	if err != nil {
		return nil, err
	}

	size, err := cs.Size(ctx)
	if err != nil {
		logger.WithError(err).Error("error calling Size")
		return nil, err
	}

	return &remotesapi.GetRepoMetadataResponse{
		NbfVersion:  cs.Version(),
		NbsVersion:  req.ClientRepoFormat.NbsVersion,
		StorageSize: size,
	}, nil
}

func (rs *RemoteChunkStore) ListTableFiles(ctx context.Context, req *remotesapi.ListTableFilesRequest) (*remotesapi.ListTableFilesResponse, error) {
	logger := getReqLogger(rs.lgr, "ListTableFiles")
	if err := ValidateListTableFilesRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	repoPath := getRepoPath(req)
	logger = logger.WithField(RepoPathField, repoPath)
	defer func() { logger.Info("finished") }()

	cs, err := rs.getStore(logger, repoPath)
	if err != nil {
		return nil, err
	}

	root, tables, appendixTables, err := cs.Sources(ctx)
	if err != nil {
		logger.WithError(err).Error("error getting chunk store Sources")
		return nil, status.Error(codes.Internal, "failed to get sources")
	}

	md, _ := metadata.FromIncomingContext(ctx)

	tableFileInfo, err := getTableFileInfo(logger, md, rs, tables, req, cs)
	if err != nil {
		logger.WithError(err).Error("error getting table file info")
		return nil, err
	}

	appendixTableFileInfo, err := getTableFileInfo(logger, md, rs, appendixTables, req, cs)
	if err != nil {
		logger.WithError(err).Error("error getting appendix table file info")
		return nil, err
	}

	logger = logger.WithFields(logrus.Fields{
		"num_table_files":          len(tableFileInfo),
		"num_appendix_table_files": len(appendixTableFileInfo),
	})

	resp := &remotesapi.ListTableFilesResponse{
		RootHash:              root[:],
		TableFileInfo:         tableFileInfo,
		AppendixTableFileInfo: appendixTableFileInfo,
	}

	return resp, nil
}

func getTableFileInfo(
	logger *logrus.Entry,
	md metadata.MD,
	rs *RemoteChunkStore,
	tableList []chunks.TableFile,
	req *remotesapi.ListTableFilesRequest,
	cs RemoteSrvStore,
) ([]*remotesapi.TableFileInfo, error) {
	prefix, err := rs.getRelativeStorePath(cs)
	if err != nil {
		return nil, err
	}
	appendixTableFileInfo := make([]*remotesapi.TableFileInfo, 0)
	for _, t := range tableList {
		url := rs.getDownloadUrl(md, prefix+"/"+t.FileID())

		appendixTableFileInfo = append(appendixTableFileInfo, &remotesapi.TableFileInfo{
			FileId:    t.FileID(),
			NumChunks: uint32(t.NumChunks()),
			Url:       url.String(),
		})
	}
	return appendixTableFileInfo, nil
}

// AddTableFiles updates the remote manifest with new table files without modifying the root hash.
func (rs *RemoteChunkStore) AddTableFiles(ctx context.Context, req *remotesapi.AddTableFilesRequest) (*remotesapi.AddTableFilesResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "this server only provides read-only access")
}

func (rs *RemoteChunkStore) getStore(logger *logrus.Entry, repoPath string) (RemoteSrvStore, error) {
	return rs.getOrCreateStore(logger, repoPath, types.Format_Default.VersionString())
}

func (rs *RemoteChunkStore) getOrCreateStore(logger *logrus.Entry, repoPath, nbfVerStr string) (RemoteSrvStore, error) {
	cs, err := rs.csCache.Get(repoPath, nbfVerStr)
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve chunkstore")
		if errors.Is(err, ErrUnimplemented) {
			return nil, status.Error(codes.Unimplemented, err.Error())
		}
		return nil, err
	}
	if cs == nil {
		logger.Error("internal error getting chunk store; csCache.Get returned nil")
		return nil, status.Error(codes.Internal, "Could not get chunkstore")
	}
	return cs, nil
}

var requestId int32

func incReqId() int {
	return int(atomic.AddInt32(&requestId, 1))
}

func getReqLogger(lgr *logrus.Entry, method string) *logrus.Entry {
	lgr = lgr.WithFields(logrus.Fields{
		"method":      method,
		"request_num": strconv.Itoa(incReqId()),
	})
	lgr.Info("starting request")
	return lgr
}