package cni

import (
	"context"

	cnilibrary "demo/others/cni/libcni"
	types100 "demo/others/cni/pkg/types/100"
)

type Network struct {
	cni    cnilibrary.CNI
	config *cnilibrary.NetworkConfigList
	ifName string
}

func (n *Network) Attach(ctx context.Context, ns *Namespace) (*types100.Result, error) {
	r, err := n.cni.AddNetworkList(ctx, n.config, ns.config(n.ifName))
	if err != nil {
		return nil, err
	}
	return types100.NewResultFromResult(r)
}

func (n *Network) Remove(ctx context.Context, ns *Namespace) error {
	return n.cni.DelNetworkList(ctx, n.config, ns.config(n.ifName))
}

func (n *Network) Check(ctx context.Context, ns *Namespace) error {
	return n.cni.CheckNetworkList(ctx, n.config, ns.config(n.ifName))
}

type Namespace struct {
	id             string // sanbox id
	path           string
	capabilityArgs map[string]interface{}
	args           map[string]string
}

func (ns *Namespace) config(ifName string) *cnilibrary.RuntimeConf {
	c := &cnilibrary.RuntimeConf{
		ContainerID: ns.id, //
		NetNS:       ns.path,
		IfName:      ifName,
	}
	for k, v := range ns.args {
		c.Args = append(c.Args, [2]string{k, v})
	}
	c.CapabilityArgs = ns.capabilityArgs
	return c
}

func newNamespace(id, path string, opts ...NamespaceOpts) (*Namespace, error) {
	ns := &Namespace{
		id:             id,
		path:           path,
		capabilityArgs: make(map[string]interface{}),
		args:           make(map[string]string),
	}
	for _, o := range opts {
		if err := o(ns); err != nil {
			return nil, err
		}
	}
	return ns, nil
}
