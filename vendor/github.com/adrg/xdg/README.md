xdg
===
[![Build Status](https://travis-ci.org/adrg/xdg.svg?branch=master)](https://travis-ci.org/adrg/xdg)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/adrg/xdg)
[![License: MIT](https://img.shields.io/badge/license-MIT-red.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/adrg/xdg)](https://goreportcard.com/report/github.com/adrg/xdg)

Provides an implementation of the [XDG Base Directory Specification](https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html).
The specification defines a set of standard paths for storing application files,
including data and configuration files. For portability and flexibility reasons,
applications should use the XDG defined locations instead of hardcoding paths.
The package also includes the locations of well known user directories.
The current implementation supports Windows, Mac OS and most flavors of Unix.

Full documentation can be found at: https://godoc.org/github.com/adrg/xdg

## Installation
    go get github.com/adrg/xdg

## Default locations

The package defines sensible defaults for XDG variables which are empty or not
present in the environment.

#### XDG Base Directory

|                 | Unix                                | Mac OS                          | Windows                                 |
| :---            | :---                                | :-----                          | :---                                    |
| XDG_DATA_HOME   | `~/.local/share`                    | `~/Library/Application Support` | `%LOCALAPPDATA%`                        |
| XDG_DATA_DIRS   | `/usr/local/share`<br/>`/usr/share` | `/Library/Application Support`  | `%APPDATA%\Roaming`<br/>`%PROGRAMDATA%` |
| XDG_CONFIG_HOME | `~/.config`                         | `~/Library/Preferences`         | `%LOCALAPPDATA%`                        |
| XDG_CONFIG_DIRS | `/etc/xdg`                          | `/Library/Preferences`          | `%PROGRAMDATA%`                         |
| XDG_CACHE_HOME  | `~/.cache`                          | `~/Library/Caches`              | `%LOCALAPPDATA%\cache`                  |
| XDG_RUNTIME_DIR | `/run/user/UID`                     | `~/Library/Application Support` | `%LOCALAPPDATA%`                        |

#### XDG user directories

|                     | Unix          | Mac OS        | Windows                   |
| :---                | :---          | :-----        | :---                      |
| XDG_DESKTOP_DIR     | `~/Desktop`   | `~/Desktop`   | `%USERPROFILE%/Desktop`   |
| XDG_DOWNLOAD_DIR    | `~/Downloads` | `~/Downloads` | `%USERPROFILE%/Downloads` |
| XDG_DOCUMENTS_DIR   | `~/Documents` | `~/Documents` | `%USERPROFILE%/Documents` |
| XDG_MUSIC_DIR       | `~/Music`     | `~/Music`     | `%USERPROFILE%/Music`     |
| XDG_PICTURES_DIR    | `~/Pictures`  | `~/Pictures`  | `%USERPROFILE%/Pictures`  |
| XDG_VIDEOS_DIR      | `~/Videos`    | `~/Movies`    | `%USERPROFILE%/Videos`    |
| XDG_TEMPLATES_DIR   | `~/Templates` | `~/Templates` | `%USERPROFILE%/Templates` |
| XDG_PUBLICSHARE_DIR | `~/Public`    | `~/Public`    | `%PUBLIC%`                |

#### Non-standard directories

Application directories

```
Unix:
- $XDG_DATA_HOME/applications
- ~/.local/share/applications
- /usr/local/share/applications
- /usr/share/applications
- $XDG_DATA_DIRS/applications

Mac OS:
- /Applications

Windows:
- %APPDATA%\Roaming\Microsoft\Windows\Start Menu\Programs
```

Font Directories

```
Unix:
- $XDG_DATA_HOME/fonts
- ~/.fonts
- ~/.local/share/fonts
- /usr/local/share/fonts
- /usr/share/fonts
- $XDG_DATA_DIRS/fonts

Mac OS:
- ~/Library/Fonts
- /Library/Fonts
- /System/Library/Fonts
- /Network/Library/Fonts

Windows:
- %windir%\Fonts
- %LOCALAPPDATA%\Microsoft\Windows\Fonts
```

## Usage

#### XDG Base Directory

```go
package main

import (
	"log"

	"github.com/adrg/xdg"
)

func main() {
	// XDG Base Directory paths.
	log.Println("Home config directory:", xdg.DataHome)
	log.Println("Data directories:", xdg.DataDirs)
	log.Println("Home config directory:", xdg.ConfigHome)
	log.Println("Config directories:", xdg.ConfigDirs)
	log.Println("Cache directory:", xdg.CacheHome)
	log.Println("Runtime directory:", xdg.RuntimeDir)

	// Non-standard directories.
	log.Println("Application directories:", xdg.ApplicationDirs)
	log.Println("Font directories:", xdg.FontDirs)

	// Obtain a suitable location for application config files.
	// ConfigFile takes one parameter which must contain the name of the file,
	// but it can also contain a set of parent directories. If the directories
	// don't exists, they will be created relative to the base config directory.
	configFilePath, err := xdg.ConfigFile("appname/config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Save the config file at:", configFilePath)

	// For other types of application files use:
	// xdg.DataFile()
	// xdg.CacheFile()
	// xdg.RuntimeFile()

	// Finding application config files.
	// SearchConfigFile takes one parameter which must contain the name of
	// the file, but it can also contain a set of parent directories relative
	// to the config search paths (xdg.ConfigHome and xdg.ConfigDirs).
	configFilePath, err = xdg.SearchConfigFile("appname/config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Config file was found at:", configFilePath)

	// For other types of application files use:
	// xdg.SearchDataFile()
	// xdg.SearchCacheFile()
	// xdg.SearchRuntimeFile()
}
```

#### XDG user directories

```go
package main

import (
	"log"

	"github.com/adrg/xdg"
)

func main() {
	// XDG user directories.
	log.Println("Desktop directory:", xdg.UserDirs.Desktop)
	log.Println("Download directory:", xdg.UserDirs.Download)
	log.Println("Documents directory:", xdg.UserDirs.Documents)
	log.Println("Music directory:", xdg.UserDirs.Music)
	log.Println("Pictures directory:", xdg.UserDirs.Pictures)
	log.Println("Videos directory:", xdg.UserDirs.Videos)
	log.Println("Templates directory:", xdg.UserDirs.Templates)
	log.Println("Public directory:", xdg.UserDirs.PublicShare)
}
```

## References
For more information see the
[XDG Base Directory Specification](https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html) and
[XDG user directories](https://wiki.archlinux.org/index.php/XDG_user_directories).

## Contributing

Contributions in the form of pull requests, issues or just general feedback,
are always welcome.
See [CONTRIBUTING.MD](https://github.com/adrg/xdg/blob/master/CONTRIBUTING.md).

## License
Copyright (c) 2014 Adrian-George Bostan.

This project is licensed under the [MIT license](https://opensource.org/licenses/MIT).
See [LICENSE](https://github.com/adrg/xdg/blob/master/LICENSE) for more details.
