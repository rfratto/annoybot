package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/nlopes/slack"
)

type annoySubject struct {
	// Name is the target name of the user or channel to annoy.
	Name string

	// If true, anyone typing in the channel provided by Name will
	// be annoyed. Otherwise, only annoys the username provided by
	// Name in a DM with them.
	Channel bool
}

func getTargetUser(rtm *slack.RTM, username string) (annoySubject, error) {
	users, err := rtm.GetUsers()
	if err != nil {
		return annoySubject{}, err
	}

	for _, user := range users {
		if user.Name == username {
			return annoySubject{Name: user.ID}, nil
		}
	}

	return annoySubject{}, fmt.Errorf("user not found")
}

func getTargetChannel(rtm *slack.RTM, channel string) (annoySubject, error) {
	channels, err := rtm.GetChannels(true)
	if err != nil {
		return annoySubject{}, err
	}

	for _, ch := range channels {
		if ch.Name == channel {
			return annoySubject{Name: channel, Channel: true}, nil
		}
	}

	return annoySubject{}, fmt.Errorf("channel not found")
}

func main() {
	apiKey, hasKey := os.LookupEnv("SLACK_API_KEY")
	if !hasKey {
		fmt.Fprintln(os.Stderr, "SLACK_API_KEY not found in environment")
		return
	} else if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Pass a username to annoy as first argument to program")
		return
	}

	rand.Seed(time.Now().Unix())

	api := slack.New(apiKey)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	var target annoySubject
	var err error

	targetUsername := os.Args[1]
	if len(targetUsername) > 0 && targetUsername[0] == '#' {
		target, err = getTargetChannel(rtm, targetUsername[1:])
	} else {
		target, err = getTargetUser(rtm, targetUsername)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "could not find %v\n", targetUsername)
		return
	}

	fmt.Printf("found %v (channel=%v), waiting for user to type...\n", targetUsername, target.Channel)

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.UserTypingEvent:
			ch, _ := rtm.GetConversationInfo(ev.Channel, false)
			if target.Channel != ch.IsChannel || ch == nil {
				continue
			}

			if !target.Channel && ev.User != target.Name {
				// Wrong user
				continue
			} else if target.Channel && ch.Name != target.Name {
				// Wrong channel
				continue
			}

			fmt.Printf("%v is typing, annoying them now...\n", targetUsername)
			rtm.SendMessage(rtm.NewTypingMessage(ev.Channel))
		}
	}
}
