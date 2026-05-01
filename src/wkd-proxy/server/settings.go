package server

import (
	"bytes"
	"os"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
)

type HTTPConfig struct {
	Bind              string `toml:"bind"`
	LogRequestDetails bool   `toml:"logRequestDetails"`
}

type HTTPSConfig struct {
	Bind              string `toml:"bind"`
	LogRequestDetails bool   `toml:"logRequestDetails"`
	Cert              string `toml:"cert"`
	Key               string `toml:"key"`
}

type Settings struct {
	Keyserver string `toml:"keyserver"`

	LogFile  string `toml:"logfile"`
	LogLevel string `toml:"loglevel"`

	Hostname string `toml:"hostname"`

	// HTTPSConfig is a pointer so it can default to nil
	HTTP  HTTPConfig   `toml:"http"`
	HTTPS *HTTPSConfig `toml:"https"`

	Software string
	Version  string
	BuiltAt  string
}

func DefaultSettings() *Settings {
	return &Settings{
		LogLevel: "INFO",
		Hostname: "localhost",
		Software: "wkd-proxy",
		Version:  "~unreleased",
		HTTP: HTTPConfig{
			Bind: ":8080",
		},
	}
}

func ParseSettings(data string) (*Settings, error) {
	// Parse the configuration file as a template first
	tmpl, err := template.New("config").Funcs(sprig.TxtFuncMap()).Funcs(envFuncMap()).Parse(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Initialize a writer to render the template
	w := &bytes.Buffer{}

	// Render the template
	err = tmpl.Execute(w, readEnv())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	doc := DefaultSettings()
	_, err = toml.Decode(w.String(), &doc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return doc, nil
}

// EnvFuncMap returns a map of functions that can be used in a template
func envFuncMap() template.FuncMap {
	return template.FuncMap(
		map[string]interface{}{
			"osenv": func(prefix string) map[string]string {
				env := make(map[string]string)
				for _, e := range os.Environ() {
					pair := strings.SplitN(e, "=", 2)
					// if the environment variable starts with the prefix, add it to the map
					if strings.HasPrefix(pair[0], prefix) {
						env[pair[0]] = pair[1]
					}
				}
				return env
			},
		},
	)
}

// ReadEnv returns a map of environment variables
func readEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		env[pair[0]] = pair[1]
	}
	return env
}
