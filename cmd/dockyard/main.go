package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/avast/retry-go"
	"github.com/the127/dockyard/internal/args"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/config"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/server"
	"github.com/the127/dockyard/internal/setup"
	"github.com/the127/dockyard/internal/utils"
)

func main() {
	args.Init()
	logging.Init()
	config.Init()

	dc := ioc.NewDependencyCollection()

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return clock.NewSystemClock()
	})

	database := setup.Database(dc, config.C.Database)

	err := retry.Do(
		func() error {
			return database.Migrate()
		},
		retry.Attempts(5),
		retry.Delay(time.Second*5),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(n uint, err error) {
			logging.Logger.Warnf("failed to migrate database: %s, retrying in 5 seconds", err)
		}),
	)
	if err != nil {
		logging.Logger.Panicf("failed to migrate database: %s", err)
	}

	setup.Kv(dc, config.C.Kv)
	setup.Mediator(dc)
	setup.Blob(dc, config.C.Blob)
	setup.Kms(dc, config.C.Kms)

	dp := dc.BuildProvider()

	var hostBlobApi bool
	switch config.C.Blob.Mode {
	case config.BlobStorageModeInMemory:
		hostBlobApi = true

	case config.BlobStorageModeDirectory:
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

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to get transaction: %w", err))
	}

	anyTenant, err := dbContext.Tenants().First(ctx, repositories.NewTenantFilter())
	if err != nil {
		logging.Logger.Panicf("failed to get any tenant: %s", err)
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
		logging.Logger.Panicf("failed to create initial tenant: %s", err)
	}

	err = dbContext.SaveChanges(ctx)
	if err != nil {
		logging.Logger.Panicf("failed to save changes: %s", err)
	}

	logging.Logger.Infof("initial tenant created: %s", config.C.InitialTenant.Slug)
}
