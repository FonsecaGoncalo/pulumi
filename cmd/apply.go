// Copyright 2016 Marapongo, Inc. All rights reserved.

package cmd

import (
	"bytes"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/marapongo/mu/pkg/compiler/errors"
	"github.com/marapongo/mu/pkg/diag/colors"
	"github.com/marapongo/mu/pkg/resource"
	"github.com/marapongo/mu/pkg/util/contract"
)

func newApplyCmd() *cobra.Command {
	var delete bool
	var detailed bool
	var cmd = &cobra.Command{
		Use:   "apply [blueprint] [-- [args]]",
		Short: "Apply a deployment plan from a Mu blueprint",
		Long: "Apply a deployment plan from a Mu blueprint.\n" +
			"\n" +
			"This command performs creation, update, and delete operations against a target\n" +
			"environment, in order to carry out the activities laid out in a deployment plan.\n" +
			"\n" +
			"These activities are the result of compiling a MuPackage blueprint into a MuGL\n" +
			"graph, and comparing that graph to another one representing the current known\n" +
			"state of the target environment.  After the completion of this command, the target\n" +
			"environment's state will have been advanced so that it matches the input blueprint." +
			"\n" +
			"By default, a blueprint package is loaded from the current directory.  Optionally,\n" +
			"a path to a blueprint elsewhere can be provided as the [blueprint] argument.",
		Run: func(cmd *cobra.Command, args []string) {
			if comp, plan := plan(cmd, args, delete); plan != nil {
				progress := newProgress(detailed)
				if err, _, _ := plan.Apply(progress); err != nil {
					// TODO: we want richer diagnostics in the event that a plan apply fails.  For instance, we want to
					//     know precisely what step failed, we want to know whether it was catastrophic, etc.  We also
					//     probably want to plumb diag.Sink through apply so it can issue its own rich diagnostics.
					comp.Diag().Errorf(errors.ErrorPlanApplyFailed, err)
				}
			}
		},
	}

	// TODO: options; most importantly, what to compare the blueprint against.
	cmd.PersistentFlags().BoolVar(
		&delete, "delete", false,
		"Delete the entirety of the blueprint's resources")
	cmd.PersistentFlags().BoolVar(
		&detailed, "detailed", false,
		"Display detailed output during the application of changes")

	return cmd
}

// applyProgress pretty-prints the plan application process as it goes.
type applyProgress struct {
	c        int
	detailed bool
}

func newProgress(detailed bool) *applyProgress {
	return &applyProgress{detailed: detailed}
}

func (prog *applyProgress) Before(step resource.Step) {
	var b bytes.Buffer
	prog.c++
	b.WriteString(fmt.Sprintf("Applying step #%v\n", prog.c))
	printStep(&b, step, !prog.detailed, "    ")
	s := colors.Colorize(b.String())
	fmt.Printf(s)
}

func (prog *applyProgress) After(step resource.Step, err error, state resource.ResourceState) {
	if err != nil {
		var b bytes.Buffer
		// Print the state of the resource; we don't issue the error, because the apply above will do that.
		b.WriteString(fmt.Sprintf("Step #%v failed: ", prog.c))
		switch state {
		case resource.StateOK:
			b.WriteString(colors.BrightYellow)
			b.WriteString("provider successfully recovered from this failure")
		case resource.StateUnknown:
			b.WriteString(colors.BrightRed)
			b.WriteString("this failure was catastrophic and the provider cannot guarantee recovery")
		default:
			contract.Failf("Unrecognized resource state: %v", state)
		}
		b.WriteString(colors.Reset)
		b.WriteString("\n")
		s := colors.Colorize(b.String())
		fmt.Printf(s)
	}
}
