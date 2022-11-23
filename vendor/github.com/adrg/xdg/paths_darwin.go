package xdg

import (
	"path/filepath"
)

func initBaseDirs(home string) {
	// Initialize base directories.
	baseDirs.dataHome = xdgPath(envDataHome, filepath.Join(home, "Library", "Application Support"))
	baseDirs.data = xdgPaths(envDataDirs, "/Library/Application Support")
	baseDirs.configHome = xdgPath(envConfigHome, filepath.Join(home, "Library", "Preferences"))
	baseDirs.config = xdgPaths(envConfigDirs, "/Library/Preferences")
	baseDirs.cacheHome = xdgPath(envCacheHome, filepath.Join(home, "Library", "Caches"))
	baseDirs.runtime = xdgPath(envRuntimeDir, filepath.Join(home, "Library", "Application Support"))

	// Initialize non-standard directories.
	baseDirs.applications = []string{
		"/Applications",
	}
	baseDirs.fonts = []string{
		filepath.Join(home, "Library/Fonts"),
		"/Library/Fonts",
		"/System/Library/Fonts",
		"/Network/Library/Fonts",
	}
}

func initUserDirs(home string) {
	UserDirs.Desktop = xdgPath(envDesktopDir, filepath.Join(home, "Desktop"))
	UserDirs.Download = xdgPath(envDownloadDir, filepath.Join(home, "Downloads"))
	UserDirs.Documents = xdgPath(envDocumentsDir, filepath.Join(home, "Documents"))
	UserDirs.Music = xdgPath(envMusicDir, filepath.Join(home, "Music"))
	UserDirs.Pictures = xdgPath(envPicturesDir, filepath.Join(home, "Pictures"))
	UserDirs.Videos = xdgPath(envVideosDir, filepath.Join(home, "Movies"))
	UserDirs.Templates = xdgPath(envTemplatesDir, filepath.Join(home, "Templates"))
	UserDirs.PublicShare = xdgPath(envPublicShareDir, filepath.Join(home, "Public"))
}
