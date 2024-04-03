package db

import "os"

// Config specifies the configurations to connect to the database.
type Config struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Database        string `yaml:"database"`
	PasswordEnvName string `yaml:"passwordEnvName"`
}

// password returns the password for the connection to the database.
func (c Config) password() string {
	return os.Getenv(c.PasswordEnvName)
}
