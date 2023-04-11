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
	P2P     *p2p.P2P
	LocalCS chunks.ChunkStore
}

func (fact CustomFactory) PrepareDB(ctx context.Context, nbf *types.NomsBinFormat, urlObj *url.URL, params map[string]interface{}) error {
	// nothing to prepare
	return nil
}

func (fact CustomFactory) CreateDB(ctx context.Context, nbf *types.NomsBinFormat, urlObj *url.URL, params map[string]interface{}) (datas.Database, types.ValueReadWriter, tree.NodeStore, error) {
	var db datas.Database

	cs, err := fact.newChunkStore()
	if err != nil {
		return nil, nil, nil, err
	}

	vrw := types.NewValueStore(cs)
	ns := tree.NewNodeStore(cs)
	db = datas.NewTypesDatabase(vrw, ns)

	return db, vrw, ns, nil
}

func (fact CustomFactory) newChunkStore() (chunks.ChunkStore, error) {

	cs, err := NewRemoteChunkStore(fact.P2P, fact.LocalCS)
	if err != nil {
		return nil, fmt.Errorf("could not create remote cs : %w", err)
	}
	return cs, err
}
