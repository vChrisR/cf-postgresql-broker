# cf-postgresql-broker

PostgreSQL single database instance service broker for the Cloud Foundry.

## Installation

Download the source code and install dependencies.

```
$ go get github.com/Altoros/cf-postgresql-broker
$ cd $GOPATH/src/github.com/Altoros/cf-postgresql-broker
```

Push the code base to the Cloud Foundry.

```
$ cf push --no-start
```

Set the environment.

* `PG_SERVICES` can be customized according to [this go library](https://github.com/pivotal-cf/brokerapi/blob/master/catalog.go#L3)
* `{GUID}` for `id` attributes will be replaced with its runtime value

```
$ AUTH_USER=admin
$ AUTH_PASSWORD=admin
$ PG_SOURCE=postgres://username:password@host:port/dbname
$ PG_SERVICES='[{
  "id": "service-postgresql-{GUID}",
  "name": "postgresql",
  "description": "DBaaS",
  "bindable": true,
  "plan_updateable": false,
  "plans": [{
    "id": "plan-basic-{GUID}",
    "name": "basic",
    "description": "Shared plan"
  }]
}]'

$ cf set-env cf-postgresql-broker AUTH_USER "$AUTH_USER"
$ cf set-env cf-postgresql-broker AUTH_PASSWORD "$AUTH_PASSWORD"
$ cf set-env cf-postgresql-broker PG_SOURCE "$PG_SOURCE"
$ cf set-env cf-postgresql-broker PG_SERVICES "$PG_SERVICES"
```

Start the application and register a service broker

```
$ cf start cf-postgresql-broker
$ BROKER_URL=$(cf app cf-postgresql-broker | grep urls: | awk '{print $2}')
$ cf create-service-broker cf-postgresql-broker $AUTH_USER $AUTH_PASSWORD http://$BROKER_URL
$ cf enable-service-access cf-postgresql-broker
```

Then you should see `cf-postgresql-broker` in `$ cf marketplace`

## Binding applications

```
$ cf create-service cf-postgresql-broker basic pgsql
$ cf bind-service my-app pgsql
$ cf restage my-app
```

## Development

1. Copy `.envrc.example` to `.envrc`, then load it by `$ source .envrc` if you don't have the [direnv](http://direnv.net) package installed.
