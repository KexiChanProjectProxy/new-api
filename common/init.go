package common

import (
	"flag"
	"fmt"
	"os"
)

var (
	PrintVersion = flag.Bool("version", false, "print version and exit")
	PrintHelp    = flag.Bool("help", false, "print help and exit")
	ConfigFile   = flag.String("config", "", "path to JSON configuration file")
)

var LogDir string
var Port *int

func PrintHelpMessage() {
	fmt.Println("NewAPI(Based OneAPI) " + Version + " - The next-generation LLM gateway and AI asset management system supports multiple languages.")
	fmt.Println("Original Project: OneAPI by JustSong - https://github.com/songquanpeng/one-api")
	fmt.Println("Maintainer: QuantumNous - https://github.com/QuantumNous/new-api")
	fmt.Println("Usage: newapi --config <config.json> [--version] [--help]")
}

func printHelp() {
	PrintHelpMessage()
}

func ParseFlags() {
	flag.Parse()
}

func InitEnv() {
	if *PrintVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *PrintHelp {
		printHelp()
		os.Exit(0)
	}
}