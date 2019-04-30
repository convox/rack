package atom

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	ca "github.com/convox/rack/pkg/atom/pkg/apis/convox/v1"
	cv "github.com/convox/rack/pkg/atom/pkg/client/clientset/versioned"
	"github.com/convox/rack/pkg/templater"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/apimachinery/pkg/api/errors"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	templates = templater.New(packr.NewBox("../atom/templates"), nil)
)

type Client struct {
	config *rest.Config
	convox cv.Interface
	k8s    kubernetes.Interface
}

func New(cfg *rest.Config) (*Client, error) {
	cc, err := cv.NewForConfig(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	c := &Client{
		config: cfg,
		convox: cc,
		k8s:    kc,
	}

	return c, nil
}

func Apply(namespace, name, version string, template []byte, timeout int32) error {
	params := map[string]interface{}{
		"Name":      name,
		"Namespace": namespace,
		"Template":  base64.StdEncoding.EncodeToString(template),
		"Timeout":   timeout,
		"Version":   version,
	}

	if err := exec.Command("kubectl", "get", fmt.Sprintf("ns/%s", namespace)).Run(); err != nil {
		data, err := templates.Render("namespace.yml.tmpl", params)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := kubectlApply(data); err != nil {
			return err
		}

		for {
			if err := exec.Command("kubectl", "get", fmt.Sprintf("ns/%s", namespace)).Run(); err == nil {
				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	data, err := templates.Render("atom.yml.tmpl", params)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := kubectlApply(data); err != nil {
		return err
	}

	return nil
}

func Wait(namespace, name string) error {
	for {
		data, err := exec.Command("kubectl", "get", fmt.Sprintf("atom/%s", name), "-n", namespace, "-o", "jsonpath={.status}").CombinedOutput()
		if err != nil {
			return errors.WithStack(err)
		}

		switch string(data) {
		case "Success":
			return nil
		case "Failure":
			return errors.WithStack(fmt.Errorf("atom failed"))
		}
	}
}

func (c *Client) Apply(ns, name string, version string, template []byte, timeout int32) error {
	if _, err := c.k8s.CoreV1().Namespaces().Get(ns, am.GetOptions{}); ae.IsNotFound(err) {
		_, err := c.k8s.CoreV1().Namespaces().Create(&ac.Namespace{
			ObjectMeta: am.ObjectMeta{
				Name: ns,
			},
		})
		if err != nil {
			return errors.WithStack(err)
		}

		for {
			if ns, err := c.k8s.CoreV1().Namespaces().Get(ns, am.GetOptions{}); err == nil && ns != nil {
				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	a, err := c.convox.ConvoxV1().Atoms(ns).Get(name, am.GetOptions{})
	switch {
	case ae.IsNotFound(err):
		a, err = c.convox.ConvoxV1().Atoms(ns).Create(&ca.Atom{
			ObjectMeta: am.ObjectMeta{
				Name: name,
			},
		})
		if err != nil {
			return errors.WithStack(err)
		}
	case err != nil:
		return errors.WithStack(err)
	default:
		a.Spec.Previous = a.Spec.Current
	}

	a.Spec.Current.Version = version
	a.Spec.Current.Template = template
	a.Spec.ProgressDeadlineSeconds = timeout
	a.Status = "Pending"

	if _, err := c.convox.ConvoxV1().Atoms(ns).Update(a); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) Status(ns, name string) (string, string, error) {
	a, err := c.convox.ConvoxV1().Atoms(ns).Get(name, am.GetOptions{})
	if ae.IsNotFound(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", errors.WithStack(err)
	}

	return string(a.Status), a.Spec.Current.Version, nil
}

func (c *Client) Wait(ns, name string) error {
	for {
		a, err := c.convox.ConvoxV1().Atoms(ns).Get(name, am.GetOptions{})
		if err != nil {
			return errors.WithStack(err)
		}

		switch a.Status {
		case "Success":
			return nil
		case "Failure":
			return errors.WithStack(fmt.Errorf("atom failed"))
		}
	}
}

func (c *Client) apply(a *ca.Atom) error {
	var err error

	a.Status = "Building"

	a, err = c.convox.ConvoxV1().Atoms(a.Namespace).Update(a)
	if err != nil {
		return errors.WithStack(err)
	}

	cs, err := extractConditions(a.Spec.Current.Template)
	if err != nil {
		return errors.WithStack(err)
	}

	a.Spec.Conditions = cs

	out, err := applyTemplate(a.Spec.Current.Template, fmt.Sprintf("atom=%s.%s", a.Namespace, a.Name))
	if err != nil {
		return errors.WithStack(errors.New(strings.TrimSpace(string(out))))
	}

	time.Sleep(1 * time.Second)

	a.Started = am.Now()
	a.Status = "Running"

	a, err = c.convox.ConvoxV1().Atoms(a.Namespace).Update(a)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) check(a *ca.Atom) (bool, error) {
	cfg := *c.config

	cfg.APIPath = "/apis"
	cfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	for _, c := range a.Spec.Conditions {
		gv, err := schema.ParseGroupVersion(c.ApiVersion)
		if err != nil {
			return false, errors.WithStack(err)
		}

		cfg.GroupVersion = &gv

		rc, err := rest.RESTClientFor(&cfg)
		if err != nil {
			return false, errors.WithStack(err)
		}

		data, err := rc.Get().Namespace(c.Namespace).Name(c.Name).VersionedParams(&am.GetOptions{}, scheme.ParameterCodec).Resource(fmt.Sprintf("%ss", strings.ToLower(c.Kind))).Do().Raw()
		if err != nil {
			return false, errors.WithStack(err)
		}

		var o struct {
			Status struct {
				Conditions []struct {
					Type   string
					Status string
					Reason string
				}
			}
		}

		if err := json.Unmarshal(data, &o); err != nil {
			return false, errors.WithStack(err)
		}

		css := map[string]string{}
		crs := map[string]string{}

		for _, c := range o.Status.Conditions {
			css[c.Type] = c.Status
			crs[c.Type] = c.Reason
		}

		for k, c := range c.Conditions {
			if c.Status != css[k] {
				return false, nil
			}
			if c.Reason != "" && c.Reason != crs[k] {
				return false, nil
			}
		}
	}

	return true, nil
}

func (c *Client) rollback(a *ca.Atom) error {
	out, err := applyTemplate(a.Spec.Previous.Template, fmt.Sprintf("atom=%s.%s", a.Namespace, a.Name))
	if err != nil {
		return errors.WithStack(errors.New(strings.TrimSpace(string(out))))
	}

	time.Sleep(1 * time.Second)

	a.Spec.Current = a.Spec.Previous
	a.Started = am.Now()
	a.Status = "Rollback"

	a, err = c.convox.ConvoxV1().Atoms(a.Namespace).Update(a)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) status(a *ca.Atom, status string) error {
	var err error

	a.Status = ca.AtomStatus(status)

	a, err = c.convox.ConvoxV1().Atoms(a.Namespace).Update(a)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func applyLabels(data []byte, labels map[string]string) ([]byte, error) {
	var v map[string]interface{}

	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, errors.WithStack(err)
	}

	if len(v) == 0 {
		return data, nil
	}

	switch t := v["metadata"].(type) {
	case nil:
		v["metadata"] = map[string]interface{}{"labels": labels}
	case map[interface{}]interface{}:
		switch u := t["labels"].(type) {
		case nil:
			t["labels"] = labels
			v["metadata"] = t
		case map[interface{}]interface{}:
			for k, v := range labels {
				u[k] = v
			}
			t["labels"] = u
			v["metadata"] = t
		default:
			return nil, errors.WithStack(fmt.Errorf("unknown labels type: %T", u))
		}
	default:
		return nil, errors.WithStack(fmt.Errorf("unknown metadata type: %T", t))
	}

	pd, err := yaml.Marshal(v)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return pd, nil
}

func applyTemplate(data []byte, filter string) ([]byte, error) {
	rs, err := templateResources(filter)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	labels := parseLabels(filter)

	parts := bytes.Split(data, []byte("---\n"))

	for i := range parts {
		dp, err := applyLabels(parts[i], labels)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		parts[i] = dp
	}

	data = bytes.Join(parts, []byte("---\n"))

	args := []string{"--prune", "-l", filter}

	for _, r := range rs {
		args = append(args, "--prune-whitelist", r)
	}

	out, err := kubectlApplyOutput(data, args...)
	if err != nil {
		if !strings.Contains(string(out), "is immutable") {
			return out, errors.WithStack(err)
		}

		out, err := kubectlApplyOutput(data, "--force")
		if err != nil {
			return out, errors.WithStack(err)
		}
	}

	return out, nil
}

func extractConditions(data []byte) ([]ca.AtomCondition, error) {
	cs := []ca.AtomCondition{}

	parts := bytes.Split(data, []byte("---\n"))

	for _, p := range parts {
		var o struct {
			ApiVersion string `yaml:"apiVersion"`
			Kind       string
			Metadata   struct {
				Annotations map[string]string
				Name        string
				Namespace   string
			}
		}

		if err := yaml.Unmarshal(p, &o); err != nil {
			return nil, errors.WithStack(err)
		}

		if ac, ok := o.Metadata.Annotations["atom.conditions"]; ok {
			acps := strings.Split(ac, ",")

			acs := map[string]ca.AtomConditionMatch{}

			for _, acp := range acps {
				if acpps := strings.SplitN(acp, "=", 2); len(acpps) == 2 {
					if vps := strings.SplitN(acpps[1], "/", 2); len(vps) == 2 {
						acs[acpps[0]] = ca.AtomConditionMatch{Status: vps[0], Reason: vps[1]}
					} else {
						acs[acpps[0]] = ca.AtomConditionMatch{Status: vps[0]}
					}
				}
			}

			cs = append(cs, ca.AtomCondition{
				ApiVersion: o.ApiVersion,
				Conditions: acs,
				Kind:       o.Kind,
				Name:       o.Metadata.Name,
				Namespace:  o.Metadata.Namespace,
			})
		}
	}

	return cs, nil
}

func kubectlApply(data []byte, args ...string) error {
	if out, err := kubectlApplyOutput(data, args...); err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}

	return nil
}

func kubectlApplyOutput(data []byte, args ...string) ([]byte, error) {
	ka := append([]string{"apply", "-f", "-"}, args...)

	cmd := exec.Command("kubectl", ka...)

	cmd.Stdin = bytes.NewReader(data)

	return cmd.CombinedOutput()
}

func parseLabels(labels string) map[string]string {
	ls := map[string]string{}

	for _, part := range strings.Split(labels, ",") {
		ps := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(ps) == 2 {
			ls[ps[0]] = ps[1]
		}
	}

	return ls
}

func templateResources(filter string) ([]string, error) {
	data, err := exec.Command("kubectl", "api-resources", "--verbs=list", "--namespaced", "-o", "name").CombinedOutput()
	if err != nil {
		return []string{}, nil
	}

	ars := strings.Split(strings.TrimSpace(string(data)), "\n")

	rsh := map[string]bool{}

	data, err = exec.Command("kubectl", "get", "-l", filter, "--all-namespaces", "-o", "json", strings.Join(ars, ",")).CombinedOutput()
	if err != nil {
		return []string{}, nil
	}

	if strings.TrimSpace(string(data)) == "" {
		return []string{}, nil
	}

	var res struct {
		Items []struct {
			ApiVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.WithStack(err)
	}

	for _, i := range res.Items {
		av := i.ApiVersion

		if !strings.Contains(av, "/") {
			av = fmt.Sprintf("core/%s", av)
		}

		rsh[fmt.Sprintf("%s/%s", av, i.Kind)] = true
	}

	rs := []string{}

	for r := range rsh {
		rs = append(rs, r)
	}

	sort.Strings(rs)

	return rs, nil
}
