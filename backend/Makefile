watch:
	go run github.com/mitranim/gow -e 'go,tsx,html' run ./cmd/cli start

watch-beta:
	go run github.com/mitranim/gow -e 'go,tsx,html' run -tags prod,toolkit -ldflags "-X robinplatform.dev/internal/config.robinVersion=v0.0.0 -X robinplatform.dev/internal/config.builtReleaseChannel=beta" ./cmd/cli start
