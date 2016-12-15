package main

import (
	"io/ioutil"
	"log"

	"encoding/json"

)

// Config contains global server configuration
type Config struct {
	Port			int
	ListenHost		string
	Name			string
	Secret			string
	DatabaseType	string
	DatabaseFile	string
	Records			map[string]*Record
	Nameservers		[]Nameserver
}

// Record contains a single DNS entry
type Record struct {
    Address string
    TTL     int
}

// Nameserver will respond if we do not know an entry
type Nameserver struct {
    Address   string
    Timeout   Duration
    Transport Transport
}

// Transport is either the string "tcp" or "udp"
type Transport string

// Duration can be JSON parsed
type Duration time.Duration

// NamedRecord contains a DNS entry and its name
type NamedRecord struct {
    Record
    Name string
}

func MustReadFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}
	return data
}

func MustParseConfig(path string) *Config {
	var config Config

	if err := json.Unmarshal(MustReadFile(path), &config); err != nil {
		log.Fatalln(err)
	}

	return &config
}
