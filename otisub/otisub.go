package otisub

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
