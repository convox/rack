package generate

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

func templateHelpers() template.FuncMap {
	return template.FuncMap{
		"args": func(m Method) string {
			as := []string{}
			for _, a := range m.Args {
				as = append(as, a.Name)
			}
			return strings.Join(as, ", ")
		},
		"args_types": func(m Method) string {
			as := []string{}
			for _, a := range m.Args {
				as = append(as, fmt.Sprintf("%s %s", a.Name, a.Type))
			}
			return strings.Join(as, ", ")
		},
		"method": func(method string) (string, error) {
			switch method {
			case "DELETE":
				return "Delete", nil
			case "GET":
				return "Get", nil
			case "HEAD":
				return "Head", nil
			case "OPTIONS":
				return "Options", nil
			case "POST":
				return "Post", nil
			case "PUT":
				return "Put", nil
			default:
				return "", fmt.Errorf("unknown method: %s", method)
			}
		},
		"params": func(m Method) (string, error) {
			ps := []string{}
			vt := "Params"
			if m.Socket() {
				vt = "Headers"
			}
			for _, a := range m.Args {
				if !a.Path(m) {
					switch a.Type.Kind() {
					case reflect.Int:
						ps = append(ps, fmt.Sprintf(`ro.%s["%s"] = fmt.Sprintf("%%d", %s)`, vt, a.Name, a.Name))
					case reflect.Slice:
						ps = append(ps, fmt.Sprintf(`ro.%s["%s"] = strings.Join(%s, ",")`, vt, a.Name, a.Name))
					case reflect.String:
						ps = append(ps, fmt.Sprintf(`ro.%s["%s"] = %s`, vt, a.Name, a.Name))
					case reflect.Interface, reflect.Struct:
						if a.Type.Implements(readerType) {
							ps = append(ps, fmt.Sprintf("ro.Body = %s", a.Name))
						}
					default:
						return "", fmt.Errorf("unknown param type: %s", a.Type.Kind())
					}
				}
			}
			return strings.Join(ps, "\n"), nil
		},
		"path": func(m Method) string {
			fs := m.Route.Path
			fa := []string{}
			for _, a := range m.Args {
				if a.Path(m) {
					ft := "%s"
					switch a.Type.Kind() {
					case reflect.Int:
						ft = "%d"
					}
					fs = regexp.MustCompile(fmt.Sprintf("{%s(:.*)?}", a.Name)).ReplaceAllString(fs, ft)
					// fs = strings.Replace(fs, fmt.Sprintf("{%s}", a.Name), ft, -1)
					fa = append(fa, a.Name)
				}
			}
			return fmt.Sprintf(`fmt.Sprintf("%s", %s)`, fs, strings.Join(fa, ", "))
		},
		"render": func(m Method) (string, error) {
			switch len(m.Returns) {
			case 1:
				if m.Writer() != "" {
					return "nil", nil
				} else {
					return "c.RenderOK()", nil
				}
			case 2:
				switch m.Returns[0].Kind() {
				case reflect.Bool:
					return "c.RenderJSON(v)", nil
				case reflect.Interface:
					return "nil", nil
				case reflect.Int:
					return "renderStatusCode(c, v)", nil
				case reflect.Ptr, reflect.Slice:
					return "c.RenderJSON(v)", nil
				case reflect.String:
					return "fmt.Fprintf(c, v)", nil
				default:
					return "", fmt.Errorf("unknown return type: %s", m.Returns[0].Kind())
				}
			default:
				return "", fmt.Errorf("dont know how to handle %d return values", len(m.Returns))
			}
		},
		"returns": func(m Method) (string, error) {
			rs := []string{}
			for _, r := range m.Returns {
				rs = append(rs, r.String())
			}
			return strings.Join(rs, ", "), nil
		},
		"return_values": func(m Method) (string, error) {
			switch len(m.Returns) {
			case 1:
				return "err", nil
			case 2:
				switch m.Returns[0].Kind() {
				case reflect.Bool:
					return "false, err", nil
				case reflect.Int:
					return "0, err", nil
				case reflect.Interface, reflect.Ptr, reflect.Slice:
					return "nil, err", nil
				case reflect.String:
					return "\"\", err", nil
				default:
					return "", fmt.Errorf("unknown return type: %s", m.Returns[0].Kind())
				}
			default:
				return "", fmt.Errorf("dont know how to handle %d return values", len(m.Returns))
			}
		},
		"return_vars": func(m Method) (string, error) {
			switch len(m.Returns) {
			case 1:
				return "err", nil
			case 2:
				return "v, err", nil
			default:
				return "", fmt.Errorf("dont know how to handle %d return values", len(m.Returns))
			}
		},
		"vars": func(m Method) string {
			vs := []string{}
			for _, a := range m.Args {
				switch {
				case a.Int():
				case a.Option():
				case a.Path(m):
					vs = append(vs, fmt.Sprintf(`%s := c.Var("%s")`, a.Name, a.Name))
				case a.Slice():
					vs = append(vs, fmt.Sprintf(`%s := strings.Split(c.Value("%s"), ",")`, a.Name, a.Name))
				case a.Stream():
					vs = append(vs, fmt.Sprintf(`%s := c`, a.Name))
				default:
					vs = append(vs, fmt.Sprintf(`%s := c.Value("%s")`, a.Name, a.Name))
				}
			}
			return strings.Join(vs, "\n")
		},
	}
}

func renderTemplate(name string, data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	path := fmt.Sprintf("pkg/generate/template/%s.tmpl", name)
	file := filepath.Base(path)

	t, err := template.New(file).Funcs(templateHelpers()).ParseFiles(path)
	if err != nil {
		return nil, err
	}

	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
