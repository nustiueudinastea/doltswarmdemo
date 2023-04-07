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

// ProtosFactory is a DBFactory implementation for creating Protos backed remotes
type ProtosFactory struct {
	p2p *p2p.P2P
}

func (fact ProtosFactory) PrepareDB(ctx context.Context, nbf *types.NomsBinFormat, urlObj *url.URL, params map[string]interface{}) error {
	// nothing to prepare
	return nil
}

// CreateDB creates an AWS backed database
func (fact ProtosFactory) CreateDB(ctx context.Context, nbf *types.NomsBinFormat, urlObj *url.URL, params map[string]interface{}) (datas.Database, types.ValueReadWriter, tree.NodeStore, error) {
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

func (fact ProtosFactory) newChunkStore() (chunks.ChunkStore, error) {

	cs, err := NewRemoteChunkStore(fact.p2p)
	if err != nil {
		return nil, fmt.Errorf("could not create remote cs : %w", err)
	}
	return cs, err
}
