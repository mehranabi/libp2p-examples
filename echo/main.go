/**
*	This is gonna be a very simple p2p app, my first app with libp2p
*	developer: github.com/mehranabi
**/

package echo

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mrand "math/rand"

	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"
)

func main() {
	fmt.Println("Hello! Welcome to p2p-ECHO!")

	golog.SetAllLoggers(gologging.INFO)

	listen := flag.Int("port", 0, "wait for incoming connections")
	target := flag.String("target", "", "target peer to dial")
	insecure := flag.Bool("insecure", false, "use an unencrypted connection")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listen == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	var r io.Reader
	if *seed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(*seed))
	}

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *listen)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	if *insecure {
		opts = append(opts, libp2p.NoSecurity)
	}

	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	multiaddr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().Pretty()))
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(multiaddr)

	fmt.Printf("Your address is %v\n", fullAddr)

	host.SetStreamHandler("/echo/1.0.0", func(s net.Stream) {
		log.Println("Got a new stream!")
		if err := doEcho(s); err != nil {
			log.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	if *target == "" {
		log.Println("listening for connections")
		select {}
	}

	p2pAddr, err := ma.NewMultiaddr(*target)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	pid, err := p2pAddr.ValueForProtocol(ma.P_P2P)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	peerId, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	targetPeerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", peer.IDB58Encode(peerId)))
	targetAddr := p2pAddr.Decapsulate(targetPeerAddr)

	host.Peerstore().AddAddr(peerId, targetAddr, pstore.PermanentAddrTTL)

	log.Println("opening a stream...")

	s, err := host.NewStream(context.Background(), peerId, "/echo/1.0.0")
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	_, err = s.Write([]byte("Hello, world!\n"))
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	out, err := ioutil.ReadAll(s)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}

	fmt.Printf("peer replied: %v\n", string(out))
}

func doEcho(s net.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	log.Printf("read: %s\n", str)
	_, err = s.Write([]byte(str))
	return err
}
