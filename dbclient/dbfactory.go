package dbclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/datas"
	"github.com/dolthub/dolt/go/store/prolly/tree"
	"github.com/dolthub/dolt/go/store/types"
	"github.com/protosio/distributeddolt/p2p"
)

type CustomFactory struct {
	P2P *p2p.P2P
}

func (fact CustomFactory) PrepareDB(ctx context.Context, nbf *types.NomsBinFormat, urlObj *url.URL, params map[string]interface{}) error {
	// nothing to prepare
	return nil
}

func (fact CustomFactory) CreateDB(ctx context.Context, nbf *types.NomsBinFormat, urlObj *url.URL, params map[string]interface{}) (datas.Database, types.ValueReadWriter, tree.NodeStore, error) {
	var db datas.Database

	cs, err := fact.newChunkStore(urlObj.Host, nbf.VersionString())
	if err != nil {
		return nil, nil, nil, err
	}

	vrw := types.NewValueStore(cs)
	ns := tree.NewNodeStore(cs)
	db = datas.NewTypesDatabase(vrw, ns)

	return db, vrw, ns, nil
}

func (fact CustomFactory) newChunkStore(peerID string, nbfVersion string) (chunks.ChunkStore, error) {

	client, err := fact.P2P.GetClient(peerID)
	if err != nil {
		return nil, fmt.Errorf("could not get client for '%s': %w", peerID, err)
	}

	cs, err := NewRemoteChunkStore(fact.P2P, client, peerID, nbfVersion)
	if err != nil {
		return nil, fmt.Errorf("could not create remote cs for '%s': %w", peerID, err)
	}
	return cs, err
}
