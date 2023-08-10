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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/protosio/distributeddolt/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
	"github.com/dolthub/dolt/go/libraries/doltcore/remotestorage"
	"github.com/dolthub/dolt/go/store/types"
)

var ErrUnimplemented = errors.New("unimplemented")
var chunkSize = 1024 * 3

const RepoPathField = "repo_path"

func NewServerChunkStore(logger *logrus.Entry, csCache DBCache, filePath string) *ServerChunkStore {
	return &ServerChunkStore{
		csCache: csCache,
		bucket:  "",
		log: logger.WithFields(logrus.Fields{
			"service": "ChunkStoreService",
		}),
		filePath: filePath,
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

type ServerChunkStore struct {
	csCache  DBCache
	bucket   string
	log      *logrus.Entry
	filePath string

	remotesapi.UnimplementedChunkStoreServiceServer
	proto.UnimplementedFileDownloaderServer
}

func (rs *ServerChunkStore) HasChunks(ctx context.Context, req *remotesapi.HasChunksRequest) (*remotesapi.HasChunksResponse, error) {
	logger := getReqLogger(rs.log, "HasChunks")
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

func (rs *ServerChunkStore) GetDownloadLocations(ctx context.Context, req *remotesapi.GetDownloadLocsRequest) (*remotesapi.GetDownloadLocsResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "HTTP download locations are not supported.")
}

func (rs *ServerChunkStore) StreamDownloadLocations(stream remotesapi.ChunkStoreService_StreamDownloadLocationsServer) error {
	ologger := getReqLogger(rs.log, "StreamDownloadLocations")
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

	var repoPath string
	var cs RemoteSrvStore
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
			// prefix, err = rs.getRelativeStorePath(cs)
			// if err != nil {
			// 	logger.WithError(err).Error("error getting file store path for chunk store")
			// 	return err
			// }
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

			logger.WithFields(logrus.Fields{
				"url":        loc,
				"ranges":     ranges,
				"sealed_url": loc,
			}).Trace("generated sealed url")

			getRange := &remotesapi.HttpGetRange{Url: loc, Ranges: ranges}
			locs = append(locs, &remotesapi.DownloadLoc{Location: &remotesapi.DownloadLoc_HttpGetRange{HttpGetRange: getRange}})
		}

		if err := stream.Send(&remotesapi.GetDownloadLocsResponse{Locs: locs}); err != nil {
			return err
		}
	}
}

func (rs *ServerChunkStore) GetUploadLocations(ctx context.Context, req *remotesapi.GetUploadLocsRequest) (*remotesapi.GetUploadLocsResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "this server only provides read-only access")
}

func (rs *ServerChunkStore) Rebase(ctx context.Context, req *remotesapi.RebaseRequest) (*remotesapi.RebaseResponse, error) {
	logger := getReqLogger(rs.log, "Rebase")
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

func (rs *ServerChunkStore) Root(ctx context.Context, req *remotesapi.RootRequest) (*remotesapi.RootResponse, error) {
	logger := getReqLogger(rs.log, "Root")
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

func (rs *ServerChunkStore) Commit(ctx context.Context, req *remotesapi.CommitRequest) (*remotesapi.CommitResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "this server only provides read-only access")
}

func (rs *ServerChunkStore) GetRepoMetadata(ctx context.Context, req *remotesapi.GetRepoMetadataRequest) (*remotesapi.GetRepoMetadataResponse, error) {
	logger := getReqLogger(rs.log, "GetRepoMetadata")
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

func (rs *ServerChunkStore) ListTableFiles(ctx context.Context, req *remotesapi.ListTableFilesRequest) (*remotesapi.ListTableFilesResponse, error) {
	logger := getReqLogger(rs.log, "ListTableFiles")
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

	tableFileInfo := buildTableFileInfo(tables)
	appendixTableFileInfo := buildTableFileInfo(appendixTables)

	resp := &remotesapi.ListTableFilesResponse{
		RootHash:              root[:],
		TableFileInfo:         tableFileInfo,
		AppendixTableFileInfo: appendixTableFileInfo,
	}

	return resp, nil
}

// AddTableFiles updates the remote manifest with new table files without modifying the root hash.
func (rs *ServerChunkStore) AddTableFiles(ctx context.Context, req *remotesapi.AddTableFilesRequest) (*remotesapi.AddTableFilesResponse, error) {
	return nil, status.Error(codes.PermissionDenied, "this server only provides read-only access")
}

func (rs *ServerChunkStore) getStore(logger *logrus.Entry, repoPath string) (RemoteSrvStore, error) {
	return rs.getOrCreateStore(logger, repoPath, types.Format_Default.VersionString())
}

func (rs *ServerChunkStore) getOrCreateStore(logger *logrus.Entry, repoPath, nbfVerStr string) (RemoteSrvStore, error) {
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

//
// FileDownloaderServer implementation
//

func (rs *ServerChunkStore) Download(req *proto.DownloadRequest, server proto.FileDownloader_DownloadServer) error {
	if req.GetId() == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}

	filePath := filepath.Join(rs.filePath, ".dolt", "noms", req.Id)
	rs.log.Info(fmt.Sprintf("Sending file %s", filePath))
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to send file %s: %w", filePath, err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	md := metadata.New(map[string]string{
		"file-name": req.Id,
		"file-size": strconv.FormatInt(fileInfo.Size(), 10),
	})

	err = server.SendHeader(md)
	if err != nil {
		return status.Error(codes.Internal, "error during sending header")
	}

	chunk := &proto.DownloadResponse{Chunk: make([]byte, chunkSize)}
	var n int

Loop:
	for {
		n, err = f.Read(chunk.Chunk)
		switch err {
		case nil:
		case io.EOF:
			break Loop
		default:
			return status.Errorf(codes.Internal, "io.ReadAll: %v", err)
		}
		chunk.Chunk = chunk.Chunk[:n]
		serverErr := server.Send(chunk)
		if serverErr != nil {
			return status.Errorf(codes.Internal, "server.Send: %v", serverErr)
		}
	}
	return nil
}
