package main

import (
	"fmt"
	"log"
)

type evtype int

const (
	evtypePush = iota
)

type acttype int

const (
	acttypeDeploy = iota
)

type checkAction func(env *environment, e *event) bool

type actCheck struct {
	acttype    acttype
	makeAction func(opts dict) action
	checks     []checkAction
}

var eventActions map[evtype][]*actCheck

func init() {
	// statically init event-to-actions map.
	eventActions = make(map[evtype][]*actCheck)

	eventActions[evtypePush] = []*actCheck{
		&actCheck{
			acttype:    acttypeDeploy,
			makeAction: makeActionDeploy,
			checks: []checkAction{
				checkHasDeployCommand,
				checkSameBranch,
			},
		},
	}
}

type action interface {
	run() (result, error)
}

type result interface {
	toJSON() []byte
}

type exec struct {
	cmd  string
	args []string
}

func newExec(cmd string, args ...string) *exec {
	return &exec{cmd: cmd, args: args}
}

func (e *exec) run() (result, error) {
	// TODO: Exec command
	var out, errs []byte
	return newExecResult(out, errs), nil
}

type execResult struct {
	output []byte
	errors []byte
}

func newExecResult(out, errs []byte) *execResult {
	return &execResult{
		output: out,
		errors: errs,
	}
}

func (er *execResult) toJSON() []byte {
	return []byte("") // TODO
}

func makeActionDeploy(opts dict) action {
	// TODO: In opts there is the name of the environment etc
	return newExec("ls", "-lh")
}

func checkHasDeployCommand(env *environment, e *event) bool {
	// TODO
	return true
}

func checkSameBranch(env *environment, e *event) bool {
	evBranch := e.data.getVal("git-branch", "")
	envBranch := env.data.getVal("git-branch", "")
	if evBranch != "" && evBranch == envBranch {
		return true
	}
	return false
}

type dict map[string]string

func makeDict() dict {
	return make(map[string]string)
}

func (d dict) get(k string) (string, bool) {
	v, ok := d[k]
	return v, ok
}

func (d dict) set(k, v string) {
	d[k] = v
}

func (d dict) getVal(k, defv string) string {
	if v, ok := d[k]; ok {
		return v
	}
	return defv
}

type environment struct {
	name   string
	data   dict
	events chan *event
}

func newEnvironment(name string) *environment {
	e := &environment{
		name:   name,
		data:   makeDict(),
		events: make(chan *event/*, 10*/),
	}
	go e.run()
	return e
}

func (e *environment) run() {
	for ev := range e.events {
		acs, ok := eventActions[ev.typ]
		if !ok {
			log.Printf("error: unknown event type %d", ev.typ)
			continue
		}
		for _, ac := range acs {
			ok := true
			for _, check := range ac.checks {
				if !check(e, ev) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			// Send job to workers
			opts := makeDict() // TODO: Add options like env name etc
			// TODO: In worker
			action := ac.makeAction(opts)
			res, err := action.run()
			if err != nil {
				log.Print("error: running action %s: %s", action, err)
				continue
			}
			_ = res // TODO: Persist res
		}
	}
}

type event struct {
	typ  evtype
	data dict
}

func newEvent(typ evtype) *event {
	return &event{
		typ:  typ,
		data: makeDict(),
	}
}

func main() {
	env := newEnvironment("test.gi")
	fmt.Printf("%v\n", env)
}
