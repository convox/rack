package kaws

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/k8s"
	"gopkg.in/cheggaaa/pb.v1"
)

const (
	cfnTemplate = "https://convox.s3.amazonaws.com/release/%s/provider/kaws/cfn/rack.yml"
)

var (
	systemTemplates = []string{"api", "autoscale", "calico", "custom", "iam", "router"}
)

func (p *Provider) SystemHost() string {
	return p.BaseDomain
}

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	if err := checkKubectl(); err != nil {
		return "", err
	}

	if err := helpers.AwsCredentialsLoad(); err != nil {
		return "", err
	}

	name := helpers.DefaultString(opts.Name, "convox")
	version := helpers.DefaultString(opts.Version, "dev")

	params := map[string]string{}

	// if opts.Id != nil {
	//   params["ClientId"] = *opts.Id
	// }

	if opts.Parameters != nil {
		for k, v := range opts.Parameters {
			params[k] = v
		}
	}

	raw := helpers.DefaultBool(opts.Raw, false)

	password := params["Password"]
	delete(params, "Password")

	if password == "" {
		pw, err := helpers.RandomString(40)
		if err != nil {
			return "", err
		}
		password = pw
	}

	var stackTotal int

	if !raw {
		fmt.Fprintf(w, "Preparing... ")
	}

	var bar *pb.ProgressBar

	s, err := session.NewSession()
	if err != nil {
		return "", err
	}

	cf := cloudformation.New(s)

	template := fmt.Sprintf(cfnTemplate, version)

	tags := map[string]string{
		"rack":   name,
		"system": "convox",
	}

	err = helpers.CloudformationInstall(cf, name, template, params, tags, func(current, total int) {
		stackTotal = total

		if raw {
			fmt.Fprintf(w, "{ \"stack\": %q, \"current\": %d, \"total\": %d }\n", name, current, total+2)
			return
		}

		if bar == nil {
			fmt.Fprintf(w, "OK\n")
			bar = pb.New(total + 1)
			bar.Format(" ██  ")
			bar.Output = w
			bar.Prefix("Installing...")
			bar.ShowBar = false
			bar.ShowCounters = false
			bar.ShowTimeLeft = false
			bar.Start()
		}

		bar.Set(current)
	})
	if err != nil {
		return "", err
	}

	outputs, err := awsStackOutputs(name)
	if err != nil {
		return "", err
	}

	p.applyOutputs(outputs)

	config, err := writeKubeConfig(outputs)
	if err != nil {
		return "", err
	}

	os.Setenv("KUBECONFIG", config)

	if _, err := p.Provider.SystemInstall(w, opts); err != nil {
		return "", err
	}

	p.Password = password
	p.Rack = name
	p.Region = outputs["Region"]

	if err := p.systemUpdate(version); err != nil {
		return "", err
	}

	time.Sleep(10 * time.Second)

	if err := p.initializeIamRoles(); err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://convox:%s@%s", password, p.SystemHost())

	if raw {
		fmt.Fprintf(w, "{ \"stack\": %q, \"current\": %d, \"total\": %d }\n", name, stackTotal+1, stackTotal+2)
	} else {
		bar.Set(stackTotal + 1)
		bar.Finish()
		fmt.Fprintf(w, "Starting... ")
	}

	if err := endpointWait(url); err != nil {
		return "", err
	}

	if raw {
		fmt.Fprintf(w, "{ \"stack\": %q, \"current\": %d, \"total\": %d }\n", name, stackTotal+2, stackTotal+2)
	} else {
		fmt.Fprintf(w, "OK, %s\n", p.SystemHost())
	}

	return url, nil
}

func (p *Provider) SystemStatus() (string, error) {
	return "running", nil
}

func (p *Provider) SystemTemplate(version string) ([]byte, error) {
	params := map[string]interface{}{
		"Version": version,
	}

	ts := [][]byte{}

	data, err := p.Provider.SystemTemplate(version)
	if err != nil {
		return nil, err
	}

	ts = append(ts, data)

	for _, st := range systemTemplates {
		data, err := p.RenderTemplate(fmt.Sprintf("system/%s", st), params)
		if err != nil {
			return nil, err
		}

		ldata, err := k8s.ApplyLabels(data, "system=convox,provider=kaws")
		if err != nil {
			return nil, err
		}

		ts = append(ts, ldata)
	}

	return bytes.Join(ts, []byte("---\n")), nil
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	template := fmt.Sprintf(cfnTemplate, helpers.DefaultString(opts.Version, p.Provider.Version))

	if err := helpers.CloudformationUpdate(p.CloudFormation, p.Rack, template, nil, nil, p.EventTopic); err != nil {
		return err
	}

	if err := p.Provider.SystemUpdate(opts); err != nil {
		return err
	}

	return nil
}

func (p *Provider) systemUpdate(version string) error {
	params := map[string]interface{}{
		"AccountId":      p.AccountId,
		"AdminUser":      p.AdminUser,
		"AutoscalerRole": p.AutoscalerRole,
		"Cluster":        p.Cluster,
		"NodesRole":      p.NodesRole,
		"Password":       p.Password,
		"Rack":           p.Rack,
		"Region":         p.Region,
		"RouterCache":    p.RouterCache,
		"RouterHosts":    p.RouterHosts,
		"RouterRole":     p.RouterRole,
		"RouterTargets":  p.RouterTargets,
		"Version":        version,
	}

	data, err := p.RenderTemplate("cluster", params)
	if err != nil {
		return err
	}

	if err := k8s.Apply(data); err != nil {
		return err
	}

	data, err = p.RenderTemplate("config", params)
	if err != nil {
		return err
	}

	if err := p.ApplyWait(p.Rack, "config", version, data, fmt.Sprintf("system=convox,provider=kaws,rack=%s", p.Rack), 30); err != nil {
		return err
	}

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/provider/kaws/k8s/rack.yml", version)

	data, err = helpers.Get(template)
	if err != nil {
		return err
	}

	tags := map[string]string{
		"ACCOUNT": p.AccountId,
		"CLUSTER": p.Cluster,
		"HOST":    p.BaseDomain,
		"RACK":    p.Rack,
		"REGION":  p.Region,
		"SOCKET":  "/var/run/docker.sock",
	}

	for k, v := range tags {
		data = bytes.Replace(data, []byte(fmt.Sprintf("==%s==", k)), []byte(v), -1)
	}

	if err := p.Apply(p.Rack, "system", version, data, fmt.Sprintf("system=convox,provider=kaws,rack=%s", p.Rack), 300); err != nil {
		return err
	}

	return nil
}

// func (p *Provider) systemUpdate(version string) error {
//   if out, err := kubectl("scale", "deployment/api", "-n", p.Rack, "--replicas=2"); err != nil {
//     return fmt.Errorf("update error: %s", strings.TrimSpace(string(out)))
//   }

//   // if out, err := kubectl("patch", "deployment/api", "-p", `{"spec":{"template":{"spec":{"containers":[{"name":"main","env":[{"name":"SOCKET","value":"/var/run/docker.sock"}]}]}}}}`, "-n", p.Rack); err != nil {
//   //   return fmt.Errorf("update error: %s", strings.TrimSpace(string(out)))
//   // }

//   params := map[string]interface{}{
//     // "AccountId":      p.AccountId,
//     // "AdminUser":      p.AdminUser,
//     // "AutoscalerRole": p.AutoscalerRole,
//     // "Cluster":        p.Cluster,
//     // "NodesRole":      p.NodesRole,
//     // "Password":       p.Password,
//     // "Rack":           p.Rack,
//     // "Region":         p.Region,
//     // "RouterCache":    p.RouterCache,
//     // "RouterHosts":    p.RouterHosts,
//     // "RouterRole":     p.RouterRole,
//     // "RouterTargets":  p.RouterTargets,
//     // "Version":        version,
//   }

//   if out, err := p.ApplyTemplate("autoscale", "system=convox,provider=kaws,scope=autoscale", params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   if out, err := p.ApplyTemplate("calico", "system=convox,provider=kaws,scope=calico", params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   if out, err := p.ApplyTemplate("cluster", "system=convox,provider=kaws,scope=cluster", params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   if out, err := p.ApplyTemplate("config", fmt.Sprintf("system=convox,provider=kaws,scope=config,rack=%s", p.Rack), params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   if out, err := p.ApplyTemplate("custom", "system=convox,provider=kaws,scope=custom", params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   if out, err := p.ApplyTemplate("iam", "system=convox,provider=kaws,scope=iam", params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   if out, err := p.ApplyTemplate("system", "system=convox,provider=kaws,scope=system", params); err != nil {
//     return fmt.Errorf("update error: %s: %s", err, strings.TrimSpace(string(out)))
//   }

//   return nil
// }

func awsCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("aws", args...)

	cmd.Env = os.Environ()

	return cmd.CombinedOutput()
}

func awsStackOutputs(name string) (map[string]string, error) {
	data, err := awsCommand("cloudformation", "describe-stacks", "--stack-name", name, "--query", "Stacks[0].Outputs")
	if err != nil {
		return nil, fmt.Errorf(strings.TrimSpace(string(data)))
	}

	var okvs []struct {
		OutputKey   string
		OutputValue string
	}

	if err := json.Unmarshal(data, &okvs); err != nil {
		return nil, err
	}

	os := map[string]string{}

	for _, okv := range okvs {
		os[okv.OutputKey] = okv.OutputValue
	}

	return os, nil
}

func checkKubectl() error {
	ch := make(chan error, 1)

	go func() { ch <- exec.Command("kubectl").Run() }()
	go time.AfterFunc(3*time.Second, func() { ch <- fmt.Errorf("timeout") })

	if err := <-ch; err != nil {
		return fmt.Errorf("kubernetes not running or kubectl not configured, try `kubectl version`")
	}

	return nil
}

func endpointWait(url string) error {
	timeout := time.After(10 * time.Minute)

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	ht := *(http.DefaultTransport.(*http.Transport))
	ht.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	hc := &http.Client{Timeout: 2 * time.Second, Transport: &ht}

	for {
		select {
		case <-tick.C:
			res, err := hc.Get(fmt.Sprintf("%s/apps", url))
			if err == nil && res.StatusCode == 200 {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}

func writeKubeConfig(outputs map[string]string) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	config := filepath.Join(dir, "kubeconfig.yml")

	data := []byte(fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: aws
  name: aws
current-context: aws
kind: Config
preferences: {}
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      command: aws-iam-authenticator
      args:
        - "token"
        - "-i"
        - %s`, outputs["ClusterEndpoint"], outputs["ClusterCertificateAuthority"], outputs["Cluster"]))

	if err := ioutil.WriteFile(config, data, 0644); err != nil {
		return "", err
	}

	return config, nil
}
