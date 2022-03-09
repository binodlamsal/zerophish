package config

import (
	"encoding/json"
	"io/ioutil"
)

// AdminServer represents the Admin server configuration details
type AdminServer struct {
	ListenURL string `json:"listen_url"`
	UseTLS    bool   `json:"use_tls"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
}

// PhishServer represents the Phish server configuration details
type PhishServer struct {
	ListenURL string `json:"listen_url"`
	UseTLS    bool   `json:"use_tls"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
}

// LoggingConfig represents configuration details for Gophish logging.
type LoggingConfig struct {
	Filename string `json:"filename"`
}

// Config represents the configuration information.
type Config struct {
	AdminConf           AdminServer   `json:"admin_server"`
	PhishConf           PhishServer   `json:"phish_server"`
	DBName              string        `json:"db_name"`
	DBPath              string        `json:"db_path"`
	MigrationsPath      string        `json:"migrations_prefix"`
	TestFlag            bool          `json:"test_flag"`
	ContactAddress      string        `json:"contact_address"`
	Logging             LoggingConfig `json:"logging"`
	SsoKey              string        `json:"sso_key"`
	ViaProxy            string        `json:"via_proxy"`
	Production          string        `json:"production"`
	UserSyncApiUser     string        `json:"usersync_api_user"`
	UserSyncApiPassword string        `json:"usersync_api_password"`
	ByPassSso           string        `json:"bypass_sso"`
}

// Conf contains the initialized configuration struct
var Conf Config

// Version contains the current gophish version
var Version = ""

// ServerName is the server type that is returned in the transparency response.
const ServerName = "gophish"

// LoadConfig loads the configuration from the specified filepath
func LoadConfig(filepath string) error {
	// Get the config file
	configFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configFile, &Conf)
	if err != nil {
		return err
	}
	// Choosing the migrations directory based on the database used.
	Conf.MigrationsPath = Conf.MigrationsPath + Conf.DBName
	// Explicitly set the TestFlag to false to prevent config.json overrides
	Conf.TestFlag = false
	return nil
}
