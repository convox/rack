package sdk

import (
	"io"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
)

type Interface interface {
	structs.Provider

	// raw http
	Get(string, stdsdk.RequestOptions, interface{}) error

	// backwards compatibility
	AppParametersGet(string) (map[string]string, error)
	AppParametersSet(string, map[string]string) error
	BuildCreateUpload(string, io.Reader, structs.BuildCreateOptions) (*structs.Build, error)
	BuildImportMultipart(string, io.Reader) (*structs.Build, error)
	BuildImportUrl(string, io.Reader) (*structs.Build, error)
	CertificateCreateClassic(string, string, structs.CertificateCreateOptions) (*structs.Certificate, error)
	EnvironmentSet(string, []byte) (*structs.Release, error)
	EnvironmentUnset(string, string) (*structs.Release, error)
	FormationGet(string) (structs.Services, error)
	FormationUpdate(string, string, structs.ServiceUpdateOptions) error
	InstanceShellClassic(string, io.ReadWriter, structs.InstanceShellOptions) (int, error)
	ProcessRunAttached(string, string, io.ReadWriter, int, structs.ProcessRunOptions) (int, error)
	ProcessRunDetached(string, string, structs.ProcessRunOptions) (string, error)
	RegistryRemoveClassic(string) error
	ResourceCreateClassic(string, structs.ResourceCreateOptions) (*structs.Resource, error)
	ResourceUpdateClassic(string, structs.ResourceUpdateOptions) (*structs.Resource, error)
}
