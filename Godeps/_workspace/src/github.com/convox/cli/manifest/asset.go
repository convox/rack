package manifest

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"os"
	"time"
	"io/ioutil"
	"path"
	"path/filepath"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindata_file_info struct {
	name string
	size int64
	mode os.FileMode
	modTime time.Time
}

func (fi bindata_file_info) Name() string {
	return fi.name
}
func (fi bindata_file_info) Size() int64 {
	return fi.size
}
func (fi bindata_file_info) Mode() os.FileMode {
	return fi.mode
}
func (fi bindata_file_info) ModTime() time.Time {
	return fi.modTime
}
func (fi bindata_file_info) IsDir() bool {
	return false
}
func (fi bindata_file_info) Sys() interface{} {
	return nil
}

var _data_dockerfile_ruby = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x72\x0b\xf2\xf7\x55\x28\x2a\x4d\xaa\xb4\x32\xd2\x03\x42\x2e\x2e\xd7\x88\x00\xff\x60\x57\x05\x63\x03\x03\x03\x2e\x57\xbf\x30\x85\x00\xff\xa0\x10\x08\x8f\x2b\xdc\x3f\xc8\xdb\xc5\x33\x48\x41\x3f\xb1\xa0\x80\x8b\xcb\xd9\x3f\x20\x52\xc1\x3d\x35\x37\x2d\x33\x27\x15\x2c\xa4\x0f\xe5\xa0\xc8\xe8\xe5\xe4\x27\x67\xa3\x48\x83\x45\xb8\x82\x42\xfd\x14\x92\x4a\xf3\x52\x80\x7a\x33\xf3\x8a\x4b\x12\x73\x72\xa0\x26\xea\x41\x8c\x07\x04\x00\x00\xff\xff\xd7\x6f\x55\x75\x98\x00\x00\x00")

func data_dockerfile_ruby_bytes() ([]byte, error) {
	return bindata_read(
		_data_dockerfile_ruby,
		"data/Dockerfile.ruby",
	)
}

func data_dockerfile_ruby() (*asset, error) {
	bytes, err := data_dockerfile_ruby_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/Dockerfile.ruby", size: 152, mode: os.FileMode(420), modTime: time.Unix(1436377657, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _data_dockerfile_unknown = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x72\x0b\xf2\xf7\x55\x48\xce\xcf\x2b\xcb\xaf\xd0\x4f\xcc\x29\xc8\xcc\x4b\xb5\x32\xd6\x33\xe4\xe2\x0a\xf7\x0f\xf2\x76\xf1\x0c\x52\xd0\x4f\x2c\x28\xe0\x72\xf6\x0f\x88\x54\xd0\x83\xb0\xb9\x9c\x7d\x5d\x14\x8a\x33\xb8\x00\x01\x00\x00\xff\xff\xf9\x23\x5b\xee\x39\x00\x00\x00")

func data_dockerfile_unknown_bytes() ([]byte, error) {
	return bindata_read(
		_data_dockerfile_unknown,
		"data/Dockerfile.unknown",
	)
}

func data_dockerfile_unknown() (*asset, error) {
	bytes, err := data_dockerfile_unknown_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/Dockerfile.unknown", size: 57, mode: os.FileMode(420), modTime: time.Unix(1436389670, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
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
	"data/Dockerfile.ruby": data_dockerfile_ruby,
	"data/Dockerfile.unknown": data_dockerfile_unknown,
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
		"Dockerfile.ruby": &_bintree_t{data_dockerfile_ruby, map[string]*_bintree_t{
		}},
		"Dockerfile.unknown": &_bintree_t{data_dockerfile_unknown, map[string]*_bintree_t{
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

