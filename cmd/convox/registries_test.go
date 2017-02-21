package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestRegistriesList(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/registries",
			Code:   200,
			Response: client.Registries{
				client.Registry{
					Server:   "testRegistryServer",
					Username: "testRegistryUser",
					Password: "testRegistryPassword",
				},
			},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries",
			Exit:    0,
			Stdout:  "SERVER              USERNAME\ntestRegistryServer  testRegistryUser\n",
		},
	)
}

func TestRegistriesAddStdin(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "POST",
			Path:   "/registries",
			Body:   "email=&password=&server=https%3A%2F%2Findex.docker.io%2Fv1%2F&username=testRegistryUser",
			Code:   200,
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries add https://index.docker.io/v1/",
			Stdin:   "testRegistryUser\ntestRegistryPassword\n",
			Exit:    0,
			Stdout:  "Username: Password: \nDone.\n",
		},
	)
}

func TestRegistriesAdd(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "POST",
			Path:   "/registries",
			Body:   "email=&password=testRegistryPassword&server=https%3A%2F%2Findex.docker.io%2Fv1%2F&username=testRegistryUser",
			Code:   200,
		},
		test.Http{
			Method: "GET",
			Path:   "/registries",
			Code:   200,
			Response: client.Registries{
				client.Registry{
					Server:   "https://index.docker.io/v1/",
					Username: "testRegistryUser",
					Password: "testRegistryPassword",
				},
			},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries add https://index.docker.io/v1/ --username testRegistryUser --password testRegistryPassword",
			Exit:    0,
			Stdout:  "Done.\n",
		},
		test.ExecRun{
			Command: "convox registries",
			Exit:    0,
			Stdout:  "SERVER                       USERNAME\nhttps://index.docker.io/v1/  testRegistryUser\n",
		},
	)
}

func TestRegistriesAddInvalidAuth(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method:   "POST",
			Path:     "/registries",
			Body:     "email=&password=testRegistryPassword&server=https%3A%2F%2Findex.docker.io%2Fv1%2F&username=testRegistryUser",
			Code:     401,
			Response: client.Error{"unable to authenticate"},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries add https://index.docker.io/v1/ --username testRegistryUser --password testRegistryPassword",
			Exit:    1,
			Stderr:  "unable to authenticate\n",
		},
	)
}

func TestRegistriesAddEcrRegistry(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "POST",
			Path:     "/registries",
			Body:     "email=&password=testRegistryPassword&server=123456789012.dkr.ecr.us-east-1.amazonaws.com&username=testRegistryUser",
			Code:     500,
			Response: client.Error{"can't add the rack's internal registry: 123456789012.dkr.ecr.us-east-1.amazonaws.com"},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries add 123456789012.dkr.ecr.us-east-1.amazonaws.com --username testRegistryUser --password testRegistryPassword",
			Exit:    1,
			Stderr:  "can't add the rack's internal registry: 123456789012.dkr.ecr.us-east-1.amazonaws.com",
		},
	)
}

func TestRegistriesDelete(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method: "DELETE",
			Path:   "/registries",
			Code:   200,
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries remove foo",
			Exit:    0,
			Stdout:  "Done.\n",
		},
	)
}

func TestRegistriesDelete404(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "DELETE",
			Path:     "/registries",
			Code:     404,
			Response: client.Error{"no such registry"},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox registries remove foo",
			Exit:    1,
			Stderr:  "no such registry",
		},
	)
}
