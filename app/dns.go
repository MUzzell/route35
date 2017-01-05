package main

import (
    "log"
    "net"
    "strings"
    "time"

    "fmt"

    "encoding/json"

    "github.com/miekg/dns"
)

var TypeMap = map[uint16]string{
    1: "A",
    28: "AAAA",
}

func GetQType(code uint16) string {
    i, exists := TypeMap[code]
    if exists {
        return i
    }
    return "??"
}

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

// MustGetAddress returns the IPv4 address for an interface or panics
func MustGetAddress(interfaceName string) net.IP {
    iface, err := net.InterfaceByName(interfaceName)

    if err == nil {
        if addrs, err := iface.Addrs(); err == nil {
            for _, addr := range addrs {
                var ip net.IP
                switch v := addr.(type) {
                case *net.IPNet:
                    ip = v.IP
                case *net.IPAddr:
                    ip = v.IP
                }
                if ip.To4() != nil {
                    return ip
                }
            }
        } else {
            panic(err)
        }
    }
    panic(err)
}

// MustRR returns a dns.RR from a template or panics
func MustRR(template string) dns.RR {
    value, err := dns.NewRR(template)
    if err != nil {
        panic(err)
    }
    return value
}

// WriteError puts a server failure message on the response
func WriteError(response dns.ResponseWriter, request *dns.Msg) {
    message := &dns.Msg{}
    message.SetReply(request)
    message.Compress = false
    message.RecursionAvailable = true
    message.SetRcode(request, dns.RcodeServerFailure)
    response.WriteMsg(message)
}

func (config *Config) ValidateQuery(client string, request *dns.Msg) (bool, error) {

    for _, question := range request.Question {
        for _, blocked := range config.Blocks {
            // TODO; work with regex
            if strings.Contains(question.Name, blocked) {
                return false, nil
            }
        }
    }
    return true, nil

}

// RecurseHandler creates a handler that will query the next responding Nameserver
func (config *Config) RecurseHandler(response dns.ResponseWriter, request *dns.Msg) {
    client := response.RemoteAddr().String()

    valid, err := config.ValidateQuery(client, request)

    if err != nil {
        log.Printf("RecurseHandler: error validating query; %v", err)
        WriteError(response, request)
    }

    if !valid {
        log.Printf("RecurseHandler: query blocked")
        WriteError(response, request)
    }

    for _, nameserver := range config.Nameservers {
        c := &dns.Client{Net: string(nameserver.Transport), Timeout: time.Duration(nameserver.Timeout)}
        var r *dns.Msg
        var err error

        r, _, err = c.Exchange(request, nameserver.Address)
        if err == nil || err == dns.ErrTruncated {
            r.Compress = false

            if err := response.WriteMsg(r); err != nil {
                log.Printf("RecurseHandler: failed to respond: %v", err)
                return
            }
            for _, question := range request.Question {
                log.Printf("%s (%s? %s) => %s", client, GetQType(question.Qtype), question.Name, nameserver.Address)
            }
            return
        }
        log.Printf("RecurseHandler: recurse failed: %v", err)
    }

    // If all resolvers fail, return a SERVFAIL message
    log.Printf("RecurseHandler: all resolvers failed for %v from client %s (%s)",
        request.Question, response.RemoteAddr().String(), response.RemoteAddr().Network())

    WriteError(response, request)
}

// RequestHandler returns a function that will look up entries in a Config
func (config *Config) RequestHandler(response dns.ResponseWriter, request *dns.Msg) {
    message := new(dns.Msg)

    var answers []dns.RR
    var unknown []dns.Question

    for _, question := range request.Question {

        key := strings.TrimSuffix(question.Name, fmt.Sprintf(".%s", config.Name))

        record, exists := config.Records[key]

        if exists {
            answers = append(answers, MustRR(fmt.Sprintf("%s 5 IN A %s", question.Name, record)))
        } else {
            unknown = append(unknown, question)
        }

        /*
        record := config.Records[key]
        if record != nil {
            answers = append(answers, MustRR(fmt.Sprintf("%s %d IN A %s", question.Name, record.TTL, record.Address)))
        } else {
            unknown = append(unknown, question)
        }
        */
    }

    if len(unknown) > 0 {
        log.Printf("Failed to resolve: %q, recursing.", unknown)

        answers = append(answers, config.Resolve(unknown)...)
    }

    message.Answer = answers

    message.Ns = []dns.RR{
        MustRR(fmt.Sprintf("%s 3600 IN NS %s.", config.Name, config.ListenHost)),
    }

    message.Authoritative = true
    message.RecursionAvailable = true
    message.SetReply(request)

    response.WriteMsg(message)
}

// UnmarshalJSON parses a string into a time.Duration
func (e *Duration) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    }
    duration, err := time.ParseDuration(s)
    if err != nil {
        return err
    }
    *e = Duration(duration)
    return nil
}

// Client creates a DNS client to a nameserver
func (nameserver *Nameserver) Client() *dns.Client {
    return &dns.Client{
        Net:     string(nameserver.Transport),
        Timeout: time.Duration(nameserver.Timeout),
    }
}

// Resolve a list of questions
func (config *Config) Resolve(questions []dns.Question) []dns.RR {
    targets := make(map[string]dns.Question)
    var answers []dns.RR

    for _, question := range questions {
        targets[question.Name] = question
    }

    for _, nameserver := range config.Nameservers {
        if len(targets) == 0 {
            break
        }

        var unknown []dns.Question

        for _, question := range targets {
            unknown = append(unknown, question)
        }

        r := new(dns.Msg)
        r.Question = questions

        c := nameserver.Client()

        r, _, err := c.Exchange(r, nameserver.Address)
        if err == nil || err == dns.ErrTruncated {
            answers = append(answers, r.Answer...)

            for _, answer := range r.Answer {
                delete(targets, answer.Header().Name)
            }
        } else {
            log.Printf("DNS resolve failed: %v", err)
        }
    }
    return answers
}

func MustStartDns(config *Config) {
    ip := fmt.Sprintf("%s:%d", config.ListenHost, config.Port)

    log.Println(fmt.Sprintf("DNS on %s", ip))

    for _, protocol := range []string{"udp", "tcp"} {
        go func(server *dns.Server) {
            if err := server.ListenAndServe(); err != nil {
                log.Fatalln(err)
            }
            log.Fatalln("DNS server crashed")
        }(&dns.Server{Addr: ip, Net: protocol})
    }

    dns.HandleFunc(config.Name, config.RequestHandler)
    dns.HandleFunc(".", config.RecurseHandler)
}
