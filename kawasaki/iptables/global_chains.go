package iptables

import (
	"fmt"
	"net"
	"os"
	"os/exec"
)

const SetupScript = `
	set -o xtrace
	set -o nounset
	set -o errexit
	shopt -s nullglob

	filter_input_chain="${GARDEN_IPTABLES_FILTER_INPUT_CHAIN}"
	filter_forward_chain="${GARDEN_IPTABLES_FILTER_FORWARD_CHAIN}"
	filter_default_chain="${GARDEN_IPTABLES_FILTER_DEFAULT_CHAIN}"
	filter_instance_prefix="${GARDEN_IPTABLES_FILTER_INSTANCE_PREFIX}"
	nat_prerouting_chain="${GARDEN_IPTABLES_NAT_PREROUTING_CHAIN}"
	nat_postrouting_chain="${GARDEN_IPTABLES_NAT_POSTROUTING_CHAIN}"
	nat_instance_prefix="${GARDEN_IPTABLES_NAT_INSTANCE_PREFIX}"
	iptables_bin="${GARDEN_IPTABLES_BIN}"

	function teardown_deprecated_rules() {
		# Remove jump to garden-dispatch from INPUT
		$iptables_bin -w -S INPUT 2> /dev/null |
		grep " -j garden-dispatch" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w

		# Remove jump to garden-dispatch from FORWARD
		$iptables_bin -w -S FORWARD 2> /dev/null |
		grep " -j garden-dispatch" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w

		# Prune garden-dispatch
		$iptables_bin -w -F garden-dispatch 2> /dev/null || true

		# Delete garden-dispatch
		$iptables_bin -w -X garden-dispatch 2> /dev/null || true
	}

	function teardown_filter() {
		teardown_deprecated_rules

		# Prune garden-forward chain
		$iptables_bin -w -S ${filter_forward_chain} 2> /dev/null |
		grep "\-g ${filter_instance_prefix}" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w

		# Prune per-instance chains
		$iptables_bin -w -S 2> /dev/null |
		grep "^-A ${filter_instance_prefix}" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w

		# Delete per-instance chains
		$iptables_bin -w -S 2> /dev/null |
		grep "^-N ${filter_instance_prefix}" |
		sed -e "s/-N/-X/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w || true

		# Remove jump to garden-forward from FORWARD
		$iptables_bin -w -S FORWARD 2> /dev/null |
		grep " -j ${filter_forward_chain}" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w || true

		$iptables_bin -w -F ${filter_forward_chain} 2> /dev/null || true
		$iptables_bin -w -F ${filter_default_chain} 2> /dev/null || true

		# Remove jump to filter input chain from INPUT
		$iptables_bin -w -S INPUT 2> /dev/null |
		grep " -j ${filter_input_chain}" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w || true

		# Empty and delete filter input chain
		$iptables_bin -w -F ${filter_input_chain} 2> /dev/null || true
		$iptables_bin -w -X ${filter_input_chain} 2> /dev/null || true
	}

	function setup_filter() {
		teardown_filter

		# Determine interface device to the outside
		default_interface=$(ip route show | grep default | cut -d' ' -f5 | head -1)

		# Create, or empty existing, filter input chain
		$iptables_bin -w -N ${filter_input_chain} 2> /dev/null || $iptables_bin -w -F ${filter_input_chain}

		# Accept inbound packets if default interface is matched by filter prefix
		$iptables_bin -w -I ${filter_input_chain} -i $default_interface --jump ACCEPT

		# Put connection tracking rule in filter input chain
		# to accept packets related to previously established connections
		$iptables_bin -w -A ${filter_input_chain} -m conntrack --ctstate ESTABLISHED,RELATED --jump ACCEPT

		if [ "${GARDEN_IPTABLES_ALLOW_HOST_ACCESS}" != "true" ]; then
		$iptables_bin -w -A ${filter_input_chain} --jump REJECT --reject-with icmp-host-prohibited
		else
		$iptables_bin -w -A ${filter_input_chain} --jump ACCEPT
		fi

		# Forward input traffic via ${filter_input_chain}
		$iptables_bin -w -A INPUT -i ${GARDEN_NETWORK_INTERFACE_PREFIX}+ --jump ${filter_input_chain}

		# Create or flush forward chain
		$iptables_bin -w -N ${filter_forward_chain} 2> /dev/null || $iptables_bin -w -F ${filter_forward_chain}
		$iptables_bin -w -A ${filter_forward_chain} -j DROP

		# Create or flush default chain
		$iptables_bin -w -N ${filter_default_chain} 2> /dev/null || $iptables_bin -w -F ${filter_default_chain}

		# Always allow established connections to containers
		$iptables_bin -w -A ${filter_default_chain} -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

		# Forward outbound traffic via ${filter_forward_chain}
		$iptables_bin -w -A FORWARD -i ${GARDEN_NETWORK_INTERFACE_PREFIX}+ --jump ${filter_forward_chain}

		# Forward inbound traffic immediately
		$iptables_bin -w -I ${filter_forward_chain} -i $default_interface --jump ACCEPT
	}

	function teardown_nat() {
		# Prune prerouting chain
		$iptables_bin -w -t nat -S ${nat_prerouting_chain} 2> /dev/null |
		grep "\-j ${nat_instance_prefix}" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w -t nat

		# Prune per-instance chains
		$iptables_bin -w -t nat -S 2> /dev/null |
		grep "^-A ${nat_instance_prefix}" |
		sed -e "s/-A/-D/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w -t nat

		# Delete per-instance chains
		$iptables_bin -w -t nat -S 2> /dev/null |
		grep "^-N ${nat_instance_prefix}" |
		sed -e "s/-N/-X/" -e "s/\s\+\$//" |
		xargs --no-run-if-empty --max-lines=1 $iptables_bin -w -t nat || true

		# Flush prerouting chain
		$iptables_bin -w -t nat -F ${nat_prerouting_chain} 2> /dev/null || true

		# Flush postrouting chain
		$iptables_bin -w -t nat -F ${nat_postrouting_chain} 2> /dev/null || true
	}

	function setup_nat() {
		teardown_nat

		# Create prerouting chain
		$iptables_bin -w -t nat -N ${nat_prerouting_chain} 2> /dev/null || true

		# Bind chain to PREROUTING
		($iptables_bin -w -t nat -S PREROUTING | grep -q "\-j ${nat_prerouting_chain}\b") ||
		$iptables_bin -w -t nat -A PREROUTING \
		--jump ${nat_prerouting_chain}

		# Bind chain to OUTPUT (for traffic originating from same host)
		($iptables_bin -w -t nat -S OUTPUT | grep -q "\-j ${nat_prerouting_chain}\b") ||
		$iptables_bin -w -t nat -A OUTPUT \
		--out-interface "lo" \
		--jump ${nat_prerouting_chain}

		# Create postrouting chain
		$iptables_bin -w -t nat -N ${nat_postrouting_chain} 2> /dev/null || true

		# Bind chain to POSTROUTING
		($iptables_bin -w -t nat -S POSTROUTING | grep -q "\-j ${nat_postrouting_chain}\b") ||
		$iptables_bin -w -t nat -A POSTROUTING \
		--jump ${nat_postrouting_chain}
	}

	case "${ACTION}" in
	setup)
	setup_filter
	setup_nat

	# Enable forwarding
	echo 1 > /proc/sys/net/ipv4/ip_forward
	;;
	teardown)
	teardown_filter
	teardown_nat
	;;
	*)
	echo "Unknown command: ${1}" 1>&2
	exit 1
	;;
	esac
`

type Starter struct {
	iptablesBin     string
	config          IPTablesConfig
	driver          IPTablesDriver
	runner          IPTablesRunner
	allowHostAccess bool
	nicPrefix       string

	denyNetworks []string
}

func NewStarter(iptablesBin string, config IPTablesConfig, driver IPTablesDriver, runner IPTablesRunner, allowHostAccess bool, nicPrefix string, denyNetworks []string) *Starter {
	return &Starter{
		iptablesBin:     iptablesBin,
		config:          config,
		driver:          driver,
		runner:          runner,
		allowHostAccess: allowHostAccess,
		nicPrefix:       nicPrefix,

		denyNetworks: denyNetworks,
	}
}

func (s Starter) Start() error {

	if !s.driver.ChainExists("filter", s.config.InputChain) {

		cmd := exec.Command("bash", "-c", SetupScript)
		cmd.Env = []string{
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			"ACTION=setup",
			fmt.Sprintf("GARDEN_IPTABLES_BIN=%s", s.iptablesBin),
			fmt.Sprintf("GARDEN_IPTABLES_FILTER_INPUT_CHAIN=%s", s.config.InputChain),
			fmt.Sprintf("GARDEN_IPTABLES_FILTER_FORWARD_CHAIN=%s", s.config.ForwardChain),
			fmt.Sprintf("GARDEN_IPTABLES_FILTER_DEFAULT_CHAIN=%s", s.config.DefaultChain),
			fmt.Sprintf("GARDEN_IPTABLES_FILTER_INSTANCE_PREFIX=%s", s.config.InstanceChainPrefix),
			fmt.Sprintf("GARDEN_IPTABLES_NAT_PREROUTING_CHAIN=%s", s.config.PreroutingChain),
			fmt.Sprintf("GARDEN_IPTABLES_NAT_POSTROUTING_CHAIN=%s", s.config.PostroutingChain),
			fmt.Sprintf("GARDEN_IPTABLES_NAT_INSTANCE_PREFIX=%s", s.config.InstanceChainPrefix),
			fmt.Sprintf("GARDEN_NETWORK_INTERFACE_PREFIX=%s", s.nicPrefix),
			fmt.Sprintf("GARDEN_IPTABLES_ALLOW_HOST_ACCESS=%t", s.allowHostAccess),
		}

		if err := s.runner.Run("setup-global-chains", cmd); err != nil {
			return fmt.Errorf("setting up global chains: %s", err)
		}
	}

	if err := s.resetDenyNetworks(); err != nil {
		return fmt.Errorf("resetting default chain: %s", err)
	}

	return nil
}

func (s Starter) resetDenyNetworks() error {
	establishedConnsRule := IPTablesFlags{
		"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "--jump", "ACCEPT",
	}

	rules := []Rule{establishedConnsRule}
	for _, cidr := range s.denyNetworks {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		rules = append(rules, RejectRule{
			DestinationIPNet: network,
		})
	}

	return s.driver.ResetChain("filter", s.config.DefaultChain, rules)
}
