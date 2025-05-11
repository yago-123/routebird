package k8s

// todo: local types for route info etc

type EventType string

const (
	EventAdd    EventType = "Add"
	EventUpdate           = "Update"
	EventDelete           = "Delete"
)

type Event struct {
	Type EventType
	Obj  any // Could be *v1.Service, *v1.Endpoints, etc.
	// todo: TBD
	Key string // Usually "namespace/name"
}
