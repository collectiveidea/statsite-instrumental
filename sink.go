package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	AuthFormat = "hello version 1.0\nauthenticate %s\n"

	DefaultHost    = "collector.instrumentalapp.com"
	DefaultPort    = 8000
	DefaultTimeout = time.Minute

	DefaultStatsitePort = 8125
)

var (
	// Mapping of key prefix given by statsite to the action that should be passed
	// to Instrumental.
	Actions = map[string]string{
		"timers":    "gauge_absolute",
		"sets":      "gauge_absolute",
		"gauges":    "gauge",
		"counts":    "increment",
		"histogram": "increment",
	}

	AuthenticationFailed = errors.New("Failed to authenticate with token")
	Config               = new(config)
)

type config struct {
	Host         string
	Port         int
	StatsitePort int
	Token        string
	Prefix       string
	Postfix      string
	Timeout      time.Duration
}

func (c *config) HostWithPort() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func (c *config) StatsiteHostWithPort() string {
	return ":" + strconv.Itoa(c.StatsitePort)
}

func configureFromFlags() {
	flag.StringVar(&Config.Host, "host", DefaultHost, "agent host")
	flag.IntVar(&Config.Port, "port", DefaultPort, "agent port")
	flag.DurationVar(&Config.Timeout, "timeout", DefaultTimeout, "agent timeout")

	flag.IntVar(&Config.StatsitePort, "statsite_port", DefaultStatsitePort, "statsite feedback port (0 to disable)")

	flag.StringVar(&Config.Prefix, "prefix", "", "Prepended to all keys (useful to end with a .)")
	flag.StringVar(&Config.Postfix, "postfix", "", "Appended to all keys (useful to start with a .)")
	flag.Parse()

	flag.Usage = func() {
		fmt.Printf("Usage: %s [token]\n", os.Args[0])
		flag.PrintDefaults()
	}

	Config.Token = flag.Arg(0)

	if Config.Token == "" {
		fmt.Print("Missing authentication token\n\n")
		flag.Usage()
		os.Exit(1)
	}

	return
}

// Determine the action based on the key
func expandKey(original string) (key, action string) {
	parts := strings.SplitN(original, ".", 2)

	kind := parts[0]
	key = parts[1]

	// Histogram data should be incremented
	if strings.Index(key, "histogram.bin") != -1 {
		kind = "histogram"
	}

	return key, Actions[kind]
}

func funnel(input io.Reader, output io.Writer) (err error) {
	scanner := bufio.NewScanner(input)
	gauges := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 3)

		key, action := expandKey(parts[0])

		if len(parts) == 3 && action != "" {
			value := parts[1]
			timestamp := parts[2]

			if action == "gauge" {
				gauges = append(gauges, fmt.Sprintf("%s:%s|g", key, value))
			}

			params := []interface{}{
				action,
				Config.Prefix,
				key,
				Config.Postfix,
				value,
				timestamp,
			}

			_, err = fmt.Fprintf(output, "%s %s%s%s %s %s\n", params...)
			if err != nil {
				return
			}
		}
	}

	feedGaugesBackToStatsite(gauges)

	err = scanner.Err()

	return
}

func connect() (conn net.Conn, err error) {
	conn, err = net.DialTimeout("tcp", Config.HostWithPort(), Config.Timeout)
	if err != nil {
		return
	}

	// Timeout after 60 seconds
	conn.SetDeadline(time.Now().Add(Config.Timeout))

	// Authenticate
	if _, err = fmt.Fprintf(conn, AuthFormat, Config.Token); err != nil {
		return
	}

	data := make([]byte, 512)
	if _, err = conn.Read(data); err != nil {
		return
	}

	if string(data)[:6] != "ok\nok\n" {
		err = AuthenticationFailed
		return
	}

	return
}

// Statsite gauges are reset back to 0 after the sink flush.
// The recommended solution (aka to make statsite work like statsd) is to feed
// gauge values back into statsite to reset the value back to what it was.
//
// From https://github.com/armon/statsite/issues/69
func feedGaugesBackToStatsite(stats []string) {
	if Config.StatsitePort == 0 {
		return
	}

	conn, err := net.DialTimeout("udp", Config.StatsiteHostWithPort(), Config.Timeout)
	if err != nil {
		return
	}

	for _, stat := range stats {
		fmt.Fprintf(conn, "%s\n", stat)
	}
}

func main() {
	configureFromFlags()

	conn, err := connect()
	if err != nil {
		if err == AuthenticationFailed {
			fmt.Printf("Authentication with token %s was declined\n", Config.Token)
			os.Exit(1)
		} else {
			panic(err)
		}
	}
	defer conn.Close()

	if err := funnel(os.Stdin, conn); err != nil {
		panic(err)
	}
}
