package setup

import (
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/the127/dockyard/internal/commands"
)

func Mediator(dc *ioc.DependencyCollection) {
	mediator := mediatr.NewMediator()

	mediatr.RegisterHandler(mediator, commands.HandleCreateTenant)

	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) mediatr.Mediator {
		return mediator
	})
}
