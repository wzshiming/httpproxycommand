package httpproxycommand

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/wzshiming/commandproxy"
	"github.com/wzshiming/httpproxy"
)

func ProxyServer(proxy []string) (string, *http.Server, error) {
	dp := commandproxy.DialProxyCommand(proxy)
	listen, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", nil, err
	}
	srv := &http.Server{
		Handler: &httpproxy.ProxyHandler{
			ProxyDial: func(ctx context.Context, network string, address string) (net.Conn, error) {
				log.Printf("Connect to %s with %q", address, strings.Join(proxy, " "))
				return dp.DialContext(ctx, network, address)
			},
			NotFound: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				http.Error(rw, fmt.Sprintf("Proxy with %q", strings.Join(proxy, " ")), http.StatusNotFound)
			}),
		},
	}
	go func() {
		err = srv.Serve(listen)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Serve error %s", err)
		}
	}()

	url := fmt.Sprintf("http://%s", listen.Addr())
	return url, srv, nil
}

func ProxyCommand(ctx context.Context, proxy []string, command []string) error {
	url, srv, err := ProxyServer(proxy)
	if err != nil {
		return err
	}
	defer srv.Close()
	log.Printf("Run command %q", strings.Join(command, " "))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	env := append(os.Environ(), fmt.Sprintf("HTTP_PROXY=%s", url), fmt.Sprintf("HTTPS_PROXY=%s", url))
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
