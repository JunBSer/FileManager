package http

type Config struct {
	Host string `env:"HTTP_HOST" envDefault:"localhost"`
	Port int    `env:"HTTP_PORT" envDefault:"8080"`
}
