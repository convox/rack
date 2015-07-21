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

var _data_dockerfile_node = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x72\x0b\xf2\xf7\x55\xc8\xcb\x4f\x49\xb5\x32\xd0\x33\x34\xe0\xe2\x72\x8d\x08\xf0\x0f\x76\x55\x30\x36\x30\x30\xe0\x72\xf5\x0b\x53\x08\xf0\x0f\x0a\x81\xf0\xb8\xc2\xfd\x83\xbc\x5d\x3c\x83\x14\xf4\x13\x0b\x0a\xb8\xb8\x9c\xfd\x03\x22\x15\x0a\x12\x93\xb3\x13\xd3\x53\xf5\xb2\x8a\xf3\xf3\xc0\xe2\xfa\xc8\x22\x5c\x41\xa1\x7e\x0a\x79\x05\xb9\x0a\x99\x79\xc5\x25\x89\x39\x39\x50\x4d\x7a\x30\x13\x7c\x5d\x14\xa2\x95\x80\xf2\x4a\x3a\x0a\x4a\x40\x05\x45\x25\x4a\xb1\x5c\x80\x00\x00\x00\xff\xff\x54\x98\x55\x86\x90\x00\x00\x00")

func data_dockerfile_node_bytes() ([]byte, error) {
	return bindata_read(
		_data_dockerfile_node,
		"data/Dockerfile.node",
	)
}

func data_dockerfile_node() (*asset, error) {
	bytes, err := data_dockerfile_node_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/Dockerfile.node", size: 144, mode: os.FileMode(420), modTime: time.Unix(1437183991, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _data_dockerfile_rails = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x54\x8e\xcb\xae\x82\x30\x10\x86\xf7\xf3\x14\x93\xae\x0f\x97\x1c\x77\x6e\x05\x8d\x31\x50\x52\xef\x31\x2e\x8a\x8c\x86\x58\xa1\x29\xc5\x84\xb7\x97\x60\x5d\x34\xb3\xfa\xbf\x3f\xf3\xcd\x2c\x05\xcf\xd0\xf4\xe5\x30\xff\x0f\xc7\x01\x10\xfb\x1c\xa5\xb6\xc1\x83\x2c\xf6\xba\x92\x96\x3c\x14\x0c\x58\x37\x9d\x95\x4a\x61\xd3\x56\x04\x90\x9e\x0a\xbe\x4d\x71\x16\xc7\x31\xa4\xf9\x01\x0b\x2e\x76\xdf\x04\x47\x2e\x36\xc9\x5a\x60\x24\xb5\x06\x58\xf0\xe2\x8c\x2b\x7a\xdd\x6b\x45\x13\x8a\x5c\xf0\x9a\x50\xb5\xb7\xa7\x57\x4f\x64\xfa\xa1\xec\x9b\x6a\xdc\x75\xf7\x9d\x31\xfc\xe9\xb3\x04\x2f\xcc\xc8\x5a\x75\xec\x0f\x59\x47\xe6\x4d\x86\x5d\xe1\x13\x00\x00\xff\xff\x72\x72\xc5\xa7\xe1\x00\x00\x00")

func data_dockerfile_rails_bytes() ([]byte, error) {
	return bindata_read(
		_data_dockerfile_rails,
		"data/Dockerfile.rails",
	)
}

func data_dockerfile_rails() (*asset, error) {
	bytes, err := data_dockerfile_rails_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/Dockerfile.rails", size: 225, mode: os.FileMode(420), modTime: time.Unix(1437442798, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
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

	info := bindata_file_info{name: "data/Dockerfile.ruby", size: 152, mode: os.FileMode(420), modTime: time.Unix(1436479841, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _data_dockerfile_unknown = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x24\xc8\x41\xaa\xc2\x30\x14\x05\xd0\x79\x56\x71\xc9\xb8\xfc\xbf\x88\x06\x41\xa4\x44\x32\x11\x11\x07\x31\xbd\xda\xd0\xf0\x12\xea\x0b\xe2\xee\x15\x9c\x1d\xce\x2e\xf8\x09\xfd\xd6\x45\xbb\x31\x27\x1f\x0e\x6e\x1f\xf0\x1f\x5b\x33\xa3\x3f\x9e\xf1\xf7\xb3\x19\x27\x87\x8b\x65\x5a\xaa\x1d\x60\xbf\x55\x72\x8a\x9a\xab\x40\xdf\x8d\xe8\xb2\x4a\x7d\xc9\x80\x56\x18\x9f\x04\xe7\xac\xd0\x85\x78\x50\xb8\x45\xe5\x0c\x57\xd3\xca\xed\x9e\x0b\xed\xd5\x7c\x02\x00\x00\xff\xff\x1b\x6d\x99\x86\x76\x00\x00\x00")

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

	info := bindata_file_info{name: "data/Dockerfile.unknown", size: 118, mode: os.FileMode(420), modTime: time.Unix(1437183991, 0)}
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
	"data/Dockerfile.node": data_dockerfile_node,
	"data/Dockerfile.rails": data_dockerfile_rails,
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
		"Dockerfile.node": &_bintree_t{data_dockerfile_node, map[string]*_bintree_t{
		}},
		"Dockerfile.rails": &_bintree_t{data_dockerfile_rails, map[string]*_bintree_t{
		}},
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

