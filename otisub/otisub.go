package otisub

import (
	"sort"
)

type Interface interface {
	Name() string
	Main(args []string)
}

type simplesub struct {
	name string
	run  func([]string)
}

func (c simplesub) Name() string       { return c.name }
func (c simplesub) Main(args []string) { c.run(args) }

var subreg = func() chan map[string]Interface {
	c := make(chan map[string]Interface, 1)
	c <- make(map[string]Interface)
	return c
}()

func regsub(subname string, step Interface) {
	m := <-subreg
	defer func() { subreg <- m }()
	if m[subname] != nil {
		panic("already registered")
	}
	m[subname] = step
}

type isort []Interface

func (is isort) Len() int           { return len(is) }
func (is isort) Less(i, j int) bool { return is[i].Name() < is[j].Name() }
func (is isort) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }

func GetAll() []Interface {
	m := <-subreg
	defer func() { subreg <- m }()
	subs := make([]Interface, 0, len(m))
	for _, s := range m {
		subs = append(subs, s)
	}
	sort.Sort(isort(subs))
	return subs
}

func Get(subname string) Interface {
	m := <-subreg
	defer func() { subreg <- m }()
	return m[subname]
}

func Register(name string, main func(args []string)) string {
	return RegisterSub(simplesub{name, main})
}

func RegisterSub(sub Interface) string {
	if sub == nil {
		panic("nil step")
	}
	name := sub.Name()
	regsub(name, sub)
	return name
}
