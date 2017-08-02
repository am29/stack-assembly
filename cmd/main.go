// Package main provides cmd claws application
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/molecule-man/claws/claws"
	"github.com/molecule-man/claws/cli"
	"github.com/molecule-man/claws/cloudprov/awsprov"
)

func main() {
	tpl := flag.String("tpl", "", "CF tpl")
	cfgFile := flag.String("cfg", "", "CF tpl")
	stackName := flag.String("stack", "", "Stack name")

	flag.Parse()

	cfg := readConfigs(cfgFile)

	if tpl != nil && *tpl != "" {

		cfg.Templates = map[string]claws.TemplateConfig{
			"tpl": {
				Path: *tpl,
				Name: stackName,
			},
		}
	}

	cp := awsprov.New()

	serv := claws.Service{
		Approver:      &cli.Approval{},
		Log:           log.New(os.Stderr, "", log.LstdFlags),
		CloudProvider: &cp,
	}

	for _, template := range cfg.Templates {
		tplBody, err := ioutil.ReadFile(template.Path)

		if err != nil {
			log.Fatal(err)
		}

		err = serv.Sync(claws.StackTemplate{
			StackName: *template.Name,
			Body:      string(tplBody),
			Params:    cfg.Parameters,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func readConfigs(cfgFiles *string) claws.Config {
	mainConfig := claws.Config{}

	if _, err := os.Stat("Claws.toml"); err == nil {
		mainConfig = readConfig("Claws.toml")
	}

	if cfgFiles == nil || *cfgFiles == "" {
		return mainConfig
	}

	for _, cf := range strings.Split(*cfgFiles, " ") {
		mainConfig.Merge(readConfig(cf))
	}

	return mainConfig
}

func readConfig(f string) claws.Config {
	cfg := claws.Config{}
	_, err := toml.DecodeFile(f, &cfg)

	if err != nil {
		log.Fatal(err)
	}

	return cfg
}
