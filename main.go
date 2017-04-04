package main

import (
	"encoding/json"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
)

func main() {
	name, err := appName()
	if err != nil {
		panic("appName: " + err.Error())
	}

	// set up logger
	logger := lager.NewLogger(name)
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	// create requests serviceBroker
	broker, err := NewServiceBroker(
		os.Getenv("PG_SOURCE"),
		os.Getenv("PG_SERVICES"),
		os.Getenv("CF_INSTANCE_GUID"))

	if err != nil {
		logger.Fatal("serviceBroker", err)
	}

	// register service broker handler
	http.Handle("/", brokerapi.New(broker, logger, brokerapi.BrokerCredentials{
		Username: os.Getenv("BASIC_AUTH_USERNAME"),
		Password: os.Getenv("BASIC_AUTH_PASSWORD"),
	}))

	// boot up
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("boot-up", lager.Data{"port": port})
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal("serve", err)
	}
}

// appName returns application name based on the $VCAP_APPLICATION
func appName() (string, error) {
	env := os.Getenv("VCAP_APPLICATION")
	vcap := &struct {
		ApplicationName string `json:"application_name"`
	}{}

	if err := json.Unmarshal([]byte(env), vcap); err != nil {
		return "", err
	}
	return vcap.ApplicationName, nil
}
