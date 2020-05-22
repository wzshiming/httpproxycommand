package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wzshiming/commandproxy"
	"github.com/wzshiming/httpproxycommand"
)

const defaults = `httpproxycommand will start an http proxy and add HTTP_PROXY and HTTPS_PROXY to environ. 
Execute the following command. proxycommand is specified by the first parameter like ssh ProxyCommand.
Usage: httpproxycommand 'proxycommand %%h:%%p' command ...
`

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, defaults)
		flag.PrintDefaults()
		return
	}

	proxyArg := os.Args[1]
	command := os.Args[2:]
	proxy, err := commandproxy.SplitCommand(proxyArg)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(os.Stderr, defaults)
		flag.PrintDefaults()
		return
	}

	log.Printf("Run command %q", strings.Join(command, " "))
	err = httpproxycommand.ProxyCommand(context.Background(), proxy, command)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(os.Stderr, defaults)
		flag.PrintDefaults()
	}
}
