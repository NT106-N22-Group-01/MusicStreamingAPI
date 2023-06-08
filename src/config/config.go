// Package config is resposible for finding, parsing and merging the HTTPMS user
// configuration  with the default. Configuration locations should be different
// depending on the host OS.
//
// Linux/BSD configurations should be in $HOME/.euterpe/config.json
// Windows configurations should be in %APPDATA%/euterpe/config.json
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"time"

	"NT106/Group01/MusicStreamingAPI/src/helpers"

	"github.com/spf13/afero"
)

const (
	// configName contains the name of the default configuration file name. This is the
	// one the user is supposed to change.
	configName = "config.json"

	defaultlistAddress = "localhost:9996"
	defaultSecretBytes = 64
)

var configFileName string

func init() {
	flag.StringVar(&configFileName, "config", configName,
		"Name of the configuration file. If the value is a base file name and this\n"+
			"file does not exist then it would be created in the user's Euterpe\n"+
			"if present or absolute then it would be used as is.")
}

// defaultConfig contains all the default values for the Euterpe configuration. Users
// can overwrite values here with their user's configuration.
var defaultConfig = Config{
	Listen:         defaultlistAddress,
	SqliteDatabase: "musicstreaming.db",
}

// Config contains representation for everything in config.json
type Config struct {
	Listen           string      `json:"listen,omitempty"`
	Auth             bool        `json:"basic_authenticate,omitempty"`
	Authenticate     Auth        `json:"authentication,omitempty"`
	Libraries        []string    `json:"libraries,omitempty"`
	LibraryScan      ScanSection `json:"library_scan,omitempty"`
	SqliteDatabase   string      `json:"sqlite_database,omitempty"`
	DiscogsAuthToken string      `json:"discogs_auth_token,omitempty"`
}

// ScanSection is used for merging the two configs. Its purpose is to essentially
// hold the default values for its properties.
type ScanSection struct {
	Disable           bool          `json:"disable,omitempty"`
	FilesPerOperation int64         `json:"files_per_operation,omitempty"`
	SleepPerOperation time.Duration `json:"sleep_after_operation,omitempty"`
	InitialWait       time.Duration `json:"initial_wait_duration,omitempty"`
}

// Auth represents a configuration HTTP Basic authentication
type Auth struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Secret   string `json:"secret"`
}

// FindAndParse actually finds the configuration file, parsing it and merging it on
// top the default configuration.
func FindAndParse(appfs afero.Fs) (Config, error) {
	if !userConfigExists(appfs) {
		err := copyDefaultOverUser(appfs)
		if err != nil {
			return Config{}, err
		}
	}

	cfg := defaultConfig
	userCfgPath := UserConfigPath(appfs)

	fh, err := appfs.Open(userCfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("opening config: %s", err)
	}
	defer fh.Close()

	dec := json.NewDecoder(fh)

	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decoding config: %s", err)
	}

	return cfg, nil
}

// UserConfigPath returns the full path to the place where the user's configuration
// file should be
func UserConfigPath(appfs afero.Fs) string {
	if st, err := appfs.Stat(configFileName); err == nil && !st.IsDir() {
		if absPath, err := filepath.Abs(configFileName); err == nil {
			return absPath
		}
	}

	path, err := helpers.ProjectUserPath(appfs)
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, configFileName)
}

// userConfigExists returns true if the user configuration is present and in order.
// Otherwise false.
func userConfigExists(appfs afero.Fs) bool {
	path := UserConfigPath(appfs)
	st, err := appfs.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// copyDefaultOverUser will create (or replace if neccessery) the user configuration
// using the default config new config.
func copyDefaultOverUser(appfs afero.Fs) error {
	var homeDir = "~"
	user, err := user.Current()
	if err == nil {
		homeDir = user.HomeDir
	}

	randBuff := make([]byte, defaultSecretBytes)
	if _, err := rand.Read(randBuff); err != nil {
		return fmt.Errorf("creating random secret: %s", err)
	}

	userCfg := Config{
		Listen: defaultlistAddress,
		Libraries: []string{
			filepath.Join(homeDir, "Music"),
		},
		Authenticate: Auth{
			Secret: hex.EncodeToString(randBuff),
		},
	}

	userCfgPath := UserConfigPath(appfs)
	fh, err := appfs.Create(userCfgPath)
	if err != nil {
		return fmt.Errorf("create config `%s`: %s", userCfgPath, err)
	}
	defer fh.Close()

	enc := json.NewEncoder(fh)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&userCfg); err != nil {
		return fmt.Errorf("encoding default config `%s`: %s", userCfgPath, err)
	}

	return nil
}
