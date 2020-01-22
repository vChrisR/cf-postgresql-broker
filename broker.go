package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pivotal-cf/brokerapi"
	"github.com/vchrisr/cf-postgresql-broker/pgp"
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
	pgp      *pgp.PGP
	services []brokerapi.Service
}

// Services implements brokerapi.ServiceBroker
func (sb *serviceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return sb.services, nil
}

// Provision implements brokerapi.ServiceBroker
func (sb *serviceBroker) Provision(ctx context.Context, instanceID string, _ brokerapi.ProvisionDetails, _ bool) (brokerapi.ProvisionedServiceSpec, error) {
	dbname, err := sb.pgp.CreateDB(ctx, instanceID)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	return brokerapi.ProvisionedServiceSpec{
		IsAsync:      false,
		DashboardURL: dbname,
	}, nil
}

func (sb *serviceBroker) GetInstance(ctx context.Context, instanceID string) (brokerapi.GetInstanceDetailsSpec, error) {
	return brokerapi.GetInstanceDetailsSpec{}, fmt.Errorf("Function not implemented")
}

// Deprovision implements brokerapi.ServiceBroker
func (sb *serviceBroker) Deprovision(ctx context.Context, instanceID string, _ brokerapi.DeprovisionDetails, _ bool) (brokerapi.DeprovisionServiceSpec, error) {
	return brokerapi.DeprovisionServiceSpec{}, sb.pgp.DropDB(ctx, instanceID)
}

// Bind implements brokerapi.ServiceBroker
func (sb *serviceBroker) Bind(ctx context.Context, instanceID, bindingID string, _ brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
	creds, err := sb.pgp.CreateUser(ctx, instanceID, bindingID)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	return brokerapi.Binding{
		Credentials: creds,
	}, nil
}

func (sb *serviceBroker) GetBinding(ctx context.Context, instanceID, bindingID string) (brokerapi.GetBindingSpec, error) {
	return brokerapi.GetBindingSpec{}, fmt.Errorf("Function not implemented")
}

// Unbind implements brokerapi.ServiceBroker
func (sb *serviceBroker) Unbind(ctx context.Context, instanceID, bindingID string, _ brokerapi.UnbindDetails, asyncAllowed bool) (brokerapi.UnbindSpec, error) {
	return brokerapi.UnbindSpec{IsAsync: false, OperationData: ""}, sb.pgp.DropUser(ctx, instanceID, bindingID)
}

// LastOperation implements brokerapi.ServiceBroker
func (sb *serviceBroker) LastOperation(ctx context.Context, instanceID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("async operations are not supported")
}

func (sb *serviceBroker) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, fmt.Errorf("function not implemented")
}

// Update implements brokerapi.ServiceBroker
func (sb *serviceBroker) Update(ctx context.Context, instanceID string, _ brokerapi.UpdateDetails, _ bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, errors.New("updates are not supported")
}
