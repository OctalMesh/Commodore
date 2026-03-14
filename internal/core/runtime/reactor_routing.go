package runtime

import (
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

func (service *Service) ShouldOperateOnNode(node domain.OrchestrationNode, tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	tagSet := make(map[string]struct{}, len(node.GetTags()))
	for _, tag := range node.GetTags() {
		tagSet[tag] = struct{}{}
	}

	for _, requestedTag := range tags {
		if _, found := tagSet[requestedTag]; found {
			return true
		}
	}
	return false
}

func (service *Service) SelectReactor(node domain.OrchestrationNode) ports.Reactor {
	switch typedNode := node.(type) {
	case *domain.SquadronNode:
		switch typedNode.Reactor.Provider {
		case domain.ReactorProviderNative:
			return service.options.NativeReactor
		case domain.ReactorProviderTilt, "":
			return service.options.TiltReactor
		default:
			return nil
		}
	case *domain.UnitNode:
		switch typedNode.Reactor.Provider {
		case domain.ReactorProviderNative:
			return service.options.NativeReactor
		case domain.ReactorProviderTilt, "":
			return service.options.TiltReactor
		default:
			return nil
		}
	default:
		return nil
	}
}
