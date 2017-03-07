package main

import (
    "regexp"
    "bufio"
    "strings"
	"log"
    "time"
    "os"
    "fmt"

    "io/ioutil"
	"encoding/json"

)

const ipv4Pattern string = `((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))`
const hostnamePattern string = `((?:(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*(?:[A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-][A-Za-z0-9])+)`
const recordEntryPattern string = "^" + ipv4Pattern + `\s+` + hostnamePattern

const DEFAULT_CONF_FILE string = "/etc/route35/config.json"

// Config contains global server configuration
type Config struct {
	Port			int
	ListenHost		string
	Name			string
	Secret			string
	DatabaseType	string
	DatabaseFile	string
	Nameservers		[]Nameserver
    BlockFile       []string
    RecordsFile     []string
    Records         map[string]string
    Blocks          []string
}

// Record contains a single DNS entry
type Record struct {
    Address string
}

// Nameserver will respond if we do not know an entry
type Nameserver struct {
    Address   string
    Timeout   Duration
    Transport Transport
}

// Transport is either the string "tcp" or "udp"
type Transport string

// UnmarshalJSON parses a transport string
func (e *Transport) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    }
    if s == "" {
        *e = "tcp"
    } else if s == "tcp" || s == "udp" {
        *e = Transport(s)
    } else {
        return fmt.Errorf("Illegal value for transport %q", s)
    }
    return nil
}

// Duration can be JSON parsed
type Duration time.Duration

// NamedRecord contains a DNS entry and its name
type NamedRecord struct {
    Record string
    Name string
}

func MustReadFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}
	return data
}

func MustWriteFile(path string, data []byte) {
    ioutil.WriteFile(path, data, 0644)
}

func MustParseConfig(path string) *Config {
	var config Config

	if err := json.Unmarshal(MustReadFile(path), &config); err != nil {
		log.Fatalln(err)
	}

    config.MustParseHostsFile()
    config.MustParseBlocksFile()

	return &config
}

func (config *Config) MustParseBlocksFile() error {

    idx := 0

    for _, blockfile := range config.BlockFile {
        file, err := os.Open(blockfile)
        if err != nil {
            log.Fatalf("Unable to open given block file: %v", err)
            continue
        }
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            line := strings.Trim(scanner.Text(), " \t")
            log.Printf("line: %s", line)
            if strings.HasPrefix(line, "#") {
                continue
            }
            block := strings.Trim(line[0:strings.Index(line, "#")], " \t")
            log.Printf("! %s", block)
            config.Blocks[idx] = block
            idx++
        }
        defer file.Close()
    }
    return nil;
}

func (config *Config) MustParseHostsFile() error {

    var recordRegex = regexp.MustCompile(recordEntryPattern)

    config.Records = make(map[string]string)

    for _, hostsfile := range config.RecordsFile {
        file, err := os.Open(hostsfile)
        if err != nil {
            log.Fatalf("Unable to open given host file: %v", err)
            continue
        }
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            line := strings.Trim(scanner.Text(), " \t")
            if strings.HasPrefix(line, "#") {
                continue
            }
            results := recordRegex.FindAllStringSubmatch(line, 1)
            if results == nil {
                continue
            }
            hostname := fmt.Sprintf("%s.", results[0][2])
            address := results[0][1]
            log.Printf("+ %s => %s", hostname, address)
            config.Records[hostname] = address
        }
        defer file.Close()

    }
    return nil;
}