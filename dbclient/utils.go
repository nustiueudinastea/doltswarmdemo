package dbclient

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
	"github.com/dolthub/dolt/go/libraries/doltcore/remotestorage"
	"github.com/protosio/distributeddolt/proto"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	grpcRetries = 5
)

type locationRefresh struct {
	RefreshAfter   time.Time
	RefreshRequest *remotesapi.RefreshTableFileUrlRequest
	URL            string
	lastRefresh    time.Time
	mu             *sync.Mutex
}

func (r *locationRefresh) Add(resp *remotesapi.DownloadLoc) {
	if r.URL == "" {
		r.URL = resp.Location.(*remotesapi.DownloadLoc_HttpGetRange).HttpGetRange.Url
	}
	if resp.RefreshAfter == nil {
		return
	}
	respTime := resp.RefreshAfter.AsTime()
	if (r.RefreshAfter == time.Time{}) || respTime.After(r.RefreshAfter) {
		r.RefreshAfter = resp.RefreshAfter.AsTime()
		r.RefreshRequest = resp.RefreshRequest
		r.URL = resp.Location.(*remotesapi.DownloadLoc_HttpGetRange).HttpGetRange.Url
	}
}

var refreshTableFileURLRetryDuration = 5 * time.Second

func (r *locationRefresh) GetURL(ctx context.Context, lastError error, client remotesapi.ChunkStoreServiceClient) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.RefreshRequest != nil {
		now := time.Now()
		wantsRefresh := now.After(r.RefreshAfter) || errors.Is(lastError, remotestorage.HttpError)
		canRefresh := time.Since(r.lastRefresh) > refreshTableFileURLRetryDuration
		if wantsRefresh && canRefresh {
			resp, err := client.RefreshTableFileUrl(ctx, r.RefreshRequest)
			if err != nil {
				return r.URL, err
			}
			r.RefreshAfter = resp.RefreshAfter.AsTime()
			r.URL = resp.Url
			r.lastRefresh = now
		}
	}
	return r.URL, nil
}

type dlLocations struct {
	ranges    map[string]*remotestorage.GetRange
	refreshes map[string]*locationRefresh
}

func (l *dlLocations) Add(resp *remotesapi.DownloadLoc) {
	gr := (*remotestorage.GetRange)(resp.Location.(*remotesapi.DownloadLoc_HttpGetRange).HttpGetRange)
	path := gr.ResourcePath()
	if v, ok := l.ranges[path]; ok {
		v.Append(gr)
		l.refreshes[path].Add(resp)
	} else {
		l.ranges[path] = gr
		refresh := &locationRefresh{mu: new(sync.Mutex)}
		refresh.Add(resp)
		l.refreshes[path] = refresh
	}
}

func batchItr(elemCount, batchSize int, cb func(start, end int) (stop bool)) {
	for st, end := 0, batchSize; st < elemCount; st, end = end, end+batchSize {
		if end > elemCount {
			end = elemCount
		}

		stop := cb(st, end)

		if stop {
			break
		}
	}
}

func processGrpcErr(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.FailedPrecondition,
		codes.Unimplemented,
		codes.OutOfRange,
		codes.Unauthenticated:
		return backoff.Permanent(err)
	}

	return err
}

func grpcBackOff(ctx context.Context) backoff.BackOff {
	ret := backoff.NewExponentialBackOff()
	return backoff.WithContext(backoff.WithMaxRetries(ret, grpcRetries), ctx)
}

func aggregateDownloads(aggDistance uint64, resourceGets map[string]*remotestorage.GetRange) []*remotestorage.GetRange {
	var res []*remotestorage.GetRange
	for _, resourceGet := range resourceGets {
		res = append(res, resourceGet.SplitAtGaps(aggDistance)...)
	}
	return res
}

func concurrentExec(work []func() error, concurrency int) error {
	if concurrency <= 0 {
		panic("Invalid argument")
	}
	if len(work) == 0 {
		return nil
	}
	if len(work) < concurrency {
		concurrency = len(work)
	}

	ch := make(chan func() error)

	eg, ctx := errgroup.WithContext(context.Background())

	// Push the work...
	eg.Go(func() error {
		defer close(ch)
		for _, w := range work {
			select {
			case ch <- w:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Do the work...
	for i := 0; i < concurrency; i++ {
		eg.Go(func() error {
			for {
				select {
				case w, ok := <-ch:
					if !ok {
						return nil
					}
					if err := w(); err != nil {
						return err
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
	}

	return eg.Wait()
}

func copyFromResponse(w *io.PipeWriter, res proto.FileDownloader_DownloadClient) {
	message := new(proto.DownloadResponse)
	var err error
	for {
		err = res.RecvMsg(message)
		if err == io.EOF {
			_ = w.Close()
			break
		}
		if err != nil {
			_ = w.CloseWithError(err)
			break
		}
		if len(message.GetChunk()) > 0 {
			_, err = w.Write(message.Chunk)
			if err != nil {
				_ = res.CloseSend()
				break
			}
		}
		message.Chunk = message.Chunk[:0]
	}
}
