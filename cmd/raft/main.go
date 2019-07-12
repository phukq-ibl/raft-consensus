package main

import (
	"flag"
	raft "github.com/phukq/raft-consensus"
	log "github.com/sirupsen/logrus"
	"strings"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	log.SetLevel(log.TraceLevel)
}

func main() {
	peers := flag.String("peers", "", "peers")
	port := flag.String("port", "3000", "Listen port")
	id := flag.String("id", "", "server id")

	flag.Parse()

	if len(*id) == 0 {
		panic("Not provide server id")
	}
	listenerAddr := "localhost:" + *port

	log.Info("Listen on: ", listenerAddr)

	server := raft.NewServer(*id, listenerAddr)
	go server.Start()

	if len(*peers) != 0 {
		ips := strings.Split(*peers, ",")
		for _, ip := range ips {
			err := server.AddPeer(ip)
			if err != nil {
				log.Error("Add peer error")
			}
		}
	}

	select {}
}
