//go:generate go run github.com/99designs/gqlgen generate

package graph

import (
	"guarddev/awsmiddleware"
	"guarddev/database/postgres"
	"guarddev/logger"
	"guarddev/modelapi"
	"guarddev/paymentsmiddleware"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Database      *postgres.Database
	Logger        *logger.LogMiddleware
	AWSMiddleware *awsmiddleware.AWSMiddleware
	Gemini        modelapi.ModelAPI
	Payments      *paymentsmiddleware.Payments
}
