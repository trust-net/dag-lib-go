package main

import (
	"flag"
	"time"
	"os"
	"fmt"
	"encoding/json"
	"github.com/trust-net/dag-lib-go/stack/p2p"
)

func main() {
	fileName := flag.String("config", "", "config file name")
	flag.Parse()
	if len(*fileName) == 0 {
		fmt.Printf("Missing required parameter \"config\"\n")
		return
	}
	// open the config file
	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Printf("Failed to open config file: %s\n", err)
		return
	}
	data := make([]byte, 2048)
	// read config data from file
	config := p2p.Config{}
	if count, err := file.Read(data); err == nil {
		data = data[:count]
		// parse json data into structure
		if err := json.Unmarshal(data, &config); err != nil {
			fmt.Printf("Failed to parse config data: %s\n", err)
			return
		}
	} else {
		fmt.Printf("Failed to read config file: %s\n", err)
		return
	}
	var p2pLayer p2p.Layer
	signal := make(chan struct{})
	if p2pLayer, err = p2p.NewDEVp2pLayer(config, func (peer p2p.Peer) error {
		fmt.Printf("Connected with new peer: %s %s\n", peer.Name(), peer.String())
		// sleep for 5 seconds
		fmt.Printf("Waiting 5 seconds .")
		for i:= 0; i<5; i++ {
			time.Sleep(1*time.Second)
			fmt.Printf(" .")
		}
		fmt.Printf("\n")
		fmt.Printf("Disconnecting with new peer: %s %s\n", peer.Name(), peer.String())
		peer.Disconnect()
		signal <- struct{}{}
		return nil
	}); err != nil {
		fmt.Printf("Failed to create P2P layer instance: %s\n", err)
		return
	}
	// start the node
	if err = p2pLayer.Start(); err != nil {
		fmt.Printf("Failed to start P2P node: %s\n", err)
		return
	}
	// wait for signal that we connected and disconnected with a peer
	fmt.Printf("P2P Layer %s created, waiting for a peer connection...\n", p2pLayer.Self())
	<- signal
	fmt.Printf("Peer connection successfully completed... Exiting.\n")
}