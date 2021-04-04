package main

import (
	"context"
	//"flag" -- add back in for cmd util
	"fmt"
	//"io/ioutil"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery"

	"github.com/libp2p/go-libp2p-core/host"

	// dht "github.com/libp2p/go-libp2p-kad-dht"
	// dht_opts "github.com/libp2p/go-libp2p-kad-dht/opts"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
const DiscoveryServiceTag = "pubsub-chat-example"

func main() {
	// parse some flags to set our nickname and the room to join
	//nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	//roomFlag := flag.String("room", "awesome-chat-room", "name of chat room to join")
	//flag.Parse()

	ctx := context.Background()

	// create a new libp2p Host that listens on a random TCP port
	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}

	//initialize kademlia DHT
	stockDHT := new(StockReviewDHT)
	err = stockDHT.initKadDHT(ctx, h)
	if err != nil {
		panic(err)
	}

	// setup local mDNS discovery
	err = setupDiscovery(ctx, h)
	if err != nil {
		panic(err)
	}

	//nick := defaultNick(h.ID())
	for true {
		time.Sleep(2*time.Second)
		stockDHT.kadDHT.RoutingTable().Print()
	}
}

// printErr is like fmt.Printf, but writes to stderr.
func printErr(m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
}

// defaultNick generates a nickname based on the $USER environment variable and
// the last 8 chars of a peer ID.
func defaultNick(p peer.ID) string {
	return fmt.Sprintf("%s-%s", os.Getenv("USER"), shortID(p))
}

// shortID returns the last 8 chars of a base58-encoded peer id.
func shortID(p peer.ID) string {
	pretty := p.Pretty()
	return pretty[len(pretty)-8:]
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID.Pretty())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
	
	//TODO: update dht store
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(ctx context.Context, h host.Host) error {
	// setup mDNS discovery to find local peers
	disc, err := discovery.NewMdnsService(ctx, h, DiscoveryInterval, DiscoveryServiceTag)
	if err != nil {
		return err
	}

	n := discoveryNotifee{h: h}
	disc.RegisterNotifee(&n)
	return nil
}
