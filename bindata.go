package main

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

var _data_app_conf = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x6c\x8f\xd1\x4a\x03\x31\x10\x45\xdf\xe7\x2b\xae\x1f\x90\x0d\xad\xad\x60\x7f\x45\x7c\x08\xc9\xa0\xc1\x98\x84\x99\xd9\xb8\x82\x1f\xef\xee\xb6\x3e\x08\x7d\x3a\x90\x7b\x02\x67\xd4\x82\x18\x5a\x85\xcc\xb5\xf0\xe0\x82\x97\xe3\xe3\xe9\xfc\x8a\x50\x13\xf6\x91\x13\x52\x8b\x1f\x2c\xa4\xd6\xfa\x7f\xf5\x61\x77\x89\x84\xb5\x87\xaf\xfa\x47\x94\xfc\x99\x0d\xab\xb6\x91\x13\x51\x6f\x6a\x6e\xff\xcf\x0b\x47\x68\x61\xee\x38\x10\x69\x94\xdc\x8d\x80\x38\x4b\xc1\xbb\x59\xbf\x78\x7f\x78\x7a\x9e\x8e\xe7\xd3\x74\xa3\x2f\xc1\x58\xcd\xcf\xca\xe2\x52\xb0\x80\x9f\x5b\xd1\x56\x02\xe7\x34\xbf\xb9\x2e\x6d\xf9\x86\xcb\x70\x03\x7e\x04\xf1\xeb\xe4\xaf\xd6\xa4\x2b\x2e\xf7\x1e\x11\x5b\x1d\x6d\xf1\xb9\x66\x23\xde\x0e\xbe\xe6\xfc\x06\x00\x00\xff\xff\xc9\x33\xf0\xc7\x15\x01\x00\x00")

func data_app_conf_bytes() ([]byte, error) {
	return bindata_read(
		_data_app_conf,
		"data/app.conf",
	)
}

func data_app_conf() (*asset, error) {
	bytes, err := data_app_conf_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/app.conf", size: 277, mode: os.FileMode(420), modTime: time.Unix(1428451593, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _data_cloudwatch_logs_conf = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x6c\x8e\xb1\x0a\xc2\x40\x0c\x86\xf7\x3c\x45\x97\x8c\x9a\xbd\xe0\xe0\xe6\x22\x14\x9c\xa4\x94\x92\xd2\x58\x0a\xbd\xde\x91\x4b\x75\x28\x7d\x77\x63\x41\x70\x70\xc8\x90\xff\xfb\x42\xfe\x7a\x90\x59\x94\xa7\x06\xb2\xb1\x49\xfb\x18\x27\x29\x4e\x05\x3d\x59\x89\x5f\x79\x8a\x43\x26\x76\xc7\x0e\x3b\x07\xa8\x77\xe4\x39\x2d\xc9\x23\x35\xe2\x94\x8e\xbe\x37\xf0\x7b\xfb\x47\x00\x9f\x76\xd0\xb8\xa4\x76\xe6\xf0\x31\xd7\xf5\x5c\x55\xdb\xb6\x83\x6c\x2a\x1c\xbe\x24\xa9\x74\x31\x1a\xf4\xfe\xd3\xc6\xe0\xbd\xa2\x06\x36\x27\xd8\x13\x76\x84\xf7\x12\x2f\x25\x5e\x4b\xbc\xc1\x3b\x00\x00\xff\xff\xe9\xef\x6e\x68\xc3\x00\x00\x00")

func data_cloudwatch_logs_conf_bytes() ([]byte, error) {
	return bindata_read(
		_data_cloudwatch_logs_conf,
		"data/cloudwatch-logs.conf",
	)
}

func data_cloudwatch_logs_conf() (*asset, error) {
	bytes, err := data_cloudwatch_logs_conf_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/cloudwatch-logs.conf", size: 195, mode: os.FileMode(420), modTime: time.Unix(1427816018, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _data_packer_json = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xbc\x95\xcb\x6e\xdb\x3a\x10\x86\xf7\x7e\x8a\x81\x36\xd9\x44\x11\xce\x69\x80\x02\xd9\xa9\x86\x50\x64\x91\xa4\xb0\x7a\x59\x34\x81\x40\x51\x63\x9b\xb0\x44\x12\xbc\xb8\x49\x05\xbd\x7b\x49\x5d\x6c\x59\xb1\x8b\xa2\x31\xba\xb1\xe1\x99\x4f\x9c\xf9\xff\x19\xca\xf5\x0c\x20\x28\x50\x53\xc5\xa4\x61\x82\x07\x37\x10\xcc\x05\xdf\x8a\x67\x88\xa5\x0c\x2e\x7d\x7a\x4b\x14\x23\x79\x89\xda\x25\x3d\xef\x42\xf7\xf1\x5d\xe2\x7e\x71\x5b\x96\x97\x5d\x24\x7d\xf8\xb2\x98\x4f\x62\x1f\xe2\x34\xc9\xe2\xbb\x5b\x7f\x28\xa9\x58\x98\x5f\x5f\xbf\x7b\xbf\x2c\x68\xd0\xe7\xe3\x6f\x69\xb6\x48\x3e\xde\x3e\xdc\x7b\xc2\xea\x10\x89\x36\xe1\x7f\xe3\x74\x3c\x9f\x27\x69\xea\xd3\x75\x8d\x7c\x0b\x8f\xa3\xe8\x63\xd0\x34\x63\x36\x4d\xe6\x8b\xe4\xf3\x94\xed\xa2\x2d\xeb\xd0\xa6\x55\x94\x5b\x56\x16\xa8\xbc\xa0\xef\xed\xf3\x9d\x2c\x97\x31\x2f\x12\xbb\x76\xc9\x4f\xc1\x43\xcc\x75\x5f\xc1\xe5\x14\xae\x7a\x87\xea\xda\x6a\x54\x7d\x81\x4e\xc1\xa8\x19\x87\x12\x4a\x51\xeb\x6c\x83\x2f\xaf\xf0\x57\xbd\x3b\x5c\x23\x55\x68\x8e\xe2\xa3\xf6\xf7\xb8\xb0\x8a\x62\xe6\x2c\x3d\xc0\x07\xbb\x0f\x61\xc6\xb5\x21\xdc\xe1\x83\x34\xf3\xff\x55\xc5\xa8\x12\xa3\xf3\xf4\x3a\xf3\x67\x70\x52\xb5\x84\xcd\x2d\x37\x76\xa4\xa6\x62\xd9\x90\xdb\x55\xf3\x2b\xe0\x2b\x85\x75\x6d\x58\x85\xae\x48\x25\xc7\x75\x95\xe5\x99\x21\xab\xfd\xd2\x74\x8b\x73\xf2\x98\x7e\x26\x41\xcf\x36\xb3\xe1\xf3\xa9\x1d\x99\x54\x62\xcb\xb4\xf3\xff\xb7\x63\xd3\x6b\x2c\xcb\x7d\x0f\xf8\x8c\xd4\x1a\xcc\xa8\xa8\x2a\xc2\x0b\x4f\xf8\xc5\xa8\x6b\xb8\xfa\x4a\x94\x86\xa6\x01\x6d\x0b\x01\x61\x02\x61\x0a\x7a\x0d\x17\x3e\xf5\x89\x98\xb5\x4b\x5d\x8c\x3d\x2c\x19\xc7\x5d\xd9\x36\x56\x6d\x0a\xa6\x20\x6a\xdb\xde\x91\x2e\x4e\xd7\xe2\x07\x87\xce\xc2\x9b\xee\x6b\xa0\x7a\xe8\xa9\x93\x76\x79\x5c\xc2\x92\x95\x38\x1d\xf5\x81\x63\xdd\x4d\xf3\x9e\x45\x7b\xce\x5d\x61\xc3\x38\x19\xae\xf0\xb8\xe0\xa9\x42\x13\xaf\x8e\x69\xd4\x68\x20\xc4\x03\x75\xc5\x11\xc9\x91\xd5\x2a\x2a\x05\x25\x65\x94\x33\x1e\x15\x82\x6e\x50\x85\xce\x74\x29\x34\x42\x28\x81\x48\x09\x7f\xf7\x94\x74\x6f\x93\x3f\xf3\xed\x1f\x8e\x5e\x55\x10\xaa\xe5\x59\xa6\xea\x34\x5e\x51\xc1\x97\xa7\x27\x69\x2a\x19\xed\xa8\x37\x3a\x70\x9e\xc5\xdf\xc2\x41\x53\x10\xa1\xa1\x11\xe3\xcc\x4c\xfa\x7c\xfb\xc4\xce\xd2\x6f\xb7\x58\xed\x26\x01\x6d\xff\xda\x22\xb2\x42\x6e\xc6\xbb\x78\x84\xf1\x82\x26\x3a\xfc\xcb\x68\xd6\xcc\x7e\x05\x00\x00\xff\xff\x64\x75\x3c\x3f\x34\x07\x00\x00")

func data_packer_json_bytes() ([]byte, error) {
	return bindata_read(
		_data_packer_json,
		"data/packer.json",
	)
}

func data_packer_json() (*asset, error) {
	bytes, err := data_packer_json_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "data/packer.json", size: 1844, mode: os.FileMode(420), modTime: time.Unix(1428526260, 0)}
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
	"data/app.conf": data_app_conf,
	"data/cloudwatch-logs.conf": data_cloudwatch_logs_conf,
	"data/packer.json": data_packer_json,
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
		"app.conf": &_bintree_t{data_app_conf, map[string]*_bintree_t{
		}},
		"cloudwatch-logs.conf": &_bintree_t{data_cloudwatch_logs_conf, map[string]*_bintree_t{
		}},
		"packer.json": &_bintree_t{data_packer_json, map[string]*_bintree_t{
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

