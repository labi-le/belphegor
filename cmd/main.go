package main

import (
	"belphegor/internal/belphegor"
	"belphegor/pkg/clipboard"
	"belphegor/pkg/encryption"
	"belphegor/pkg/ip"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"syscall"
)

var version = "main"

var (
	helpMsg  = `belphegor - ...`
	password []byte

	addressIP   string
	port        int
	secure      bool
	debug       bool
	showVersion bool
	showHelp    bool
)

func init() {
	flag.StringVar(&addressIP, "connect", "", "Address in ip:port format to connect to the node")
	flag.IntVar(&port, "port", 0, "Port to use. Default: random")
	flag.BoolVar(&secure, "secure", false, "Encrypt your data with a password")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Parse()

	initLogger(debug)
}

func main() {

	if debug {
		log.Info().Msg("Debug mode enabled")
		// set report caller to true
	}

	if secure {
		logrus.Print("Password for -secure: ")
		password, _ = terminal.ReadPassword(int(syscall.Stdin))
	}

	if showVersion {
		log.Printf("version %s", version)
	}

	if showHelp {
		log.Info().Msg(helpMsg)
		os.Exit(0)
	}

	var node *belphegor.Node
	if port != 0 {
		node = belphegor.NewNode(clipboard.NewManager(), enc(password), ip.MakeAddr(port))
	} else {
		node = belphegor.NewNodeRandomPort(clipboard.NewManager(), enc(password))
	}

	go func() {
		if err := node.Start(); err != nil {
			log.Error().Err(err).Msg("Failed to start the node")
		}
	}()

	//todo add neighbor discovery by key or short word
	if addressIP != "" {
		go func() {
			if err := node.ConnectTo(addressIP); err != nil {
				log.Fatal().Err(err).Msg("Failed to connect to the node")
			}
		}()
	}

	select {}
}

func enc(password []byte) *encryption.Cipher {
	if password == nil {
		return nil
	}

	return encryption.NewEncryption(password)
}

func initLogger(debug bool) {
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logrus.SetReportCaller(true)

		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
