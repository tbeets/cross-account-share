package poc

import (
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/tbeets/npoci"
)

type RetailFleet struct {
	// Our servers.
	servers []*server.Server
	// Beacon for the DNA service.
	beacon *time.Ticker
	// Used to measure full cycle request/response times.
	rrTime sync.Map
	// Signal we are done.
	done chan bool
}

var rf *RetailFleet

func StartRetailFleet(projBaseDir string) *RetailFleet {
	rf = &RetailFleet{servers: make([]*server.Server, 1), done: make(chan bool)}
	rf.servers[0], _ = npoci.Up(projBaseDir + "/config/cloud_server.conf")

	for _, s := range rf.servers {
		log.Printf("  Server: [%q]\n", s.ClientURL())
	}

	return rf
}

func StopRetailFleet(rf *RetailFleet) {
	for _, s := range rf.servers {
		log.Printf("  Shutting Down: [%q]\n", s.ClientURL())
		s.Shutdown()
	}

	return
}
