package generate

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/structs"
)

var (
	providerType   = reflect.TypeOf((*structs.Provider)(nil)).Elem()
	readerType     = reflect.TypeOf((*io.Reader)(nil)).Elem()
	readWriterType = reflect.TypeOf((*io.ReadWriter)(nil)).Elem()
	writerType     = reflect.TypeOf((*io.Writer)(nil)).Elem()
)

var (
	rePathVars = regexp.MustCompile(`{([a-z]+)[^}]*}`)
)

type Arg struct {
	Name string
	Type reflect.Type
}

type Method struct {
	Name    string
	Route   Route
	Args    []Arg
	Returns []reflect.Type
}

type Route struct {
	Method string
	Path   string
}

func Methods() ([]Method, error) {
	ms := []Method{}

	data, err := ioutil.ReadFile("pkg/structs/provider.go")
	if err != nil {
		return nil, err
	}

	names := []string{}

	for name := range routes {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		route := routes[name]
		routeParts := strings.SplitN(route, " ", 2)

		args, returns, err := signature(data, name)
		if err != nil {
			return nil, err
		}

		method := ""
		path := ""

		if len(routeParts) == 2 {
			method = routeParts[0]
			path = routeParts[1]
		}

		pvm := rePathVars.FindAllStringSubmatch(path, -1)

		for _, pv := range pvm {
			for _, v := range pv[1:] {
				found := false
				for _, a := range args {
					if a.Name == v {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("path variable not found in args for %s: %s", name, v)
				}
			}
		}

		m := Method{
			Name: name,
			Route: Route{
				Method: method,
				Path:   path,
			},
			Args:    args,
			Returns: returns,
		}

		ms = append(ms, m)
	}

	return ms, nil
}

func (m *Method) Ints() []Arg {
	as := []Arg{}

	for _, a := range m.Args {
		if a.Int() {
			as = append(as, a)
		}
	}

	return as
}

func (m *Method) Option() *Arg {
	for _, a := range m.Args {
		if a.Option() {
			return &a
		}
	}
	return nil
}

func (m *Method) Reader() bool {
	for _, r := range m.Returns {
		if r.Implements(readerType) {
			return true
		}
	}
	return false
}

func (m *Method) ReturnsValue() bool {
	if len(m.Returns) == 2 {
		return true
	}
	return false
}

func (m *Method) Writer() string {
	for _, a := range m.Args {
		if a.Type.Implements(writerType) {
			return a.Name
		}
	}

	return ""
}

func (m *Method) Socket() bool {
	return m.Route.Method == "SOCKET"
}

func (m *Method) SocketExit() (bool, error) {
	rt, err := m.ReturnType()
	if err != nil {
		return false, err
	}

	return (m.Socket() && rt != nil && rt.Kind() == reflect.Int), nil
}

func (m *Method) ReturnType() (reflect.Type, error) {
	switch len(m.Returns) {
	case 1:
		return nil, nil
	case 2:
		return m.Returns[0], nil
	default:
		return nil, fmt.Errorf("dont know how to handle %d return values", len(m.Returns))
	}
}

func (a *Arg) Int() bool {
	return a.Type.Kind() == reflect.Int
}

func (a *Arg) Option() bool {
	return a.Type.Kind() == reflect.Struct
}

func (a *Arg) Path(m Method) bool {
	return regexp.MustCompile(fmt.Sprintf("{%s(:.*)?}", a.Name)).MatchString(m.Route.Path)
}

func (a *Arg) Slice() bool {
	return a.Type.Kind() == reflect.Slice
}

func (a *Arg) Stream() bool {
	return a.Type.Implements(readerType) || a.Type.Implements(writerType)
}

func signature(data []byte, name string) ([]Arg, []reflect.Type, error) {
	m, ok := providerType.MethodByName(name)
	if !ok {
		return nil, nil, fmt.Errorf("no provider method: %s", name)
	}

	// fmt.Printf("m = %+v\n", m)

	r, err := regexp.Compile(fmt.Sprintf(`%s\(([^)]*)\)`, name))
	if err != nil {
		return nil, nil, err
	}

	args := []Arg{}
	returns := []reflect.Type{}

	match := r.FindStringSubmatch(string(data))

	if len(match[1]) > 0 {
		parts := strings.Split(match[1], ",")

		for i, p := range parts {
			args = append(args, Arg{
				Name: strings.Split(strings.TrimSpace(p), " ")[0],
				Type: m.Type.In(i),
			})
		}
	}

	for i := 0; i < m.Type.NumOut(); i++ {
		returns = append(returns, m.Type.Out(i))
	}

	// fmt.Printf("args = %+v\n", args)
	// fmt.Printf("returns = %+v\n", returns)

	return args, returns, nil
}
