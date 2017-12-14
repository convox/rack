package controllers

import (
	"net/http"

	"github.com/gorilla/mux"
)

func HandlerFunc(w http.ResponseWriter, req *http.Request) {
	router := NewRouter()
	router.ServeHTTP(w, req)
}

func NewRouter() (router *mux.Router) {
	router = mux.NewRouter()

	router.HandleFunc("/apps", api("app.list", AppList)).Methods("GET")
	router.HandleFunc("/apps", api("app.create", AppCreate)).Methods("POST")
	router.HandleFunc("/apps/{app}", api("app.get", AppGet)).Methods("GET")
	router.HandleFunc("/apps/{app}", api("app.delete", AppDelete)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/builds", api("build.list", BuildList)).Methods("GET")
	router.HandleFunc("/apps/{app}/builds", api("build.create", BuildCreate)).Methods("POST")
	router.HandleFunc("/apps/{app}/builds/{build}.tgz", api("build.export", BuildExport)).Methods("GET")
	router.HandleFunc("/apps/{app}/builds/{build}", api("build.get", BuildGet)).Methods("GET")
	router.HandleFunc("/apps/{app}/cancel", api("app.cancel", AppCancel)).Methods("POST")
	router.HandleFunc("/apps/{app}/environment", api("environment.list", EnvironmentGet)).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", api("environment.set", EnvironmentSet)).Methods("POST")
	router.HandleFunc("/apps/{app}/environment/{name}", api("environment.delete", EnvironmentDelete)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/parameters", api("parameters.list", ParametersList)).Methods("GET")
	router.HandleFunc("/apps/{app}/parameters", api("parameters.set", ParametersSet)).Methods("POST")
	router.HandleFunc("/apps/{app}/processes", api("process.list", ProcessList)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}", api("process.stop", ProcessStop)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/processes/{process}", api("process.get", ProcessGet)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}/run", api("process.run.detach", ProcessRunDetached)).Methods("POST")
	router.HandleFunc("/apps/{app}/releases", api("release.list", ReleaseList)).Methods("GET")
	router.HandleFunc("/apps/{app}/releases", api("release.list", ReleaseList)).Methods("GET").Queries("limit", "{limit:[0-9]+}")
	router.HandleFunc("/apps/{app}/releases/{release}", api("release.get", ReleaseGet)).Methods("GET")
	router.HandleFunc("/apps/{app}/releases/{release}/promote", api("release.promote", ReleasePromote)).Methods("POST")
	router.HandleFunc("/apps/{app}/ssl", api("ssl.list", SSLList)).Methods("GET")
	router.HandleFunc("/apps/{app}/ssl/{process}/{port}", api("ssl.update", SSLUpdate)).Methods("PUT")
	router.HandleFunc("/auth", api("auth", Auth)).Methods("GET")
	router.HandleFunc("/certificates", api("certificate.list", CertificateList)).Methods("GET")
	router.HandleFunc("/certificates", api("certificate.create", CertificateCreate)).Methods("POST")
	router.HandleFunc("/certificates/generate", api("certificate.generate", CertificateGenerate)).Methods("POST")
	router.HandleFunc("/certificates/{id}", api("certificate.delete", CertificateDelete)).Methods("DELETE")
	router.HandleFunc("/instances", api("instances.get", InstancesList)).Methods("GET")
	router.HandleFunc("/instances/{id}", api("instance.delete", InstanceTerminate)).Methods("DELETE")
	router.HandleFunc("/instances/keyroll", api("instances.keyroll", InstancesKeyroll)).Methods("POST")
	router.HandleFunc("/racks", api("rack.list", RackList)).Methods("GET")
	router.HandleFunc("/registries", api("registry.list", RegistryList)).Methods("GET")
	router.HandleFunc("/registries", api("registry.create", RegistryAdd)).Methods("POST")
	router.HandleFunc("/registries", api("registry.delete", RegistryRemove)).Methods("DELETE")
	router.HandleFunc("/resources", api("resource.list", ResourceList)).Methods("GET")
	router.HandleFunc("/resources", api("resource.create", ResourceCreate)).Methods("POST")
	router.HandleFunc("/resources/{resource}", api("resource.show", ResourceShow)).Methods("GET")
	router.HandleFunc("/resources/{resource}", api("resource.update", ResourceUpdate)).Methods("PUT")
	router.HandleFunc("/resources/{resource}", api("resource.delete", ResourceDelete)).Methods("DELETE")
	router.HandleFunc("/resources/{resource}/links", api("link.create", LinkCreate)).Methods("POST")
	router.HandleFunc("/resources/{resource}/links/{app}", api("link.delete", LinkDelete)).Methods("DELETE")
	router.HandleFunc("/sns", SNSProxy).Methods("POST").Headers("X-Amz-Sns-Message-Type", "Notification")
	router.HandleFunc("/sns", SNSConfirm).Methods("POST").Headers("X-Amz-Sns-Message-Type", "SubscriptionConfirmation")
	router.HandleFunc("/system", api("system.show", SystemShow)).Methods("GET")
	router.HandleFunc("/system", api("system.update", SystemUpdate)).Methods("PUT")
	router.HandleFunc("/system/capacity", api("system.capacity", SystemCapacity)).Methods("GET")
	router.HandleFunc("/system/processes", api("system.processes", SystemProcesses)).Methods("GET")
	router.HandleFunc("/switch", api("switch", Switch)).Methods("POST")

	// deprecated
	router.HandleFunc("/apps/{app}/formation", api("formation.list", FormationList)).Methods("GET")
	router.HandleFunc("/apps/{app}/formation/{process}", api("formation.set", FormationSet)).Methods("POST")

	// websockets
	router.Handle("/apps/{app}/logs", ws("app.logs", AppLogs)).Methods("GET")
	router.Handle("/apps/{app}/builds/{build}/logs", ws("build.logs", BuildLogs)).Methods("GET")
	router.Handle("/apps/{app}/processes/{pid}/exec", ws("process.exec.attach", ProcessExecAttached)).Methods("GET")
	router.Handle("/apps/{app}/processes/{process}/run", ws("process.run.attach", ProcessRunAttached)).Methods("GET")
	router.Handle("/instances/{id}/ssh", ws("instance.ssh", InstanceSSH)).Methods("GET")
	router.Handle("/proxy/{host}/{port}", ws("proxy", Proxy)).Methods("GET")
	router.Handle("/system/logs", ws("system.logs", SystemLogs)).Methods("GET")

	// utility
	router.HandleFunc("/boom", UtilityBoom).Methods("GET")
	router.HandleFunc("/check", UtilityCheck).Methods("GET")

	// limbo
	// auth.HandleFunc("/apps/{app}/debug", controllers.AppDebug).Methods("GET")
	// auth.HandleFunc("/apps/{app}/processes/{id}", controllers.ProcessStop).Methods("DELETE")
	// auth.HandleFunc("/apps/{app}/processes/{id}/top", controllers.ProcessTop).Methods("GET")
	// auth.HandleFunc("/top/{metric}", controllers.ClusterTop).Methods("GET")

	return
}
