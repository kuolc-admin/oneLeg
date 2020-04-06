package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(".env not found")
		return
	}
}

type Environment = string

const (
	EnvironmentLocal       = "local"
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

func Env() Environment {
	return Environment(os.Getenv("ENV"))
}

func IsLocal() bool {
	return Env() == EnvironmentLocal
}

func IsDevelopment() bool {
	return Env() == EnvironmentDevelopment
}

func IsProduction() bool {
	return Env() == EnvironmentProduction
}

func TLSCertificatePath() string {
	return os.Getenv("TLS_CERTIFICATE_PATH")
}

func TLSPrivateKeyPath() string {
	return os.Getenv("TLS_PRIVATE_KEY_PATH")
}
