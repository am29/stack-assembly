package assembly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/molecule-man/stack-assembly/conf"
)

func Sync(cfg conf.Config, nonInteractive bool) {
	stackCfgs, err := cfg.StackConfigsSortedByExecOrder()
	MustSucceed(err)

	MustSucceed(cfg.Hooks.Pre.Exec())

	for _, stackCfg := range stackCfgs {
		logger := cli.PrefixedLogger(fmt.Sprintf("[%s] ", stackCfg.Name))

		logger.Info("Synchronizing template")

		cs, err := cfg.ChangeSetFromStackConfig(stackCfg)
		MustSucceed(err)

		chSet, err := cs.Register()

		if paramerr, ok := err.(*awscf.ParametersMissingError); ok {
			logger.Warn(paramerr.Error())
			for _, p := range paramerr.MissingParameters {
				response, rerr := cli.Ask("Enter %s: ", p)
				MustSucceed(rerr)
				cs.WithParameter(p, response)
			}

			chSet, err = cs.Register()
		}

		if err == awscf.ErrNoChange {
			logger.Info("No changes to be synchronized")
		} else {
			MustSucceed(err)

			logger.Infof("Change set is created: %s", chSet.ID)

			showChanges(chSet.Changes)

			if !nonInteractive {
				letUserChooseNextAction(cs)
			}

			MustSucceed(cfg.Hooks.PreSync.Exec())
			MustSucceed(stackCfg.Hooks.PreSync.Exec())

			if chSet.IsUpdate {
				MustSucceed(cfg.Hooks.PreUpdate.Exec())
				MustSucceed(stackCfg.Hooks.PreUpdate.Exec())
			} else {
				MustSucceed(cfg.Hooks.PreCreate.Exec())
				MustSucceed(stackCfg.Hooks.PreCreate.Exec())
			}

			wait := showEvents(cs.Stack(), logger)

			err = chSet.Exec()

			wait <- true
			<-wait

			MustSucceed(err)

			MustSucceed(cfg.Hooks.PostSync.Exec())
			MustSucceed(stackCfg.Hooks.PostSync.Exec())

			if chSet.IsUpdate {
				MustSucceed(cfg.Hooks.PostUpdate.Exec())
				MustSucceed(stackCfg.Hooks.PostUpdate.Exec())
			} else {
				MustSucceed(cfg.Hooks.PostCreate.Exec())
				MustSucceed(stackCfg.Hooks.PostCreate.Exec())
			}
			logger.Print(color.Success("Synchronization is complete"))
		}

		for _, r := range stackCfg.Blocked {
			logger.Infof("Blocking resource %s", r)
			err := cs.Stack().BlockResource(r)

			MustSucceed(err)
		}
	}

	MustSucceed(cfg.Hooks.Post.Exec())
}

func showEvents(stack *awscf.Stack, logger *cli.Logger) chan bool {
	wait := make(chan bool)

	if _, err := stack.EventsTrack().FreshEvents(); err != nil {
		logger.Warnf("got an error while requesting stack events: %s", err)
	}

	go func() {
		writer := cli.NewColWriter(cli.Output, " ")

		for {
			events, err := stack.EventsTrack().FreshEvents()
			if err != nil {
				logger.Warnf("got an error while requesting stack events: %s", err)
			}

			for _, e := range events.Reversed() {
				logger.Fprint(writer, sprintEvent(e))
			}
			writer.Flush()

			select {
			case <-wait:
				wait <- true
				return
			default:
				time.Sleep(2 * time.Second)
			}
		}
	}()

	return wait
}

func showChanges(changes []awscf.Change) {
	if len(changes) > 0 {
		t := cli.NewTable()
		t.Header("Action", "Resource Type", "Resource ID", "Replacement needed")

		for _, c := range changes {
			action := color.Neutral(c.Action)
			switch strings.ToLower(c.Action) {
			case "add":
				action = color.Success(c.Action)
			case "remove":
				action = color.Fail(c.Action)
			}

			repl := color.Success(fmt.Sprintf("%t", c.ReplacementNeeded))
			if c.ReplacementNeeded {
				repl = color.Fail(fmt.Sprintf("%t", c.ReplacementNeeded))
			}
			t.Row(action, c.ResourceType, c.LogicalResourceID, repl)

		}

		cli.Print(t.Render())
	}
}

func letUserChooseNextAction(chSet *awscf.ChangeSet) {
	continueSync := false
	for !continueSync {
		err := cli.Prompt([]cli.PromptCmd{
			{
				Description:   "[s]ync",
				TriggerInputs: []string{"s", "sync"},
				Action: func() {
					continueSync = true
				},
			},
			{
				Description:   "[d]iff",
				TriggerInputs: []string{"d", "diff"},
				Action: func() {
					diff, derr := awscf.Diff(chSet)
					MustSucceed(derr)

					cli.Print(diff)
				},
			},
			{
				Description:   "[q]uit",
				TriggerInputs: []string{"q", "quit"},
				Action: func() {
					cli.Error("Interrupted by user")
					MustSucceed(errors.New("sync is cancelled"))
				},
			},
		})
		if err != cli.ErrPromptCommandIsNotKnown {
			MustSucceed(err)
		}
	}
}