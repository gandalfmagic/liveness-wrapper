package internal

const (
	RootName = "liveness-wrapper"
	RootDescriptionShort = "An executable wrapper with readiness/liveness endpoints"
	RootDescriptionLong  = `liveness-wrapper a tool to wrap another executable and generate
the readiness and liveness http endpoints needed by kubernetes.`

	ConfigurationFile = ".liveness-wrapper"
)
