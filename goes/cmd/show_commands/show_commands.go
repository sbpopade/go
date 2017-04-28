// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package show_commands

import (
	"fmt"
	"sort"

	"github.com/platinasystems/go/goes"
	"github.com/platinasystems/go/goes/lang"
)

const (
	Name    = "show-commands"
	Apropos = "list all commands and daemons"
	Usage   = "show-commands"
)

type Interface interface {
	Apropos() lang.Alt
	ByName(goes.ByName)
	Kind() goes.Kind
	Main(...string) error
	String() string
	Usage() string
}

func New() Interface { return new(cmd) }

type cmd goes.ByName

func (*cmd) Apropos() lang.Alt { return apropos }

func (c *cmd) ByName(byName goes.ByName) { *c = cmd(byName) }

func (*cmd) Kind() goes.Kind { return goes.DontFork }

func (c *cmd) Main(args ...string) error {
	if len(args) > 0 {
		return fmt.Errorf("%v: unexpected", args)
	}

	keys := make([]string, 0, len(goes.ByName(*c)))
	for k := range goes.ByName(*c) {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g := goes.ByName(*c)[k]
		if g.Kind.IsDaemon() {
			fmt.Printf("\t%s - daemon\n", k)
		} else if g.Kind.IsHidden() {
			fmt.Printf("\t%s - hidden\n", k)
		} else {
			fmt.Printf("\t%s\n", k)
		}
	}
	return nil
}

func (*cmd) String() string { return Name }
func (*cmd) Usage() string  { return Usage }

var apropos = lang.Alt{
	lang.EnUS: Apropos,
}