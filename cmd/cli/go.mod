module github.com/sguter90/weathermaestro/cmd/cli

go 1.25

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/sguter90/weathermaestro/pkg/database v0.1.0
	github.com/sguter90/weathermaestro/pkg/models v0.1.0
	github.com/sguter90/weathermaestro/pkg/puller v0.0.0-20260204072708-47cd9d9a8178
	github.com/sguter90/weathermaestro/pkg/pusher v0.1.0
	github.com/spf13/cobra v1.7.0
	golang.org/x/term v0.39.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.40.0 // indirect
)
