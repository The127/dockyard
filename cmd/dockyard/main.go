package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/the127/dockyard/internal/args"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/server"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/setup"
	"github.com/the127/dockyard/internal/utils"
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

	initApp(dp)

	server.Serve(dp, config.C.Server, hostBlobApi)
	waitForExit()
}

func waitForExit() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func initApp(dp *ioc.DependencyProvider) {
	scope := dp.NewScope()
	defer utils.PanicOnError(scope.Close, "closing scope")

	ctx := middlewares.ContextWithScope(context.Background(), scope)

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		panic(fmt.Errorf("failed to get transaction: %w", err))
	}

	anyTenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter())
	if err != nil {
		panic(fmt.Errorf("failed to get any tenant: %w", err))
	}
	if anyTenant != nil {
		// app already initialized
		return
	}

	mediator := ioc.GetDependency[mediatr.Mediator](scope)
	_, err = mediatr.Send[*commands.CreateTenantResponse](ctx, mediator, commands.CreateTenant{
		Slug:            config.C.InitialTenant.Slug,
		DisplayName:     config.C.InitialTenant.DisplayName,
		OidcClient:      config.C.InitialTenant.Oidc.Client,
		OidcIssuer:      config.C.InitialTenant.Oidc.Issuer,
		OidcRoleClaim:   config.C.InitialTenant.Oidc.RoleClaim,
		OidcRoleFormat:  string(config.C.InitialTenant.Oidc.RoleClaimFormat),
		OidcRoleMapping: config.C.InitialTenant.Oidc.RoleClaimMapping,
	})
	if err != nil {
		panic(fmt.Errorf("failed to create initial tenant: %w", err))
	}
}
