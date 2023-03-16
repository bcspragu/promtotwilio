# bcspragu/promtotwilio

This is a fork of swatto/promtotwillio. I mainly forked it to update the Go
version, but then I hung around and changed a few other things:

* Dependency changes
  * Removed `fasthttp` for the standard library's `net/http`
  * Removed `jsonparser` for the standard library's `encoding/json`
  * Removed `logrus` for the standard library's `log`
  * Swapped out `carlosdp/twiliogo` for the official `twilio/twilio-go` client
* Removed the `?receiver=<number>` functionality
  * I imagine most `promtotwilio` instances will be deployed on private networks, but it's a dangerous attack vector for those caught unawares.
  * One way to re-implement this more safely would be a configurable list of destinations based on req params
* Updated the Dockerfile

As a general caveat: sending text message alerts isn't a great alerting
strategy for a production system, where you'd probably want something more
robust and full-featured. I use this for a home cluster where I really just
want to know if the basement is about to catch fire.

Original README below.

# Prometheus alert with text message

This is a simple and stupid program that will receive webhooks from [Prometheus](https://prometheus.io/) to send them as text message (using [Twilio](https://www.twilio.com/)) with the summary of the alert.

The [Docker image](https://hub.docker.com/r/swatto/promtotwilio/) size is less than 9MB.

![Docker Pulls](https://img.shields.io/docker/pulls/swatto/promtotwilio.svg?style=flat-square)

## Configuration

It needs 4 environment variables:

- `SID` - Twilio Account SID
- `TOKEN` - Twilio Auth Token
- `RECEIVER` - Phone number of receiver (optional parameter, representing default receiver)
- `SENDER` - Phone number managed by Twilio (friendly name)

You can see a basic launch inside the Makefile.

## API

`/`: ping promtotwilio application. Returns 200 OK if application works fine.

`/send?receiver=<rcv>`: send Prometheus firing alerts from payload to a rcv if specified, or to default receiver, represented by RECEIVER environment variable. If none is specified, status code 400 BadRequest is returned.

## Test it

To send test sms to a phone +zxxxyyyyyyy use the following command (please notice `%2B` symbols, representing a url encoded `+` sign)

```bash
$ curl -H "Content-Type: application/json" -X POST -d \
'{"version":"2","status":"firing","alerts":[{"annotations":{"summary":"Server down"},"startsAt":"2016-03-19T05:54:01Z"}]}' \
http://localhost:9090/send?receiver=%2Bzxxxyyyyyyy
```

## Configuration example

Here's a sample Docker Compose file to use it with [cAdvisor](https://github.com/google/cadvisor), [Prometheus](http://prometheus.io/), [Alertmanager](https://github.com/prometheus/alertmanager) and [Grafana](https://github.com/grafana/grafana).

```yml
sms:
  image: swatto/promtotwilio:latest
  environment:
    SID: xxx
    TOKEN: xxx
    RECEIVER: xxx
    SENDER: xxx

alert:
  image: prom/alertmanager:latest
  links:
   - sms
  volumes:
   - ./alertmanager.yml:/etc/alertmanager/config.yml

container:
  image: google/cadvisor:latest
  volumes:
   - /:/rootfs:ro
   - /var/run:/var/run:rw
   - /sys:/sys:ro
   - /var/lib/docker/:/var/lib/docker:ro

prometheus:
  image: prom/prometheus:latest
  links:
   - container
   - alert
  volumes:
   - ./prometheus.yml:/etc/prometheus/prometheus.yml
   - ./alerts.conf:/etc/prometheus/alerts.conf
  entrypoint: /bin/prometheus -config.file=/etc/prometheus/prometheus.yml -alertmanager.url=http://alert:9093

web:
  image: grafana/grafana:latest
  links:
   - prometheus
  ports:
   - "3000:3000"
  environment:
    GF_SERVER_ROOT_URL: http://stats.example.com
    GF_SECURITY_ADMIN_PASSWORD: 123456
```

Here's the AlertManager config where `sms` will be provided by Docker Compose

```yml
route:
  receiver: 'admin'

receivers:
- name: 'admin'
  webhook_configs:
  - url: 'http://sms:9090/send'
```
