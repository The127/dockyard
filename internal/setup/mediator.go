package setup

import (
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/queries"
)

func Mediator(dc *ioc.DependencyCollection) {
	mediator := mediatr.NewMediator()

	mediatr.RegisterHandler(mediator, commands.HandleCreateTenant)
	mediatr.RegisterHandler(mediator, queries.HandleListTenants)
	mediatr.RegisterHandler(mediator, queries.HandleGetTenant)

	mediatr.RegisterHandler(mediator, commands.HandleCreateProject)
	mediatr.RegisterHandler(mediator, queries.HandleListProjects)
	mediatr.RegisterHandler(mediator, queries.HandleGetProject)

	mediatr.RegisterHandler(mediator, commands.HandleCreateRepository)
	mediatr.RegisterHandler(mediator, queries.HandleListRepositories)
	mediatr.RegisterHandler(mediator, queries.HandleGetRepository)

	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) mediatr.Mediator {
		return mediator
	})
}
