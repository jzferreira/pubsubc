package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
)

var (
	debug   = flag.Bool("debug", false, "Enable debug logging")
	help    = flag.Bool("help", false, "Display usage information")
	version = flag.Bool("version", false, "Display version information")
)

// The CommitHash and Revision variables are set during building.
var (
	CommitHash = "<not set>"
	Revision   = "<not set>"
)

// Topics describes a PubSub topic and its subscriptions.
type Topics map[string][]string

func versionString() string {
	return fmt.Sprintf("pubsubc - build %s (%s) running on %s", Revision, CommitHash, runtime.Version())
}

// debugf prints debugging information.
func debugf(format string, params ...interface{}) {
	if *debug {
		fmt.Printf(format+"\n", params...)
	}
}

// fatalf prints an error to stderr and exits.
func fatalf(format string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, os.Args[0]+": "+format+"\n", params...)
	os.Exit(1)
}

// create a connection to the PubSub service and create topics and subscriptions
// for the specified project ID.
func create(ctx context.Context, projectID string, topics Topics) error {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Unable to create client to project %q: %s", projectID, err)
	}
	defer client.Close()

	debugf("Client connected with project ID %q", projectID)

	for topicID, subscriptions := range topics {
		debugf("  Creating topic %q", topicID)
		topic, err := client.CreateTopic(ctx, topicID)
		if err != nil {
			return fmt.Errorf("Unable to create topic %q for project %q: %s", topicID, projectID, err)
		}

		for _, subscription := range subscriptions {
			var (
				endpoint string
				pushConfig pubsub.PushConfig
				subConfig pubsub.SubscriptionConfig
			)

			subConfig = pubsub.SubscriptionConfig{
				Topic: topic,
			}

			//p := strings.Split(subscription, "@")
			//subscriptionId := p[0]
			//debugf("    Creating subscription %q", subscriptionId)
			//if p[1] != "" {
			//	endpoint = p[1]
			//	pushConfig = pubsub.PushConfig{
			//		Endpoint: endpoint,
			//	}
			//	subConfig = pubsub.SubscriptionConfig{
			//		Topic: topic,
			//		PushConfig: pushConfig,
			//	}
			//} else{
			//	subConfig = pubsub.SubscriptionConfig{
			//		Topic: topic,
			//	}
			//}

			_, err = client.CreateSubscription(ctx, subscriptionId, subConfig)
			if err != nil {
				return fmt.Errorf("Unable to create subscription %q on topic %q for project %q: %s", subscriptionId, topicID, projectID, err)
			}

			go func(){
				doEvery(5*time.Second, receiveMessage, projectID,  subscriptionId)
			}()
		}
	}

	return nil
}

func main() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Printf(`Usage: env PUBSUB_PROJECT1="project1,topic1,topic2>subscription1" %s`+"\n", os.Args[0])
		flag.PrintDefaults()
	}

	if *help {
		flag.Usage()
		return
	}

	if *version {
		fmt.Println(versionString())
		return
	}

	// Cycle over the numbered PUBSUB_PROJECT environment variables.
	for i := 1; ; i++ {
		// Fetch the enviroment variable. If it doesn't exist, break out.
		currentEnv := fmt.Sprintf("PUBSUB_PROJECT%d", i)
		env := os.Getenv(currentEnv)
		if env == "" {
			// If this is the first environment variable, print the usage info.
			if i == 1 {
				flag.Usage()
				os.Exit(1)
			}
			break
		}
		fmt.Printf(`Project config found:`+"\n", currentEnv)
		topics, projectId := ParseEnv(env)

		// Create the project and all its topics and subscriptions.
		if err := create(context.Background(), projectId , topics); err != nil {
			fatalf(err.Error())
		}
	}

}

func receiveMessage(projectId string, subscriptionId string){
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		log.Fatal(err.Error())
	}
	sub := client.Subscription(subscriptionId)
	err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		log.Printf("Got message: %s", m.Data)
		// NOTE: May be called concurrently; synchronize access to shared memory.
		m.Ack()
	})
}

func doEvery(d time.Duration, f func(projectId string, subscriptionId string), projectId string, subscriptionId string) {
	for _ = range time.Tick(d) {
		f(projectId, subscriptionId)
	}
}

func ParseEnv(env string) (Topics, string) {
	// Separate the projectID from the topic definitions.
	parts := strings.Split(env, ",")
	if len(parts) < 2 {
		fatalf("%s: Expected at least 1 topic to be defined")
	}

	projectId := parts[0]

	// Separate the topicID from the subscription IDs.
	topics := make(Topics)
	for _, part := range parts[1:] {
		topicParts := strings.Split(part, ">")
		topicName := topicParts[0]
		topicSubscriptions := topicParts[1:]
		fmt.Printf(`Topic declaration foun found:`+"\n", topicName)
		fmt.Printf(`With subscription declaration:`+"\n", topicSubscriptions)
		topics[topicName] = topicSubscriptions
	}
	return topics, projectId
}
