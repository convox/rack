package generate

var routes = map[string]string{
	"Initialize":           "",
	"AppCancel":            "POST /apps/{name}/cancel",
	"AppCreate":            "POST /apps",
	"AppDelete":            "DELETE /apps/{name}",
	"AppGet":               "GET /apps/{name}",
	"AppList":              "GET /apps",
	"AppLogs":              "SOCKET /apps/{name}/logs",
	"AppMetrics":           "GET /apps/{name}/metrics",
	"AppUpdate":            "PUT /apps/{name}",
	"BuildCreate":          "POST /apps/{app}/builds",
	"BuildExport":          "GET /apps/{app}/builds/{id}.tgz",
	"BuildGet":             "GET /apps/{app}/builds/{id}",
	"BuildImport":          "POST /apps/{app}/builds/import",
	"BuildLogs":            "SOCKET /apps/{app}/builds/{id}/logs",
	"BuildList":            "GET /apps/{app}/builds",
	"BuildUpdate":          "PUT /apps/{app}/builds/{id}",
	"CapacityGet":          "GET /system/capacity",
	"CertificateApply":     "PUT /apps/{app}/ssl/{service}/{port}",
	"CertificateCreate":    "POST /certificates",
	"CertificateDelete":    "DELETE /certificates/{id}",
	"CertificateGenerate":  "POST /certificates/generate",
	"CertificateList":      "GET /certificates",
	"EventSend":            "POST /events",
	"FilesDelete":          "DELETE /apps/{app}/processes/{pid}/files",
	"FilesDownload":        "GET /apps/{app}/processes/{pid}/files",
	"FilesUpload":          "POST /apps/{app}/processes/{pid}/files",
	"InstanceKeyroll":      "POST /instances/keyroll",
	"InstanceList":         "GET /instances",
	"InstanceShell":        "SOCKET /instances/{id}/shell",
	"InstanceTerminate":    "DELETE /instances/{id}",
	"ObjectDelete":         "DELETE /apps/{app}/objects/{key:.*}",
	"ObjectExists":         "HEAD /apps/{app}/objects/{key:.*}",
	"ObjectFetch":          "GET /apps/{app}/objects/{key:.*}",
	"ObjectList":           "GET /apps/{app}/objects",
	"ObjectStore":          "POST /apps/{app}/objects/{key:.*}",
	"ProcessExec":          "SOCKET /apps/{app}/processes/{pid}/exec",
	"ProcessGet":           "GET /apps/{app}/processes/{pid}",
	"ProcessList":          "GET /apps/{app}/processes",
	"ProcessLogs":          "SOCKET /apps/{app}/processes/{pid}/logs",
	"ProcessRun":           "POST /apps/{app}/services/{service}/processes",
	"ProcessStop":          "DELETE /apps/{app}/processes/{pid}",
	"Proxy":                "SOCKET /proxy/{host}/{port}",
	"ReleaseCreate":        "POST /apps/{app}/releases",
	"ReleaseGet":           "GET /apps/{app}/releases/{id}",
	"ReleaseList":          "GET /apps/{app}/releases",
	"ReleasePromote":       "POST /apps/{app}/releases/{id}/promote",
	"RegistryAdd":          "POST /registries",
	"RegistryList":         "GET /registries",
	"RegistryRemove":       "DELETE /registries/{server:.*}",
	"ResourceGet":          "GET /apps/{app}/resources/{name}",
	"ResourceList":         "GET /apps/{app}/resources",
	"ServiceList":          "GET /apps/{app}/services",
	"ServiceUpdate":        "PUT /apps/{app}/services/{name}",
	"SystemGet":            "GET /system",
	"SystemLogs":           "SOCKET /system/logs",
	"SystemInstall":        "",
	"SystemMetrics":        "GET /system/metrics",
	"SystemProcesses":      "GET /system/processes",
	"SystemReleases":       "GET /system/releases",
	"SystemResourceCreate": "POST /resources",
	"SystemResourceDelete": "DELETE /resources/{name}",
	"SystemResourceGet":    "GET /resources/{name}",
	"SystemResourceLink":   "POST /resources/{name}/links",
	"SystemResourceList":   "GET /resources",
	"SystemResourceTypes":  "OPTIONS /resources",
	"SystemResourceUnlink": "DELETE /resources/{name}/links/{app}",
	"SystemResourceUpdate": "PUT /resources/{name}",
	"SystemUninstall":      "",
	"SystemUpdate":         "PUT /system",
	"Workers":              "",
}

func Routes() ([]byte, error) {
	ms, err := Methods()
	if err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"Methods": ms,
	}

	data, err := renderTemplate("routes", params)
	if err != nil {
		return nil, err
	}

	return gofmt(data)
}
