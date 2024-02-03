# Backoffice Cache Service
Backend cache REST api for Backoffice

Written in golang to increase RPS

All requirements you can find in `go.mod`

## Getting started

### Setup Env Vars

create `.env` file and add variables like in `.env.example`

### To run via docker

Install `Docker` and `docker-compose`

Run
```sh
$ docker-compose build
$ docker-compose up
```

### Testing
```sh
$ go test -v    
```

```sh
NewCacheKeyService

// Define a mapping of URL patterns to cache key representations.
	urlMap := map[string]string{
		"v1/bet_insights/partner-sports":           "v1_bet_insights_partner_sports",
		"v1/bet_insights/partner-widgets_settings": "v1_bet_insights_widgets_settings",
	}
```
is in the services package

if the request path does not match the map, a 404 error will be thrown

if the headers do not have a Partner-Id and Authorization, a 401 error will be thrown

all responses (except for answers with 500 statuses) are cached for 30 minutes

Django backoffice is responsible for cache invalidation

