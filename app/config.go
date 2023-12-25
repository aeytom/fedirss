package app

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-yaml/yaml"
)

type Settings struct {
	AppDir   string         `yaml:"dir,omitempty" json:"app_dir,omitempty"`
	Mastodon MastodonConfig `yaml:"mastodon,omitempty" json:"mastodon,omitempty"`
	Db       struct {
		Dsn string `yaml:"dsn,omitempty" json:"dsn,omitempty"`
		Opt string `yaml:"opt,omitempty" json:"opt,omitempty"`
	} `yaml:"db,omitempty" json:"db,omitempty"`
	Feed struct {
		Url string `yaml:"url,omitempty" json:"url,omitempty"`
	} `json:"yaml,omitempty"`
}

var (
	Config Settings
)

func LoadConfig() *Settings {

	help := flag.Bool("help", false, "show command line usage")
	envPath := getEnvArg("DOT_ENV", "dotEnv", "env.yaml", "dot env path (YAML)")
	showCfg := flag.Bool("showCfg", false, "show config content")
	flag.Parse()

	if ep, err := filepath.Abs(*envPath); err != nil {
		log.Println(*envPath, err)
	} else {
		*envPath = ep
	}

	ed, err := os.ReadFile(*envPath)
	if err != nil {
		log.Println(*envPath, err)
	} else {
		err = yaml.Unmarshal([]byte(ed), &Config)
		if err != nil {
			log.Fatalln(err)
		}
	}

	if Config.Db.Dsn == "" {
		// default lookup shops.json in directory of .env.yaml
		Config.Db.Dsn = "root:just-for-testing@(mysql)/payment"
	}

	if Config.Db.Opt == "" {
		// default lookup shops.json in directory of .env.yaml
		Config.Db.Opt = "loc=Local&parseTime=true"
	}

	if Config.AppDir == "" {
		Config.AppDir = "."
	}

	Config.AppDir, err = filepath.Abs(Config.AppDir)
	if err != nil {
		log.Fatalln(err)
	}

	if *showCfg {
		showConfig()
		os.Exit(0)
	}

	if *help {
		usage()
		os.Exit(0)
	}

	return &Config
}

func getEnvArg(env string, arg string, dflt string, usage string) *string {
	ev, avail := os.LookupEnv(env)
	if avail {
		dflt = ev
	}
	v := flag.String(arg, dflt, usage)
	return v
}

func showConfig() {
	cb, _ := yaml.Marshal(Config)
	fmt.Println(string(cb))
}

func usage() {
	fmt.Println("")
	fmt.Printf("== Usage %s ==\n", os.Args[0])
	fmt.Println("")
	showConfig()
	fmt.Println("")
	fmt.Printf("Run: %s -dotEnv .env.yaml\n", os.Args[0])
	fmt.Println("")
}
