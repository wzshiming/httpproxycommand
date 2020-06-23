package httpproxycommand

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/wzshiming/commandproxy"
	"github.com/wzshiming/httpproxy"
)

func ProxyServer(proxy []string) *httptest.Server {
	dp := commandproxy.DialProxyCommand(proxy)
	s := httptest.NewServer(&httpproxy.ProxyHandler{
		ProxyDial: func(ctx context.Context, network string, address string) (net.Conn, error) {
			log.Printf("Connect to %s with %q", address, strings.Join(proxy, " "))
			return dp.DialContext(ctx, network, address)
		},
		NotFound: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			http.Error(rw, fmt.Sprintf("Proxy with %q", strings.Join(proxy, " ")), http.StatusNotFound)
		}),
	})
	return s
}

func ProxyCommand(ctx context.Context, proxy []string, command []string) error {
	s := ProxyServer(proxy)
	defer s.Close()
	log.Printf("Run command %q", strings.Join(command, " "))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	env := append(os.Environ(), fmt.Sprintf("HTTP_PROXY=%s", s.URL), fmt.Sprintf("HTTPS_PROXY=%s", s.URL))
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigs := make(chan os.Signal)
	defer close(sigs)
	go func() {
		signal.Notify(sigs)
		defer signal.Stop(sigs)
		for sig := range sigs {
			cmd.Process.Signal(sig)
		}
	}()
	return cmd.Run()
}
