package server

import (
	"flag"
	"time"
)

// Options holds all configuration options for the authz-gateway server.
type Options struct {
	ServerListen      string
	LogFormat         string
	LogLevel          string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// ConfigureOptions parses command-line arguments and returns an Options struct.
// It handles -h/--help and -v/--version flags by calling the provided callbacks.
// Returns nil options and nil error when help or version flags are used.
func ConfigureOptions(fs *flag.FlagSet, args []string, printVersion, printHelp func()) (*Options, error) {
	opts := &Options{}
	var (
		showVersion bool
		showHelp    bool
	)
	fs.BoolVar(&showVersion, "v", false, "Print version information.")
	fs.BoolVar(&showVersion, "version", false, "Print version information.")
	fs.BoolVar(&showHelp, "h", false, "Print usage.")
	fs.BoolVar(&showHelp, "help", false, "Print usage.")

	fs.StringVar(&opts.ServerListen, "listen", "0.0.0.0:8080", "Network host:port to listen on")
	fs.StringVar(&opts.LogFormat, "log.format", "logfmt", "log output format: logfmt or json")
	fs.StringVar(&opts.LogLevel, "log.level", "info", "log level: debug, info, warn, error")
	fs.DurationVar(&opts.ReadTimeout, "http.read-timeout", 10*time.Second, "HTTP server read timeout (for large uploads)")
	fs.DurationVar(&opts.WriteTimeout, "http.write-timeout", 10*time.Second, "HTTP server write timeout (for large downloads)")
	fs.DurationVar(&opts.IdleTimeout, "http.idle-timeout", 60*time.Second, "HTTP server idle timeout")
	fs.DurationVar(&opts.ReadHeaderTimeout, "http.read-header-timeout", 5*time.Second, "HTTP server read header timeout (slowloris protection)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if showVersion {
		printVersion()
		return nil, nil
	}

	if showHelp {
		printHelp()
		return nil, nil
	}

	return opts, nil
}
