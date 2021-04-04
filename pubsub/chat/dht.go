package main

import (
	"fmt"
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	//"github.com/libp2p/go-libp2p-core/peer"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	dht_opts "github.com/libp2p/go-libp2p-kad-dht/opts"
)

// how should bootstrapping nodes be selected?
// multiple dhts for multiple stocks or just concatentate stock ticker and nick?
// use local files for datastore, or just leave as in memory?
// validate with key pairs or just default trust?

type StockReviewDHT struct {
	kadDHT *dht.IpfsDHT
	//ticker string
}

type StockReviewValidator struct{}

func (sv StockReviewValidator) Validate(key string, value []byte) error {
	fmt.Printf("Validate: %s - %s", key, string(value))

	//TODO: any validation checks performed here
	return nil
}

func (sv StockReviewValidator) Select(key string, values [][]byte) (int, error) {
	strs := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		strs[i] = string(values[i])
	}
	fmt.Printf("Validator Select: %s - %v", key, strs)

	return 0, nil
}

func (srdht StockReviewDHT) initKadDHT(ctx context.Context, h host.Host) error {
	//need condition in here to prevent initializing dht twice

	var err error
	srdht.kadDHT, err = dht.New(ctx, h, dht_opts.Validator(StockReviewValidator{}))

	return err
}

// func updateKadDHT(ctx context.Context, pi peer.AddrInfo) {
// 	seems this method is deprecated ? or something
// 	need a way to connect newly discovered peers to dht
// 	//kadDHT.update(ctx, pi.ID())
// }

func (srdht StockReviewDHT) putValue(ctx context.Context, key string, value []byte) error {
	return srdht.kadDHT.PutValue(ctx, key, value, dht.Quorum(1));
}

func (srdht StockReviewDHT) getValue(ctx context.Context, key string) ([]byte, error) {
	return srdht.kadDHT.GetValue(ctx, key, dht.Quorum(1))
}