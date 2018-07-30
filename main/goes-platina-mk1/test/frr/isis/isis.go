// Copyright © 2015-2017 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package isis

import (
	"testing"
	"time"

	"github.com/platinasystems/go/internal/test"
	"github.com/platinasystems/go/internal/test/docker"
)

type isis struct {
	docker.Docket
}

var Suite = test.Suite{
	Name: "isis",
	Tests: test.Tests{
		&isis{
			docker.Docket{
				Name: "eth",
				Tmpl: "testdata/frr/isis/conf.yaml.tmpl",
			},
		},
		&isis{
			docker.Docket{
				Name: "vlan",
				Tmpl: "testdata/frr/isis/vlan/conf.yaml.tmpl",
			},
		},
	},
}

func (isis *isis) Test(t *testing.T) {
	isis.Docket.Tests = test.Tests{
		&test.Unit{"", isis.checkConnectivity},
		&test.Unit{"", isis.checkFrr},
		&test.Unit{"", isis.addIntfConf},
		&test.Unit{"", isis.checkNeighbors},
		&test.Unit{"", isis.checkRoutes},
		&test.Unit{"", isis.checkInterConnectivity},
		&test.Unit{"", isis.checkFlap},
	}
	isis.Docket.Test(t)
}

func (isis *isis) checkConnectivity(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		host   string
		target string
	}{
		{"R1", "192.168.120.10"},
		{"R1", "192.168.150.4"},
		{"R2", "192.168.222.2"},
		{"R2", "192.168.120.5"},
		{"R3", "192.168.222.10"},
		{"R3", "192.168.111.4"},
		{"R4", "192.168.111.2"},
		{"R4", "192.168.150.5"},
	} {
		out, err := isis.ExecCmd(t, x.host, "ping", "-c3", x.target)
		assert.Nil(err)
		assert.Match(out, "[1-3] packets received")
	}
}

func (isis *isis) checkFrr(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)

	for _, r := range isis.Routers {
		t.Logf("Checking FRR on %v", r.Hostname)
		out, err := isis.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.Match(out, ".*isisd.*")
		assert.Match(out, ".*zebra.*")
	}
}

func (isis *isis) addIntfConf(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range isis.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := isis.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t",
				"-c", "interface "+intf,
				"-c", "ip router isis "+r.Hostname)
			assert.Nil(err)
		}
	}
}

func (isis *isis) checkNeighbors(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		peer     string
		address  string
	}{
		{"R1", "R2", "192.168.120.10"},
		{"R1", "R4", "192.168.150.4"},
		{"R2", "R1", "192.168.120.5"},
		{"R2", "R3", "192.168.222.2"},
		{"R3", "R2", "192.168.222.10"},
		{"R3", "R4", "192.168.111.4"},
		{"R4", "R3", "192.168.111.2"},
		{"R4", "R1", "192.168.150.5"},
	} {
		timeout := 60
		found := false
		for i := timeout; i > 0; i-- {
			out, err := isis.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show isis neighbor "+x.peer)
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.address) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No isis neighbor for %v: %v",
				x.hostname, x.peer)
		}
	}
}

func (isis *isis) checkRoutes(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"R1", "192.168.222.0/24"},
		{"R1", "192.168.111.0/24"},
		{"R2", "192.168.150.0/24"},
		{"R2", "192.168.111.0/24"},
		{"R3", "192.168.120.0/24"},
		{"R3", "192.168.150.0/24"},
		{"R4", "192.168.120.0/24"},
		{"R4", "192.168.222.0/24"},
	} {
		found := false
		timeout := 60
		for i := timeout; i > 0; i-- {
			out, err := isis.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show ip route isis")
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.route) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No isis route for %v: %v", x.hostname, x.route)
		}
	}
}

func (isis *isis) checkInterConnectivity(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"R1", "192.168.222.2"},
		{"R1", "192.168.111.2"},
		{"R2", "192.168.111.4"},
		{"R2", "192.168.150.4"},
		{"R3", "192.168.120.5"},
		{"R3", "192.168.150.5"},
		{"R4", "192.168.120.10"},
		{"R4", "192.168.222.10"},
	} {
		out, err := isis.ExecCmd(t, x.hostname, "ping", "-c3", x.target)
		assert.Nil(err)
		assert.Match(out, "[1-3] packets received")
		assert.Program(test.Self{}, "vnet", "show", "ip", "fib")
	}
}

func (isis *isis) checkFlap(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range isis.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := isis.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = isis.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(test.Self{}, "vnet", "show", "ip", "fib")
		}
	}
}
