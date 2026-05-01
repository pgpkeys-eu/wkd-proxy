package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"wkd-proxy/server"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

var (
	configFile = flag.String("config", "", "config file")
	logLevel   = flag.String("log", "", "log level")
)

var Sigmap = map[os.Signal]func(){}

// Init handles common command line flags, logging, profiling etc. for all CLI commands.
// The caller MUST import "flag" and call flag.Parse() before calling Init().
func Init(isServer bool) (settings *server.Settings) {
	if configFile != nil {
		conf, err := os.ReadFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration file '%s'.\n", *configFile)
			Die(errors.WithStack(err))
		}
		settings, err = server.ParseSettings(string(conf))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing configuration file '%s'.\n", *configFile)
			Die(errors.WithStack(err))
		}
	}

	if *logLevel != "" {
		settings.LogLevel = *logLevel
	}
	if !isServer {
		level, err := log.ParseLevel(strings.ToLower(settings.LogLevel))
		if err != nil {
			log.Warningf("invalid LogLevel=%q: %v", settings.LogLevel, err)
		} else {
			log.SetLevel(level)
		}
	}
	return
}

func HandleSignals() {
	c := make(chan os.Signal, 1)
	keys := make([]os.Signal, len(Sigmap))
	i := 0
	for k := range Sigmap {
		keys[i] = k
		i++
	}
	signal.Notify(c, keys...)
	go func() {
		// BEWARE: go-staticcheck will suggest that you replace the following with `for range`.
		// This is not how signal handling works (it is SUPPOSED to loop forever).
		// Please DO NOT change this function unless you can explain how it works. :-)
		for {
			select {
			case sig := <-c:
				if Sigmap[sig] != nil {
					Sigmap[sig]()
				}
			}
		}
	}()
}

// Die prints the error and exits with a non-zero exit code
func Die(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
