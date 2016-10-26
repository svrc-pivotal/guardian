package bundlerules

import (
	"code.cloudfoundry.org/guardian/gardener"
	"code.cloudfoundry.org/guardian/rundmc/goci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type Limits struct {
}

func (l Limits) Apply(bndl goci.Bndl, spec gardener.DesiredContainerSpec) goci.Bndl {
	memlimit := uint64(spec.Limits.Memory.LimitInBytes)
	bndl = bndl.WithMemoryLimit(specs.Memory{Limit: &memlimit, Swap: &memlimit})

	shares := uint64(spec.Limits.CPU.LimitInShares)
	bndl = bndl.WithCPUShares(specs.CPU{Shares: &shares})

	pidlimit := int64(spec.Limits.PID.Limit)
	bndl = bndl.WithPIDLimit(specs.Pids{Limit: &pidlimit})

	return bndl
}
