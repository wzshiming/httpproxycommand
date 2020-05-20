package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"

	"github.com/wzshiming/commandproxy"
	"github.com/wzshiming/httpproxy"
)

func main() {
	if len(os.Args) < 3 {
		flag.PrintDefaults()
		return
	}

	proxyArg := os.Args[1]
	command := os.Args[2:]
	proxy, err := commandproxy.SplitCommand(proxyArg)
	if err != nil {
		log.Println(err)
		flag.PrintDefaults()
		return
	}

	ph := &httpproxy.ProxyHandler{
		ProxyDial: func(ctx context.Context, _ string, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			m := map[byte]string{
				'h': host,
				'p': port,
			}
			p := make([]string, len(proxy))
			copy(p, proxy)
			for i := range p {
				p[i] = commandproxy.ReplaceEscape(p[i], m)
			}
			log.Printf("Connect to %s with %q", address, strings.Join(p, " "))
			cp := commandproxy.ProxyCommand(ctx, p[0], p[1:]...)
			return cp.Stdio()
		},
	}

	s := httptest.NewServer(ph)
	cmd := exec.Command(command[0], command[1:]...)
	env := append(os.Environ(), fmt.Sprintf("HTTP_PROXY=%s", s.URL), fmt.Sprintf("HTTPS_PROXY=%s", s.URL))
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("Run command %q", strings.Join(command, " "))
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
