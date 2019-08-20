// Code generated by go-bindata.
// sources:
// 000001_create_org_user_tables.down.sql
// 000001_create_org_user_tables.up.sql
// DO NOT EDIT!

package schema

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var __000001_create_org_user_tablesDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x2d\x4e\x2d\x2a\xb6\xe6\xc2\x2a\x97\x5f\x94\x5e\x6c\xcd\x05\x08\x00\x00\xff\xff\x93\xee\xc5\x1a\x37\x00\x00\x00")

func _000001_create_org_user_tablesDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__000001_create_org_user_tablesDownSql,
		"000001_create_org_user_tables.down.sql",
	)
}

func _000001_create_org_user_tablesDownSql() (*asset, error) {
	bytes, err := _000001_create_org_user_tablesDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "000001_create_org_user_tables.down.sql", size: 55, mode: os.FileMode(436), modTime: time.Unix(1565981121, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __000001_create_org_user_tablesUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x90\x4f\x4b\xc3\x30\x18\xc6\xef\xf9\x14\x0f\x3b\x35\xa0\x30\x41\x4f\x9e\xea\xf6\x56\x82\x33\xd3\x34\x81\xed\x54\x82\x8d\x33\xb0\xb6\x92\xac\xfb\xfc\xf2\x6a\xf5\xb0\xb1\x8b\xc7\x3c\x7f\xc2\xef\x79\x17\x86\x4a\x4b\xa0\x8d\x25\x5d\xab\xb5\x86\xaa\xa0\xd7\x16\xb4\x51\xb5\xad\x31\x1b\xc7\xd8\x5e\x0f\x39\x7f\xce\xee\x85\x98\xc2\xb6\x7c\x58\x11\x86\xb4\xcb\x28\x04\x10\x5b\x38\xa7\x96\x70\x5a\xbd\x3a\xc2\x92\xaa\xd2\xad\x2c\xb8\xd9\xec\x42\x1f\x92\x3f\x84\xe6\x78\x5b\xc8\x2b\x01\x6e\x35\xbd\xef\x02\x8e\x3e\xbd\x7d\xf8\x54\xdc\xcd\xe5\xd4\x64\xbb\x1d\x3a\x1f\xfb\xcb\x09\x01\xbc\x18\xf5\x5c\x9a\x2d\x9e\x68\x5b\xc4\x56\x0a\x79\x0a\x36\xe6\x90\xfe\x49\x36\x15\xf8\xc9\xbf\x9c\x62\xb0\xfe\x1e\x53\x3e\x9c\x01\xb2\xb3\xf7\x17\x8c\xd0\xf9\xb8\xff\x13\x6f\xe6\xac\x9e\x0f\xe1\x64\xb5\x36\xa4\x1e\x35\x4b\x28\x7e\x80\x24\x0c\x55\x64\x48\x2f\xa8\xfe\xbe\xf9\xef\xe8\xaf\x00\x00\x00\xff\xff\xb4\x85\x91\x1a\xba\x01\x00\x00")

func _000001_create_org_user_tablesUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__000001_create_org_user_tablesUpSql,
		"000001_create_org_user_tables.up.sql",
	)
}

func _000001_create_org_user_tablesUpSql() (*asset, error) {
	bytes, err := _000001_create_org_user_tablesUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "000001_create_org_user_tables.up.sql", size: 442, mode: os.FileMode(436), modTime: time.Unix(1566260661, 0)}
	a := &asset{bytes: bytes, info: info}
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

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
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
	"000001_create_org_user_tables.down.sql": _000001_create_org_user_tablesDownSql,
	"000001_create_org_user_tables.up.sql": _000001_create_org_user_tablesUpSql,
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
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"000001_create_org_user_tables.down.sql": &bintree{_000001_create_org_user_tablesDownSql, map[string]*bintree{}},
	"000001_create_org_user_tables.up.sql": &bintree{_000001_create_org_user_tablesUpSql, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
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

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

