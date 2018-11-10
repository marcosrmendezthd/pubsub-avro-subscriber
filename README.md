# pubsub-avro-subscriber

## Development 
 
 * [go](https://golang.org/dl) 1.11+
 * [modules](https://github.com/golang/go/wiki/Modules)
 * [goimports](http://godoc.org/golang.org/x/tools/cmd/goimports)
 
# Getting Started

     git config --global core.autocrlf input
     git config --global push.default simple
     
     cd $GOPATH/src
         
     mkdir -p github.com/marcosrmendezthd  
           
     cd github.com/marcosrmendezthd            
     
     git clone git@github.com:marcosrmendezthd/pubsub-avro-subscriber.git    
         
     cd pubsub-avro-subscriber
     
     script/cibuild
     
# Dependency Management

We're using [go modules](https://github.com/golang/go/wiki/Modules) and go 1.11+, so please follow the steps
below to add a new dependency in the siphon root directory.

    GO111MODULE=on go get -u <dependency>
    GO111MODULE=on go mod tidy
    GO111MODULE=on go mod verify
    GO111MODULE=on go mod vendor

After the dependecy is added, please run `script/cibuild` to ensure everything works as expected.
     
# Build

Run `./script/cibuild` to cross-compile to MacOS, Windows and Linux.

# Usage

    usage: pubsub-avro-subscriber --gcp-project=GCP-PROJECT --pubsub-topic=PUBSUB-TOPIC --pubsub-subscription=PUBSUB-SUBSCRIPTION [<flags>]
    
    GCP Subscriber
    
    Flags:
      --help                       Show context-sensitive help (also try --help-long and --help-man).
      --debug                      Enables debug logging
      --gcp-key-file=GCP-KEY-FILE  Google Cloud Key File
      --gcp-key=GCP-KEY            Base64-encoded Google Cloud Key
      --gcp-project=GCP-PROJECT    Pub Sub Project
      --pubsub-topic=PUBSUB-TOPIC  Pub Sub Topic
      --pubsub-subscription=PUBSUB-SUBSCRIPTION  
                                   Pub Sub Subscription
      --all                        Reads all messages in a subscription
      --ack                        Acknowledge messages as they are received
      --version                    Show application version.

