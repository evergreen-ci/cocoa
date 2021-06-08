package cocoa

// ECSPod represents a pod that is backed by ECS.
type ECSPod interface {
	// ID is the pod's unique identifier, which must uniquely identify the
	// backing resource in ECS.
	ID() string
	// DefinitionID is the unique identifier for the pod's template definition,
	// which must uniquely identify the backing resource in ECS.
	DefinitionID() string
}

type BasicECSPod struct{}

func (p *BasicECSPod) ID() string {
	return ""
}

func (p *BasicECSPod) DefinitionID() string {
	return ""
}
