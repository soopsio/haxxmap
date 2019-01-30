package main

import (
	"./go-imap-proxy"
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/server"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
)

type ContextHook struct {
}

func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(8); ok {
		funcName := runtime.FuncForPC(pc).Name()
		entry.Data["file"] = path.Base(file)
		entry.Data["func"] = path.Base(funcName)
		entry.Data["line"] = line
	}
	return nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: proxy imap.example.com:993 cert.pem cert.key")
		return
	}

	// https://gist.github.com/spikebike/2232102#file-server-go
	cert, err := tls.LoadX509KeyPair(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("server: loadkeys: %s", err)
	}
	tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	pTlsConfig := tls.Config{InsecureSkipVerify: true}
	be := proxy.NewTLS(os.Args[1], &pTlsConfig)

	// Create a new server
	s := server.New(be)
	logger := logrus.New()
	logger.AddHook(ContextHook{})
	//logger.ReportCaller = true
	logger.Level = logrus.TraceLevel
	s.ErrorLog = logger
	s.TLSConfig = &tlsConfig
	s.Addr = ":993"
	//s.Debug = os.Stdout

	rf, wf, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	s.Debug = wf
	bf := bufio.NewReader(rf)
	imapRead := NewReader(bf)

	go func() {
		i := 0
		p := regexp.MustCompile("\\d+")
		_ = p
		for {
			log.Println("------------------------------------------------")
			fields, err := imapRead.ReadLine()
			if err != nil {
				if imap.IsParseError(err) {
					log.Println("err:", err)
				} else {
					log.Println("cannot read command:", err)
				}
			} else {
				cmd := &imap.Command{}
				if err := cmd.Parse(fields); err != nil {
					log.Println("err:", err)
				} else {
					log.Println("Name::", cmd.Name)
					log.Println("Tag:", cmd.Tag)

					log.Printf("Arguments: [%d]\n", len(cmd.Arguments))
					// TODO: 逆向 github.com/emersion/go-imap/message.go 的 BodySectionName 结构，精细化 debug
				}
			}
			i++
			log.Println(i)
		}
	}()

	log.Printf("Starting IMAP TLS server at :993 proxying to %s\n", os.Args[1])
	if err := s.ListenAndServeTLS(); err != nil {
		log.Fatal(err)
	}
}

// ParseNamedResp attempts to parse a named data response.
func ParseNamedResp(resp interface{}) (name string, fields []interface{}, ok bool) {
	data, ok := resp.(*imap.DataResp)
	if !ok || len(data.Fields) == 0 {
		return
	}

	// Some responses (namely EXISTS and RECENT) are formatted like so:
	//   [num] [name] [...]
	// Which is fucking stupid. But we handle that here by checking if the
	// response name is a number and then rearranging it.
	if len(data.Fields) > 1 {
		name, ok := data.Fields[1].(string)
		if ok {
			if _, err := ParseNumber(data.Fields[0]); err == nil {
				fields := []interface{}{data.Fields[0]}
				fields = append(fields, data.Fields[2:]...)
				return strings.ToUpper(name), fields, true
			}
		}
	}

	// IMAP commands are formatted like this:
	//   [name] [...]
	name, ok = data.Fields[0].(string)
	if !ok {
		return
	}
	return strings.ToUpper(name), data.Fields[1:], true
}
