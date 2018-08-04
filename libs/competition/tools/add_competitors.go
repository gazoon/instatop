package main

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"instarate/libs/competition"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		panic("You must provide path to the file with links.")
	}
	filePath := os.Args[1]
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(errors.Wrap(err, "can't read links file"))
	}
	links := strings.Split(string(fileContent), "\n")
	api := competition.InitCompetition()
	ctx := context.Background()
	for _, link := range links {
		_, err := api.Add(ctx, link)
		if err == nil {
			continue
		}
		log.WithFields(log.Fields{"link": link, "reason": err}).
			Warn("Competitor wasn't added. Skip.")
	}
	log.Info("Done!")
}
