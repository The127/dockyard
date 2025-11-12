package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/args"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/server"
	"github.com/the127/dockyard/internal/setup"
)

func main() {
	args.Init()
	logging.Init()
	config.Init()

	dc := ioc.NewDependencyCollection()

	db := setup.Database(dc, config.C.Database)
	err := db.Migrate()
	if err != nil {
		logging.Logger.Panicf("failed to migrate database: %s", err)
	}

	setup.Kv(dc, config.C.Kv)
	setup.Mediator(dc)
	setup.Blob(dc, config.C.Blob)

	dp := dc.BuildProvider()

	var hostBlobApi bool
	switch config.C.Blob.Mode {
	case config.BlobStorageModeInMemory:
		hostBlobApi = true

	case config.BlobStorageModeS3:
		hostBlobApi = false

	default:
		panic(fmt.Errorf("unsupported blob storage mode: %s", config.C.Blob.Mode))
	}

	server.Serve(dp, config.C.Server, hostBlobApi)
	waitForExit()
}

func waitForExit() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
