package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"

	"encoding/base64"
	"io/ioutil"
	"log"

	"gopkg.in/alecthomas/kingpin.v2"

	"context"
	"fmt"
	"net/http"

	avro "github.com/linkedin/goavro"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/pubsub/v1"
)

var (
	app                = kingpin.New("pubsub-avro-subscriber", "GCP Subscriber").Author("THD Engineering & Operations")
	debug              = app.Flag("debug", "Enables debug logging").Envar("DEBUG").Default("true").Bool()
	gcpKeyFile         = app.Flag("gcp-key-file", "Google Cloud Key File").Envar("GCP_KEY_FILE").File()
	gcpKey             = app.Flag("gcp-key", "Base64-encoded Google Cloud Key").Envar("GCP_KEY").String()
	gcpProject         = app.Flag("gcp-project", "Pub Sub Project").Envar("GCP_PROJECT").Required().String()
	pubsubTopic        = app.Flag("pubsub-topic", "Pub Sub Topic").Envar("PUBSUB_TOPIC").Required().String()
	pubsubSubscription = app.Flag("pubsub-subscription", "Pub Sub Subscription").Envar("PUBSUB_SUBSCRIPTION").Required().String()
	all                = app.Flag("all", "Reads all messages in a subscription").Envar("ALL").Default("false").Bool()
	ack                = app.Flag("ack", "Acknowledge messages as they are received").Envar("ACK").Default("false").Bool()
)

var (
	Version     = "0.0.1"
	GitCommit   = "HEAD"
	BuildStamp  = "UNKNOWN"
	FullVersion = Version + "+" + GitCommit + "-" + BuildStamp
)

func init() {
	app.Version(FullVersion)
}

func GetAuthClient(key []byte) (*http.Client, error) {
	conf, err := google.JWTConfigFromJSON(key, pubsub.PubsubScope, pubsub.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, http.DefaultClient)
	ts := conf.TokenSource(ctx)

	return oauth2.NewClient(ctx, ts), nil

}

func GetClient(key []byte) (*pubsub.Service, error) {
	client, authError := GetAuthClient(key)
	if authError != nil {
		return nil, authError
	}
	return pubsub.New(client)
}

func main() {

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var gcpkey []byte

	if *gcpKeyFile != nil {
		keyBytes, readKeyFileError := ioutil.ReadAll(*gcpKeyFile)
		if readKeyFileError != nil {
			log.Fatal("Error reading key file:", readKeyFileError)
		}
		gcpkey = keyBytes
	}
	if len(gcpkey) == 0 && len(*gcpKey) > 0 {
		keyBytes, decodeError := base64.StdEncoding.DecodeString(*gcpKey)
		if decodeError != nil {
			log.Fatal("Error decoding key string:", decodeError)
		}
		gcpkey = keyBytes
	}

	if len(gcpkey) == 0 {
		log.Fatal("GCP Key not defined")
	}

	service, err := GetClient(gcpkey)
	if err != nil {
		log.Fatal(err)
	}

	tbs := pubsub.NewProjectsTopicsSubscriptionsService(service)
	topicName := fmt.Sprintf("projects/%s/topics/%s", *gcpProject, *pubsubTopic)
	resp, err := tbs.List(topicName).Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("subscriptions available", resp.Subscriptions)

	pss := pubsub.NewProjectsSubscriptionsService(service)
	subscriptionName := fmt.Sprintf("projects/%s/subscriptions/%s", *gcpProject, *pubsubSubscription)
	sub, err := pss.Get(subscriptionName).Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("got subscription", sub.Name)

	log.Println("reading messages from", subscriptionName)

	var maxMessages int64

	if !*all {
		maxMessages = 1
	} else {
		maxMessages = 100
	}

	messagesReceived := 0

	for {
		presp := requestMessages(pss, subscriptionName, maxMessages)
		receiveMessages(presp)
		messagesReceived += len(presp.ReceivedMessages)
		log.Printf("-- received %d messages", messagesReceived)
		if !*all || len(presp.ReceivedMessages) == 0 {
			break
		}
	}

	log.Printf("%d messages total", messagesReceived)

}
func requestMessages(pss *pubsub.ProjectsSubscriptionsService, subscriptionName string, maxMessages int64) *pubsub.PullResponse {
	pr := &pubsub.PullRequest{
		MaxMessages:       maxMessages,
		ReturnImmediately: false,
	}
	presp, err := pss.Pull(subscriptionName, pr).Do()
	if err != nil {
		log.Fatal(err)
	}
	if *ack && len(presp.ReceivedMessages) > 0 {
		var ids []string
		for _, message := range presp.ReceivedMessages {
			ids = append(ids, message.AckId)
		}
		ackReq := &pubsub.AcknowledgeRequest{
			AckIds: ids,
		}
		ackCall := pss.Acknowledge(subscriptionName, ackReq)
		_, err := ackCall.Do()
		if err != nil {
			log.Fatalf("error on ack: %v\n", err)
		}
	}
	return presp
}

func receiveMessages(presp *pubsub.PullResponse) {
	for _, message := range presp.ReceivedMessages {
		log.Println("received", message.Message.MessageId)
		if *debug {
			log.Println("attributes", message.Message.Attributes)
			log.Println("publishTime", message.Message.PublishTime)
			log.Println("size", len(message.Message.Data))
			log.Println("data", message.Message.Data)
		}
		decodeMessage(message.Message.Data)
	}
}

func decodeMessage(data string) {
	log.Println("base64 decoding data")
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Println("base64 decoding failed", err.Error())
		return
	}
	if *debug {
		log.Println("base64 decoding successful", "bytes", len(decoded))
	}

	log.Println("gzip decompressing data")
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(decoded))
	if err != nil {

		log.Println("gzip decompression failed. trying base64 decode")
		out := make([]byte, base64.StdEncoding.DecodedLen(len(decoded)))
		_, err = base64.StdEncoding.Decode(out, []byte(decoded))
		if err != nil {
			log.Println("second base64 decoding failed", err.Error())
			return
		}
		if *debug {
			log.Println("base64 decoding successful", "bytes", len(out))
		}
		gzipReader, err = gzip.NewReader(bytes.NewBuffer(out))
	}
	defer gzipReader.Close()

	recordCount := 0
	ocfReader, err := avro.NewOCFReader(gzipReader)
	for ocfReader.Scan() {

		raw, err := ocfReader.Read()
		if err != nil {
			log.Println("unable to read ocf record", err.Error())
			continue
		}

		record, ok := raw.(map[string]interface{})
		if !ok {
			log.Println("unable to map ocf record", err.Error())
			continue
		}

		recordCount++

		if *debug {
			// pretty print map
			b, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				log.Println("unable to pretty print record", err.Error())
				fmt.Printf("%v\n", record)
			} else {
				fmt.Println(string(b))
			}
		}

	}

	log.Printf("%d records", recordCount)

}
