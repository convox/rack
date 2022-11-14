package xdg

import "os"

// XDG Base Directory environment variables.
var (
	envDataHome   = "XDG_DATA_HOME"
	envDataDirs   = "XDG_DATA_DIRS"
	envConfigHome = "XDG_CONFIG_HOME"
	envConfigDirs = "XDG_CONFIG_DIRS"
	envCacheHome  = "XDG_CACHE_HOME"
	envRuntimeDir = "XDG_RUNTIME_DIR"
)

type baseDirectories struct {
	dataHome   string
	data       []string
	configHome string
	config     []string
	cacheHome  string
	runtime    string

	// Non-standard directories.
	fonts        []string
	applications []string
}

func (bd baseDirectories) dataFile(relPath string) (string, error) {
	return createPath(relPath, append([]string{bd.dataHome}, bd.data...))
}

func (bd baseDirectories) configFile(relPath string) (string, error) {
	return createPath(relPath, append([]string{bd.configHome}, bd.config...))
}

func (bd baseDirectories) cacheFile(relPath string) (string, error) {
	return createPath(relPath, []string{bd.cacheHome})
}

func (bd baseDirectories) runtimeFile(relPath string) (string, error) {
	fi, err := os.Lstat(bd.runtime)
	if err != nil {
		if os.IsNotExist(err) {
			return createPath(relPath, []string{bd.runtime})
		}
		return "", err
	}

	if fi.IsDir() {
		// The runtime directory must be owned by the user.
		if err = os.Chown(bd.runtime, os.Getuid(), os.Getgid()); err != nil {
			return "", err
		}
	} else {
		// For security reasons, the runtime directory cannot be a symlink.
		if err = os.Remove(bd.runtime); err != nil {
			return "", err
		}
	}

	return createPath(relPath, []string{bd.runtime})
}

func (bd baseDirectories) searchDataFile(relPath string) (string, error) {
	return searchFile(relPath, append([]string{bd.dataHome}, bd.data...))
}

func (bd baseDirectories) searchConfigFile(relPath string) (string, error) {
	return searchFile(relPath, append([]string{bd.configHome}, bd.config...))
}

func (bd baseDirectories) searchCacheFile(relPath string) (string, error) {
	return searchFile(relPath, []string{bd.cacheHome})
}

func (bd baseDirectories) searchRuntimeFile(relPath string) (string, error) {
	return searchFile(relPath, []string{bd.runtime})
}
