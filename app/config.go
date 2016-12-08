package main

import (
	"io/ioutil"
	"log"

	"encoding/json"

)

// Config contains global server configuration
type Config struct {
	Port        int
	ListenHost	string
	Name        string
	Secret      string
	Records     map[string]*Record
	Nameservers []Nameserver
}

func MustReadFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}
	return data
}

func MustParseConfig(path string) Config {
	var config Config

	if err := json.Unmarshal(MustReadFile(path), &config); err != nil {
		log.Fatalln(err)
	}

	return config
}
