package main

import (
	"belphegor/internal/belphegor"
	"belphegor/pkg/clipboard"
	"belphegor/pkg/encryption"
	"belphegor/pkg/ip"
	"flag"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
)

var version = "main"

var (
	helpMsg  = `belphegor - ...`
	password []byte

	addr        string
	port        int
	secure      bool
	debug       bool
	showVersion bool
	help        bool
)

func init() {
	flag.StringVar(&addr, "connect", "", "address to connect to (client mode)")
	flag.IntVar(&port, "port", 0, "port to use (server mode)")
	flag.BoolVar(&secure, "secure", false, "encrypt your data with a password")
	flag.BoolVar(&debug, "debug", false, "print debug logs")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.BoolVar(&help, "help", false, "print help")

	flag.Parse()

}

func main() {

	if debug {
		belphegor.Logger().Debug("Debug mode enabled")
		belphegor.Logger().SetReportCaller(true)
		belphegor.Logger().SetLevel(logrus.DebugLevel)
	}

	if secure {
		logrus.Print("Password for -secure: ")
		password, _ = terminal.ReadPassword(int(syscall.Stdin))
	}

	if showVersion {
		belphegor.Logger().Infof("version %s", version)
	}

	if help {
		belphegor.Logger().Info(helpMsg)
		belphegor.Logger().Exit(0)
		return
	}

	var node *belphegor.Node
	if port != 0 {
		node = belphegor.NewNode(clipboard.NewManager(), enc(password), ip.MakeAddr(port))
	} else {
		node = belphegor.NewNodeRandomPort(clipboard.NewManager(), enc(password))
	}

	go node.Start()

	//todo add neighbor discovery by key or short word
	if addr != "" {
		go node.ConnectTo(addr)
	}

	select {}
}

func enc(password []byte) *encryption.Cipher {
	if password == nil {
		return nil
	}

	return encryption.NewEncryption(password)
}
