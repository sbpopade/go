// autogenerated: do not edit!
// generated from gentemplate [gentemplate -id HwIfAddDelHook -d Package=vnet -d DepsType=HwIfAddDelHookVec -d Type=HwIfAddDelHook -d Data=hooks github.com/platinasystems/go/elib/dep/dep.tmpl]

// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vnet

import (
	"github.com/platinasystems/go/elib/dep"
)

type HwIfAddDelHookVec struct {
	deps  dep.Deps
	hooks []HwIfAddDelHook
}

func (t *HwIfAddDelHookVec) Len() int {
	return t.deps.Len()
}

func (t *HwIfAddDelHookVec) Get(i int) HwIfAddDelHook {
	return t.hooks[t.deps.Index(i)]
}

func (t *HwIfAddDelHookVec) Add(x HwIfAddDelHook, ds ...*dep.Dep) {
	if len(ds) == 0 {
		t.deps.Add(&dep.Dep{})
	} else {
		t.deps.Add(ds[0])
	}
	t.hooks = append(t.hooks, x)
}
