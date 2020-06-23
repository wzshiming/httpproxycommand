package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wzshiming/commandproxy"
	"github.com/wzshiming/httpproxycommand"
	"github.com/wzshiming/notify"
)

const defaults = `httpproxycommand will start an http proxy and add HTTP_PROXY and HTTPS_PROXY to environ. 
Execute the following command. proxycommand is specified by the first parameter like ssh ProxyCommand.
Usage: 
	httpproxycommand 'proxycommand %h:%p' command ...
	HTTP_PROXY=$(httpproxycommand 'proxycommand %h:%p') HTTPS_PROXY=$(httpproxycommand 'proxycommand %h:%p') command ...

Example:
	httpproxycommand 'nc %h %p' curl http://example.org/
	HTTP_PROXY=$(httpproxycommand 'nc %h %p') HTTPS_PROXY=$(httpproxycommand 'nc %h %p') curl http://example.org/
`

var (
	homeDir, _ = os.UserHomeDir()
	prefix     = filepath.Join(homeDir, ".httpproxycommand")
	ctx        = context.Background()
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Println(defaults)
		flag.PrintDefaults()
		return
	}

	if len(args) < 3 {
		proxuUrl, err := getProxyServer(prefix, args[1], true)
		if err != nil {
			log.Println(err)
			log.Println(defaults)
			flag.PrintDefaults()
			return
		}
		if proxuUrl != "" {
			log.Printf("Proxy server %s", proxuUrl)
			fmt.Println(proxuUrl)
			os.Stdout.Close()
		}
		return
	}

	if args[2] == "-" {
		proxuUrl, err := getProxyServer(prefix, args[1], false)
		if err != nil {
			log.Println(err)
			log.Println(defaults)
			flag.PrintDefaults()
			return
		}
		if proxuUrl != "" {
			log.Printf("Proxy server %s", proxuUrl)
			fmt.Println(proxuUrl)
			os.Stdout.Close()
		}
		<-make(chan struct{})
		return
	}

	proxyArg := args[1]
	command := args[2:]
	proxy, err := commandproxy.SplitCommand(proxyArg)
	if err != nil {
		log.Println(err)
		log.Println(defaults)
		flag.PrintDefaults()
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	notify.Once(os.Interrupt, cancel)
	err = httpproxycommand.ProxyCommand(ctx, proxy, command)
	if err != nil {
		log.Fatal(err)
	}
}

func background() error {
	cmd := exec.Command(os.Args[0], append(os.Args[1:], "-")...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Start()
}

func getProxyServer(prefix, proxy string, bg bool) (string, error) {
	args, err := commandproxy.SplitCommand(proxy)
	if err != nil {
		return "", err
	}

	sum := md5.Sum([]byte(strings.Join(args, " ")))
	h := hex.EncodeToString(sum[:]) + ".txt"

	proxyname := filepath.Join(prefix, h)
	file, err := ioutil.ReadFile(proxyname)
	if err != nil {
		return startServer(proxyname, args, bg)
	}
	proxyUrl := string(file)
	resp, err := http.Head(proxyUrl)
	if err != nil {
		return startServer(proxyname, args, bg)
	}
	if resp.StatusCode != http.StatusNotFound {
		return startServer(proxyname, args, bg)
	}
	return proxyUrl, nil
}

func startServer(proxyname string, args []string, bg bool) (string, error) {
	if bg {
		return "", background()
	}
	url, _, err := httpproxycommand.ProxyServer(args)
	if err != nil {
		return "", err
	}
	os.MkdirAll(filepath.Dir(proxyname), 0755)
	err = ioutil.WriteFile(proxyname, []byte(url), 0755)
	if err != nil {
		return "", err
	}
	return url, nil
}
