package main

import (
	"flag"
	"fmt"
	"github.com/wpnpeiris/authz-gateway/internal/server"

	"os"
)

var usageStr = `
Usage: authz-gateway [options]

Server Options:
    --listen <host:port>             HTTP bind address for authz-gateway (default: 0.0.0.0:8080)

Logging Options:
    --log.format <format>            Log output format: logfmt or json (default: logfmt)
    --log.level <level>              Log level: debug, info, warn, error (default: info)

HTTP Server Timeout Options:
    --http.read-timeout <duration>   HTTP server read timeout (default: 10s)
    --http.write-timeout <duration>  HTTP server write timeout (default: 10s)
    --http.idle-timeout <duration>   HTTP server idle timeout (default: 60s)
    --http.read-header-timeout <dur> HTTP server read header timeout (default: 5s)

Common Options:
    -h, --help                       Show this message
    -v, --version                    Show version
`

// Version is set at build time via -ldflags.
var Version string

// usage will print out the flag options of authz-gateway.
func usage() {
	fmt.Printf("%s\n", usageStr)
	os.Exit(0)
}

// printVersionAndExit will print our version and exit.
func printVersionAndExit() {
	fmt.Printf("authz-gateway: v%s\n", Version)
	os.Exit(0)
}

func main() {
	fs := flag.NewFlagSet("authz-gateway", flag.ExitOnError)
	fs.Usage = usage
	opts, err := server.ConfigureOptions(fs, os.Args[1:], printVersionAndExit, fs.Usage)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	gateway, err := server.NewGatewayServer(opts)
	if err != nil {
		server.LogAndExit(err.Error())
	}

	err = gateway.Start()
	if err != nil {
		server.LogAndExit(err.Error())
	}
}
