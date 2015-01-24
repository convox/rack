package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"os"
	"path"
	"path/filepath"
)

// bindata_read reads the given file from disk. It returns an error on failure.
func bindata_read(path, name string) ([]byte, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset %s at %s: %v", name, path, err)
	}
	return buf, err
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

// data_ds_store reads file data from disk. It returns an error on failure.
func data_ds_store() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/.DS_Store"
	name := "data/.DS_Store"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_app_tmpl reads file data from disk. It returns an error on failure.
func data_app_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/app.tmpl"
	name := "data/app.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_cluster_tmpl reads file data from disk. It returns an error on failure.
func data_cluster_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/cluster.tmpl"
	name := "data/cluster.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_packer_tmpl reads file data from disk. It returns an error on failure.
func data_packer_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/packer.tmpl"
	name := "data/packer.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_postgres_tmpl reads file data from disk. It returns an error on failure.
func data_postgres_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/postgres.tmpl"
	name := "data/postgres.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_process_tmpl reads file data from disk. It returns an error on failure.
func data_process_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/process.tmpl"
	name := "data/process.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_redis_tmpl reads file data from disk. It returns an error on failure.
func data_redis_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/redis.tmpl"
	name := "data/redis.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_upstart_tmpl reads file data from disk. It returns an error on failure.
func data_upstart_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/upstart.tmpl"
	name := "data/upstart.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// data_userdata_tmpl reads file data from disk. It returns an error on failure.
func data_userdata_tmpl() (*asset, error) {
	path := "/Users/david/Code/convox/builder/data/userdata.tmpl"
	name := "data/userdata.tmpl"
	bytes, err := bindata_read(path, name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("Error reading asset info %s at %s: %v", name, path, err)
	}

	a := &asset{bytes: bytes, info: fi}
	return a, err
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"data/.DS_Store": data_ds_store,
	"data/app.tmpl": data_app_tmpl,
	"data/cluster.tmpl": data_cluster_tmpl,
	"data/packer.tmpl": data_packer_tmpl,
	"data/postgres.tmpl": data_postgres_tmpl,
	"data/process.tmpl": data_process_tmpl,
	"data/redis.tmpl": data_redis_tmpl,
	"data/upstart.tmpl": data_upstart_tmpl,
	"data/userdata.tmpl": data_userdata_tmpl,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() (*asset, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"data": &_bintree_t{nil, map[string]*_bintree_t{
		".DS_Store": &_bintree_t{data_ds_store, map[string]*_bintree_t{
		}},
		"app.tmpl": &_bintree_t{data_app_tmpl, map[string]*_bintree_t{
		}},
		"cluster.tmpl": &_bintree_t{data_cluster_tmpl, map[string]*_bintree_t{
		}},
		"packer.tmpl": &_bintree_t{data_packer_tmpl, map[string]*_bintree_t{
		}},
		"postgres.tmpl": &_bintree_t{data_postgres_tmpl, map[string]*_bintree_t{
		}},
		"process.tmpl": &_bintree_t{data_process_tmpl, map[string]*_bintree_t{
		}},
		"redis.tmpl": &_bintree_t{data_redis_tmpl, map[string]*_bintree_t{
		}},
		"upstart.tmpl": &_bintree_t{data_upstart_tmpl, map[string]*_bintree_t{
		}},
		"userdata.tmpl": &_bintree_t{data_userdata_tmpl, map[string]*_bintree_t{
		}},
	}},
}}

// Restore an asset under the given directory
func RestoreAsset(dir, name string) error {
        data, err := Asset(name)
        if err != nil {
                return err
        }
        info, err := AssetInfo(name)
        if err != nil {
                return err
        }
        err = os.MkdirAll(_filePath(dir, path.Dir(name)), os.FileMode(0755))
        if err != nil {
                return err
        }
        err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
        if err != nil {
                return err
        }
        err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
        if err != nil {
                return err
        }
        return nil
}

// Restore assets under the given directory recursively
func RestoreAssets(dir, name string) error {
        children, err := AssetDir(name)
        if err != nil { // File
                return RestoreAsset(dir, name)
        } else { // Dir
                for _, child := range children {
                        err = RestoreAssets(dir, path.Join(name, child))
                        if err != nil {
                                return err
                        }
                }
        }
        return nil
}

func _filePath(dir, name string) string {
        cannonicalName := strings.Replace(name, "\\", "/", -1)
        return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

