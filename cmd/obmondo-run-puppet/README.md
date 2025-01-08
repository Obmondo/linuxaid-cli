# Readme

## For local testing

* Copy over the puppet cert or generate a new one from puppet server

```sh
make run_puppet
PUPPETCERT=./cert.pem PUPPETPRIVKEY=./priv.pem ./run_puppet
```

## ToDo list

### Puppet agent state

The idea behind the puppet agent state was to get info by calling the API
and do certain actions based on the state.

Like if a server is decommissioned, it'll mark the particular sever's entry
as deleted in the DB. Now, when the API is called, it'll look for the server
and find the server is deleted, and the same will be passed as response.
So, based on this, we can do different operations.

As of now, the API only returns a bool value, which is hard-coded to true.
So, we only using it to run puppet agent in noop. However, we need to update the API
to check for server's status, and accordingly send a proper response.
And in the puppet code, we'll do various operations based on the status.

### Shifting the codebase

This repository was created to store all custom scripts which we needded.
Over the time, we ended up creating lots of these, and in diffrent languages,
and it's becoming difficult to manage and keep track of scripts.

We planning to move the Go scripts present here to go-scripts repository.
This will help us to reduce redunandacy of creating same packages/functions.

### Prometheus metrics

We want the prometheus alerts to fire/notify about puppet not run for customer servers
after 2 days. This is to ensure we don't get bomabarded with several alerts frequently.
The `/opt/puppetlabs/puppet/cache/state/last_run_report.yaml` stores the puppet last run details
which can be used to trigger the alert.
