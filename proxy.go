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

func ProxyServer(ctx context.Context, proxy []string) (string, *http.Server, error) {
	dp := commandproxy.DialProxyCommand(proxy)
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		listen, err = net.Listen("tcp", "[::1]:0")
		if err != nil {
			return "", nil, err
		}
	}
	cmd := strings.Join(proxy, " ")
	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler: &httpproxy.ProxyHandler{
			ProxyDial: func(ctx context.Context, network string, address string) (net.Conn, error) {
				log.Printf("Connect to %s with %q", address, cmd)
				return dp.DialContext(ctx, network, address)
			},
			NotFound: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				http.Error(rw, fmt.Sprintf("Proxy with %q", cmd), http.StatusNotFound)
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

func ProxyCommand(ctx context.Context, envs, proxy, command []string) error {
	url, srv, err := ProxyServer(ctx, proxy)
	if err != nil {
		return err
	}
	defer srv.Close()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = append(envs,
		fmt.Sprintf("http_proxy=%s", url),
		fmt.Sprintf("https_proxy=%s", url),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
