package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	negronilogrus "github.com/meatballhat/negroni-logrus"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/henrywallace/mdpreview/server"
)

var (
	addr        = flag.String("addr", "", "address to serve preview like :8080 or 0.0.0.0:7000")
	api         = flag.Bool("api", false, "whether to render via the Github API")
	debug       = flag.Bool("debug", false, "debug logging")
	listener, _ = net.Listen("tcp", ":0")
	definePort  bool
)

func main() {
	flag.Parse()

	log := logrus.New()
	if *debug {
		log.SetLevel(logrus.DebugLevel)
	}

	if len(os.Args) < 2 {
		err := fmt.Errorf("path must be given")
		checkErr(err)
	}

	path := os.Args[1]
	if filepath.Ext(path) != ".md" {
		log.Warnf("path %s doesn't look like a Markdown file", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := fmt.Errorf("path %s does not exist", path)
		checkErr(err)
	}

	s, err := server.New(path, log, !*api)
	checkErr(err)

	h, err := s.Run()
	checkErr(err)

	n := negroni.New()
	n.Use(negroni.NewRecovery())
	n.Use(negronilogrus.NewMiddlewareFromLogger(log, "web"))
	n.UseHandler(h)

	if strings.HasPrefix(*addr, ":") {
		*addr = fmt.Sprintf("http://localhost%s", *addr)
		log.Info(fmt.Sprintf("Starting mdpreview server at %s", *addr))
		definePort = true
	} else {
		listener, err = net.Listen("tcp", ":0")
		checkErr(err)
		*addr = "http://localhost:" + fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
		log.Info(fmt.Sprintf("Starting mdpreview server at %s", *addr))
	}

	openbrowser(*addr)

	if definePort {
		err = http.ListenAndServe(*addr, n)
		checkErr(err)
	} else {
		err = http.Serve(listener, n)
		checkErr(err)
	}
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Fatal(err)
	}
}

func checkErr(err error) {
	if err != nil {
		// notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		pc, fn, line, _ := runtime.Caller(1)
		log.Panicf("[error] in %s[%s:%d] %v", runtime.FuncForPC(pc).Name(), fn, line, err)
	}
}
