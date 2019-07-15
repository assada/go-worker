package config

import (
	"os"
	"strings"
	"syscall"
)

const (
	DefaultLogLevel = "INFO"

	DefaultReloadSignal = syscall.SIGHUP

	DefaultKillSignal = syscall.SIGINT
)

type Config struct {
	KillSignal   *os.Signal     `mapstructure:"kill_signal"`
	LogLevel     *string        `mapstructure:"log_level"`
	PidFile      *string        `mapstructure:"pid_file"`
	ReloadSignal *os.Signal     `mapstructure:"reload_signal"`
}

func DefaultConfig() *Config {
	return &Config{
		LogLevel: stringFromEnv([]string{
			"LOG_LEVEL",
		}, DefaultLogLevel),
	}
}

func (c *Config) Finalize() {
	if c.KillSignal == nil {
		c.KillSignal = Signal(DefaultKillSignal)
	}

	if c.LogLevel == nil {
		c.LogLevel = stringFromEnv([]string{
			"LOG_LEVEL",
		}, DefaultLogLevel)
	}

	if c.PidFile == nil {
		c.PidFile = String("")
	}

	if c.ReloadSignal == nil {
		c.ReloadSignal = Signal(DefaultReloadSignal)
	}
}

func (c *Config) Copy() *Config {
	var o Config

	o.KillSignal = c.KillSignal

	o.LogLevel = c.LogLevel

	o.PidFile = c.PidFile

	o.ReloadSignal = c.ReloadSignal

	return &o
}

func (c *Config) Merge(o *Config) *Config {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.KillSignal != nil {
		r.KillSignal = o.KillSignal
	}

	if o.LogLevel != nil {
		r.LogLevel = o.LogLevel
	}

	if o.PidFile != nil {
		r.PidFile = o.PidFile
	}

	if o.ReloadSignal != nil {
		r.ReloadSignal = o.ReloadSignal
	}

	return r
}

func stringFromEnv(list []string, def string) *string {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			return String(strings.TrimSpace(v))
		}
	}
	return String(def)
}