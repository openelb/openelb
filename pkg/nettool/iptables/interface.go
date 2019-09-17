package iptables

import (
	"fmt"

	coreosiptables "github.com/coreos/go-iptables/iptables"
)

// IptablesIface wrapper package coreos/iptables
type IptablesIface interface {
	Exists(table, chain string, rulespec ...string) (bool, error)
	Insert(table, chain string, pos int, rulespec ...string) error
	Append(table, chain string, rulespec ...string) error
	Delete(table, chain string, rulespec ...string) error
	List(table, chain string) ([]string, error)
	NewChain(table, chain string) error
	ClearChain(table, chain string) error
	DeleteChain(table, chain string) error
	ListChains(table string) ([]string, error)
	HasRandomFully() bool
}

type IptablesRule struct {
	Name         string
	ShouldExist  bool
	Table, Chain string
	Rule         []string
}

func (r IptablesRule) String() string {
	return fmt.Sprintf("%s/%s rule %s", r.Table, r.Chain, r.Name)
}

func NewIPTables() IptablesIface {
	ipt, err := coreosiptables.New()
	if err != nil {
		panic(err)
	}
	return ipt
}
