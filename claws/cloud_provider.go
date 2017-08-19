package claws

import (
	"errors"
)

// CloudProvider wraps the cloud provider functions
type CloudProvider interface {
	ValidateTemplate(tplBody string) ([]string, error)
	StackExists(stackName string) (bool, error)
	CreateChangeSet(stackName string, tplBody string, params map[string]string, op ChangeSetOperation) (string, error)
	WaitChangeSetCreated(ID string) error
	ChangeSetChanges(ID string) ([]Change, error)
	ExecuteChangeSet(ID string) error
	WaitStack(stackName string) error
	StackEvents(stackName string) ([]StackEvent, error)
	StackOutputs(stackName string) (map[string]string, error)
}

// ChangeSetOperation enum
type ChangeSetOperation int

const (
	// CreateOperation is a ChangeSetOperation enum value
	CreateOperation ChangeSetOperation = iota
	// UpdateOperation is a ChangeSetOperation enum value
	UpdateOperation
)

// Change is a change that is applied to the stack
type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceID string
	ReplacementNeeded bool
}

// StackEvent is a stack event
type StackEvent struct {
	ID                string
	ResourceType      string
	Status            string
	LogicalResourceID string
	StatusReason      string
}

//ErrNoChange is error that indicate that there are no changes to apply
var ErrNoChange = errors.New("No changes")