package config

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2/core"
)

type DeployToProdType string

func (d *DeployToProdType) WriteAnswer(field string, v interface{}) error {
	a, ok := v.(core.OptionAnswer)
	if !ok {
		return fmt.Errorf("invalid non-answer value")
	}

	dt := DeployToProdType(a.Value)
	(*d) = dt
	return nil
}

const (
	DeployToProdFalse DeployToProdType = "false"
	DeployToProdAuto  DeployToProdType = "auto"
	DeployToOps       DeployToProdType = "ops"
)

type Bento struct {
	Name        string `json:"name"`
	Cluster     string `json:"cluster"`
	Channel     string `json:"channel"`
	Environment string `json:"environment"`
	Passed      string `json:"passed"`
	Next        string `json:"next"`
}

type Channel struct {
	// Name of the channel
	Name string `json:"channel"`

	// PromoteToNextCron is the cron used to promote to then next channel
	PromoteToNextCron string

	// Next Channel after this one
	Next string `json:"next"`

	// Bentos are the bentos in this channel
	Bentos []Bento
}

// Note: Ordering matters
var channels = []Channel{
	// Used by things that don't have channels
	{Name: "default"},

	// 🌈 R a i n b o w 🌈
	{Name: "white", PromoteToNextCron: "0 20 * * 1-5", Next: "orange"},
	{Name: "orange", PromoteToNextCron: "30 19 * * 1-5", Next: "yellow"},
	{Name: "yellow", PromoteToNextCron: "0 19 * * 1-5", Next: "green"},
	{Name: "green"},
}

//nolint:gochecknoglobals // TODO: Cleanup globals.
var bentos = []Bento{
	{Name: "staging1a", Cluster: "staging.us-east-2", Channel: "white", Environment: "staging", Next: "app1d"},
	{Name: "app1d", Cluster: "production.us-west-2", Channel: "orange", Environment: "production", Passed: "staging1a", Next: "app1b"},
	{Name: "app1b", Cluster: "production.us-west-2", Channel: "yellow", Environment: "production", Passed: "app1d", Next: "app1e"},
	{Name: "app1e", Cluster: "production.us-west-2", Channel: "green", Environment: "production", Passed: "app1b", Next: "app1a"},
	{Name: "app1a", Cluster: "production.us-west-2", Channel: "green", Environment: "production", Passed: "app1e", Next: "app1c"},
	{Name: "app1c", Cluster: "production.us-west-2", Channel: "green", Environment: "production", Passed: "app1a", Next: "app1f"},
	{Name: "app1f", Cluster: "production.us-west-2", Channel: "green", Environment: "production", Passed: "app1c", Next: "app2a"},
	{Name: "app2a", Cluster: "production.us-east-1", Channel: "green", Environment: "production", Passed: "app1f", Next: "app2b"},
	{Name: "app2b", Cluster: "production.us-east-1", Channel: "green", Environment: "production", Passed: "app2a", Next: "app2c"},
	{Name: "app2c", Cluster: "production.us-east-1", Channel: "green", Environment: "production", Passed: "app2b"},
}

// Potentially in the future we can define multiple ops bentos, one for prod
// one for staging?
var opsBentos = []Bento{
	{Name: "ops", Cluster: "ops.us-west-2", Channel: "default", Environment: "ops"},
}

// GetBentosByConfig returns a slice of Bento objects based on the arguments passed in
func GetBentosByConfig(deployType DeployToProdType) []Bento {
	var bentoList []Bento

	switch deployType { //nolint:exhaustive
	case DeployToProdFalse:
		bentoList = getStagingBentos()
	case DeployToOps:
		bentoList = opsBentos
	default:
		bentoList = getAllBentos()
	}

	// Ensure the last bento always has next empty
	if len(bentoList) != 0 {
		bentoList[len(bentoList)-1].Next = ""
	}

	return bentoList
}

// GetBentosByChannel returns a hash map of channels
// with the value of the hash map being an array of bentos that
// belong to this channel.
func GetBentosByChannel(bentos []Bento) []Channel {
	roChans := make([]Channel, len(channels))
	copy(roChans, channels)

	channelHM := map[string]*Channel{}
	for i := range roChans {
		c := &roChans[i]
		channelHM[c.Name] = c
	}

	for _, b := range bentos {
		channel := b.Channel

		if _, ok := channelHM[channel]; !ok {
			panic(fmt.Errorf("unknown channel %s", channel))
		}

		channelHM[channel].Bentos = append(channelHM[channel].Bentos, b)
	}

	newChannels := make([]Channel, 0)
	for _, c := range roChans {
		hashedChannel := *channelHM[c.Name]
		if len(hashedChannel.Bentos) != 0 {
			newChannels = append(newChannels, hashedChannel)
		}
	}

	return newChannels
}

func getStagingBentos() []Bento {
	var stagingBentos []Bento

	for i := range bentos {
		if bentos[i].Environment == "staging" {
			stagingBentos = append(stagingBentos, bentos[i])
		}
	}
	return stagingBentos
}

func getAllBentos() []Bento {
	return bentos
}
