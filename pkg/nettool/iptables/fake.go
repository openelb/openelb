package iptables

type FakeIPTables struct {
	EnableRandomFully bool
	Data              map[string]map[string][]IptablesRule
}

func ruleEqual(rule IptablesRule, rulespec ...string) bool {
	if len(rule.Rule) == len(rulespec) {
		for index := 0; index < len(rulespec); index++ {
			if rule.Rule[index] != rulespec[index] {
				return false
			}
		}
		return true
	}
	return false
}

func NewFakeIPTables() *FakeIPTables {
	result := &FakeIPTables{
		EnableRandomFully: true,
		Data:              make(map[string]map[string][]IptablesRule),
	}
	//add used tables

	result.Data["nat"] = make(map[string][]IptablesRule)
	result.Data["filter"] = make(map[string][]IptablesRule)
	result.Data["mangle"] = make(map[string][]IptablesRule)

	return result
}

func (f *FakeIPTables) Exists(table, chain string, rulespec ...string) (bool, error) {
	Rules := f.Data[table][chain]
	for _, rule := range Rules {
		if ruleEqual(rule, rulespec...) {
			return true, nil
		}
	}
	return false, nil
}

func (f *FakeIPTables) Insert(table, chain string, pos int, rulespec ...string) error {
	if _, ok := f.Data[table][chain]; !ok {
		f.Data[table][chain] = make([]IptablesRule, 0)
	}
	newSlice := []IptablesRule{IptablesRule{
		Table: table,
		Chain: chain,
		Rule:  rulespec,
	}}
	f.Data[table][chain] = append(newSlice, f.Data[table][chain]...)
	return nil
}

func (f *FakeIPTables) Append(table, chain string, rulespec ...string) error {
	if _, ok := f.Data[table][chain]; !ok {
		f.Data[table][chain] = make([]IptablesRule, 0)
	}
	f.Data[table][chain] = append(f.Data[table][chain], IptablesRule{
		Table: table,
		Chain: chain,
		Rule:  rulespec,
	})
	return nil
}

func (f *FakeIPTables) Delete(table, chain string, rulespec ...string) error {
	rules := f.Data[table][chain]
	for index, rule := range rules {
		if ruleEqual(rule, rulespec...) {
			if index < len(rules)-1 {
				f.Data[table][chain] = append(rules[0:index], rules[index+1:]...)
			} else {
				f.Data[table][chain] = rules[0:index]
			}
			break
		}
	}
	return nil
}

func (f *FakeIPTables) List(table, chain string) ([]string, error) {
	rules := f.Data[table][chain]
	result := make([]string, len(rules))
	for index := 0; index < len(result); index++ {
		result[index] = rules[index].String()
	}
	return result, nil
}

func (f *FakeIPTables) NewChain(table, chain string) error {
	if _, ok := f.Data[table][chain]; !ok {
		f.Data[table][chain] = make([]IptablesRule, 0)
	}
	return nil
}

func (f *FakeIPTables) ClearChain(table, chain string) error {
	f.Data[table][chain] = make([]IptablesRule, 0)
	return nil
}

func (f *FakeIPTables) DeleteChain(table, chain string) error {
	delete(f.Data[table], chain)
	return nil
}

func (f *FakeIPTables) ListChains(table string) ([]string, error) {
	result := make([]string, 0)
	for key := range f.Data[table] {
		result = append(result, key)
	}
	return result, nil
}

func (f *FakeIPTables) HasRandomFully() bool {
	return f.EnableRandomFully
}
