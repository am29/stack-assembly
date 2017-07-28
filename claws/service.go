// Package claws provides claws stuff
package claws

import (
	"errors"
	"fmt"
	"sync"

	"github.com/molecule-man/claws/cloudprov"
	"github.com/molecule-man/claws/cloudprov/awsprov"
)

type approver interface {
	Approve() bool
}
type logger interface {
	Print(v ...interface{})
}
type changePresenter interface {
	ShowChanges([]cloudprov.Change)
}

// Service orchestrates synchronization of templates
type Service struct {
	Approver        approver
	Log             logger
	ChangePresenter changePresenter
}

// Sync syncs
func (s *Service) Sync(stackName, body string, params map[string]string) error {
	log := s.logFunc(stackName)

	log("Syncing template")

	cp := awsprov.New()
	chSet, err := New(&cp, stackName, body, params)

	if err != nil {
		if err == cloudprov.ErrNoChange {
			log("No changes to be synced")
			return nil
		}
		return err
	}

	log("Change set is created")

	s.ChangePresenter.ShowChanges(chSet.Changes)

	if !s.Approver.Approve() {
		return errors.New("Sync is cancelled")
	}

	et := chSet.EventsTracker()
	events := et.StartTracking()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for event := range events {
			log(event)
		}

		wg.Done()
	}()

	err = chSet.Exec()
	et.StopTracking()
	wg.Wait()

	log("Sync is finished")

	return err
}

func (s *Service) logFunc(logID string) func(string) {
	return func(msg string) {
		s.Log.Print(fmt.Sprintf("[%s] %s", logID, msg))
	}
}
