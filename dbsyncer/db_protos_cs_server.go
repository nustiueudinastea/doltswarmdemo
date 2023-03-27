package main

// import (
// 	"context"

// 	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
// 	"google.golang.org/grpc"
// )

// func NewChunkStoreProtosServer(cc grpc.ClientConnInterface, p2pManager *P2P) remotesapi.ChunkStoreServiceClient {

// 	csServer := &chunkStoreProtosServer{cc}

// 	p2pManager.AddRPCHandler("/protos/api/v1alpha1/db/GetRepoMetadata", &rpcHandler{Func: csServer.GetRepoMetadata, RequestStruct: &PingReq{}})

// 	return csServer
// }

// type chunkStoreProtosServer struct {
// 	cc grpc.ClientConnInterface
// }

// func (c *chunkStoreProtosServer) GetRepoMetadata(ctx context.Context, in *remotesapi.GetRepoMetadataRequest, opts ...grpc.CallOption) (*remotesapi.GetRepoMetadataResponse, error) {
// 	out := new(remotesapi.GetRepoMetadataResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/GetRepoMetadata", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) HasChunks(ctx context.Context, in *remotesapi.HasChunksRequest, opts ...grpc.CallOption) (*remotesapi.HasChunksResponse, error) {
// 	out := new(remotesapi.HasChunksResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/HasChunks", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) GetDownloadLocations(ctx context.Context, in *remotesapi.GetDownloadLocsRequest, opts ...grpc.CallOption) (*remotesapi.GetDownloadLocsResponse, error) {
// 	out := new(remotesapi.GetDownloadLocsResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/GetDownloadLocations", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) StreamDownloadLocations(ctx context.Context, opts ...grpc.CallOption) (remotesapi.ChunkStoreService_StreamDownloadLocationsClient, error) {
// 	stream, err := c.cc.NewStream(ctx, &remotesapi.ChunkStoreService_ServiceDesc.Streams[0], "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/StreamDownloadLocations", opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	x := &chunkStoreServiceStreamDownloadLocationsClient{stream}
// 	return x, nil
// }

// type chunkStoreServiceStreamDownloadLocationsClient struct {
// 	grpc.ClientStream
// }

// func (x *chunkStoreServiceStreamDownloadLocationsClient) Send(m *remotesapi.GetDownloadLocsRequest) error {
// 	return x.ClientStream.SendMsg(m)
// }

// func (x *chunkStoreServiceStreamDownloadLocationsClient) Recv() (*remotesapi.GetDownloadLocsResponse, error) {
// 	m := new(remotesapi.GetDownloadLocsResponse)
// 	if err := x.ClientStream.RecvMsg(m); err != nil {
// 		return nil, err
// 	}
// 	return m, nil
// }

// func (c *chunkStoreProtosServer) GetUploadLocations(ctx context.Context, in *remotesapi.GetUploadLocsRequest, opts ...grpc.CallOption) (*remotesapi.GetUploadLocsResponse, error) {
// 	out := new(remotesapi.GetUploadLocsResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/GetUploadLocations", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) Rebase(ctx context.Context, in *remotesapi.RebaseRequest, opts ...grpc.CallOption) (*remotesapi.RebaseResponse, error) {
// 	out := new(remotesapi.RebaseResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/Rebase", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) Root(ctx context.Context, in *remotesapi.RootRequest, opts ...grpc.CallOption) (*remotesapi.RootResponse, error) {
// 	out := new(remotesapi.RootResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/Root", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) Commit(ctx context.Context, in *remotesapi.CommitRequest, opts ...grpc.CallOption) (*remotesapi.CommitResponse, error) {
// 	out := new(remotesapi.CommitResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/Commit", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) ListTableFiles(ctx context.Context, in *remotesapi.ListTableFilesRequest, opts ...grpc.CallOption) (*remotesapi.ListTableFilesResponse, error) {
// 	out := new(remotesapi.ListTableFilesResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/ListTableFiles", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) RefreshTableFileUrl(ctx context.Context, in *remotesapi.RefreshTableFileUrlRequest, opts ...grpc.CallOption) (*remotesapi.RefreshTableFileUrlResponse, error) {
// 	out := new(remotesapi.RefreshTableFileUrlResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/RefreshTableFileUrl", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

// func (c *chunkStoreProtosServer) AddTableFiles(ctx context.Context, in *remotesapi.AddTableFilesRequest, opts ...grpc.CallOption) (*remotesapi.AddTableFilesResponse, error) {
// 	out := new(remotesapi.AddTableFilesResponse)
// 	err := c.cc.Invoke(ctx, "/dolt.services.remotesapi.v1alpha1.ChunkStoreService/AddTableFiles", in, out, opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }
