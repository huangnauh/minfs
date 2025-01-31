// Copyright (c) 2021 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package minfs

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// Config is being used for storge of configuration items
type Config struct {
	bucket   string
	basePath string

	cache       string
	accountID   string
	accessKey   string
	secretKey   string
	secretToken string
	target      *url.URL
	mountpoint  string
	insecure    bool
	debug       bool

	uid  uint32
	gid  uint32
	mode os.FileMode
}

// AccessConfig - access credentials and version of `config.json`.
type AccessConfig struct {
	Version     string `json:"version"`
	AccessKey   string `json:"accessKey"`
	SecretKey   string `json:"secretKey"`
	SecretToken string `json:"secretToken"`
}

var MountTime time.Time

// InitMinFSConfig - Initialize MinFS configuration file.
func InitMinFSConfig() (*AccessConfig, error) {
	// Create db directory.
	if err := os.MkdirAll(globalDBDir, 0777); err != nil {
		return nil, err
	}
	MountTime = time.Now()
	// Config doesn't exist create it based on environment values.
	if _, err := os.Stat(globalConfigFile); err != nil {
		if os.IsNotExist(err) {
			log.Println("Initializing config.json for the first time, please update your access credentials.")
			ac := &AccessConfig{
				Version:     "1",
				AccessKey:   os.Getenv("MINFS_ACCESS_KEY"),
				SecretKey:   os.Getenv("MINFS_SECRET_KEY"),
				SecretToken: os.Getenv("MINFS_SECRET_TOKEN"),
			}
			acBytes, jerr := json.Marshal(ac)
			if jerr != nil {
				return nil, jerr
			}
			if err = ioutil.WriteFile(globalConfigFile, acBytes, 0666); err != nil {
				return nil, err
			}
			return ac, nil
		} // Exists but not accessible, fail.
		return nil, err
	} // Config exists, proceed to read.
	acBytes, err := ioutil.ReadFile(globalConfigFile)
	if err != nil {
		return nil, err
	}
	ac := &AccessConfig{}
	if err = json.Unmarshal(acBytes, ac); err != nil {
		return nil, err
	}
	// Override if access keys are set through env.
	accessKey := os.Getenv("MINFS_ACCESS_KEY")
	secretKey := os.Getenv("MINFS_SECRET_KEY")
	secretToken := os.Getenv("MINFS_SECRET_TOKEN")
	if accessKey != "" {
		ac.AccessKey = accessKey
	}
	if secretKey != "" {
		ac.SecretKey = secretKey
	}
	if secretToken != "" {
		ac.SecretToken = secretToken
	}
	return ac, nil
}

// Mountpoint configures the target mountpoint
func Mountpoint(mountpoint string) func(*Config) {
	return func(cfg *Config) {
		cfg.mountpoint = mountpoint
	}
}

// Target url target option for Config
func Target(target string) func(*Config) {
	return func(cfg *Config) {
		if u, err := url.Parse(target); err == nil {
			cfg.target = u

			if len(u.Path) > 1 {
				parts := strings.Split(u.Path[1:], "/")
				if len(parts) >= 0 {
					cfg.bucket = parts[0]
				}
				if len(parts) >= 1 {
					cfg.basePath = path.Join(parts[1:]...)
				}
			}
		}
	}
}

// CacheDir - cache directory path option for Config
func CacheDir(path string) func(*Config) {
	return func(cfg *Config) {
		cfg.cache = path
	}
}

// SetGID - sets a custom gid for the mount.
func SetGID(gid uint32) func(*Config) {
	return func(cfg *Config) {
		cfg.gid = gid
	}
}

// SetUID - sets a custom uid for the mount.
func SetUID(uid uint32) func(*Config) {
	return func(cfg *Config) {
		cfg.uid = uid
	}
}

// Insecure - enable insecure mode.
func Insecure() func(*Config) {
	return func(cfg *Config) {
		cfg.insecure = true
	}
}

// Debug - enables debug logging.
func Debug() func(*Config) {
	return func(cfg *Config) {
		cfg.debug = true
	}
}

// Validates the config for sane values.
func (cfg *Config) validate() error {
	// check if mountpoint exists
	if cfg.mountpoint == "" {
		return errors.New("Mountpoint not set")
	}

	if cfg.target == nil {
		return errors.New("Target not set")
	}

	if cfg.bucket == "" {
		return errors.New("Bucket not set")
	}

	return nil
}
