package httpproxycommand

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/wzshiming/commandproxy"
	"github.com/wzshiming/httpproxy"
)

type DialProxyCommand []string

func (p *DialProxyCommand) DialContext(ctx context.Context, _ string, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	m := map[byte]string{
		'h': host,
		'p': port,
	}
	proxy := make([]string, len(*p))
	copy(proxy, *p)
	for i := range proxy {
		proxy[i] = commandproxy.ReplaceEscape(proxy[i], m)
	}
	log.Printf("Connect to %s with %q", address, strings.Join(proxy, " "))
	cp := commandproxy.ProxyCommand(ctx, proxy[0], proxy[1:]...)
	return cp.Stdio()
}

func ProxyCommand(ctx context.Context, proxy []string, command []string) error {
	dp := DialProxyCommand(proxy)
	s := httptest.NewServer(&httpproxy.ProxyHandler{
		ProxyDial: dp.DialContext,
	})
	defer s.Close()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	env := append(os.Environ(), fmt.Sprintf("HTTP_PROXY=%s", s.URL), fmt.Sprintf("HTTPS_PROXY=%s", s.URL))
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigs := make(chan os.Signal)
	defer close(sigs)
	signal.Notify(sigs)
	defer signal.Stop(sigs)
	go func() {
		for sig := range sigs {
			cmd.Process.Signal(sig)
		}
	}()
	return cmd.Run()
}
