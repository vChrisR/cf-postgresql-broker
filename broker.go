package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Altoros/cf-postgresql-broker/pgp"
	"github.com/pivotal-cf/brokerapi"
)

// NewServiceBroker creates a new brokerapi.ServiceBroker entity
func NewServiceBroker(source, servicesJSON, GUID string) (brokerapi.ServiceBroker, error) {
	conn, err := pgp.New(source)
	if err != nil {
		return nil, err
	}

	// parse services list
	services := make([]brokerapi.Service, 0)
	if err := json.Unmarshal([]byte(servicesJSON), &services); err != nil {
		return nil, err
	}

	// replace func
	replace := func(str string) string {
		return strings.Replace(str, "{GUID}", GUID, 1)
	}

	// replace GUID with its actual value
	for i := 0; i < len(services); i++ {
		services[i].ID = replace(services[i].ID)
		for j := 0; j < len(services[i].Plans); j++ {
			services[i].Plans[j].ID = replace(services[i].Plans[j].ID)
		}
	}

	return &serviceBroker{pgp: conn, services: services}, nil
}

// serviceBroker implements brokerapi.ServiceBroker
type serviceBroker struct {
	pgp      *pgp.db
	services []brokerapi.Service
}

// Services implements brokerapi.ServiceBroker
func (sb *serviceBroker) Services(ctx context.Context) []brokerapi.Service {
	return sb.services
}

// Provision implements brokerapi.ServiceBroker
func (sb *serviceBroker) Provision(ctx context.Context, instanceID string, _ brokerapi.ProvisionDetails, _ bool) (brokerapi.ProvisionedServiceSpec, error) {
	dbname, err := sb.pgp.CreateDB(instanceID)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	return brokerapi.ProvisionedServiceSpec{
		IsAsync:      false,
		DashboardURL: dbname,
	}, nil
}

// Deprovision implements brokerapi.ServiceBroker
func (sb *serviceBroker) Deprovision(ctx context.Context, instanceID string, _ brokerapi.DeprovisionDetails, _ bool) (brokerapi.DeprovisionServiceSpec, error) {
	return brokerapi.DeprovisionServiceSpec{}, sb.pgp.DropDB(instanceID)
}

// Bind implements brokerapi.ServiceBroker
func (sb *serviceBroker) Bind(ctx context.Context, instanceID, bindingID string, _ brokerapi.BindDetails) (brokerapi.Binding, error) {
	creds, err := sb.pgp.CreateUser(instanceID, bindingID)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	return brokerapi.Binding{
		Credentials: creds,
	}, nil
}

// Unbind implements brokerapi.ServiceBroker
func (sb *serviceBroker) Unbind(ctx context.Context, instanceID, bindingID string, _ brokerapi.UnbindDetails) error {
	return sb.pgp.DropUser(instanceID, bindingID)
}

// LastOperation implements brokerapi.ServiceBroker
func (sb *serviceBroker) LastOperation(ctx context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("async operaions are not supported")
}

// Update implements brokerapi.ServiceBroker
func (sb *serviceBroker) Update(ctx context.Context, instanceID string, _ brokerapi.UpdateDetails, _ bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, errors.New("updates are not supported")
}
