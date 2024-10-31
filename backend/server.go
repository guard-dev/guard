package main

import (
	"context"
	"guarddev/auth"
	"guarddev/awsmiddleware"
	"guarddev/database/postgres"
	"guarddev/graph"
	"guarddev/logger"
	"guarddev/modelapi/geminiapi"
	"guarddev/paymentsmiddleware"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/hyperdxio/opentelemetry-logs-go/exporters/otlp/otlplogs"
	sdk "github.com/hyperdxio/opentelemetry-logs-go/sdk/logs"
	"github.com/hyperdxio/otel-config-go/otelconfig"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	godotenv.Load()
	production := os.Getenv("PRODUCTION") != ""

	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("Error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()
	ctx := context.Background()

	logExporter, _ := otlplogs.NewExporter(ctx)
	loggerProvider := sdk.NewLoggerProvider(sdk.WithBatcher(logExporter))
	defer loggerProvider.Shutdown(ctx)

	logMiddleware := logger.Connect(logger.LoggerConnectProps{Production: false})
	postgresClient := postgres.Connect(ctx, postgres.DatabaseConnectProps{Logger: logMiddleware})

	gemininClient := geminiapi.Connect(ctx, geminiapi.GeminiConnectProps{
		Logger: logMiddleware,
	})

	Logger := logMiddleware.Logger(ctx)

	payments := paymentsmiddleware.Connect(paymentsmiddleware.PaymentsConnectProps{
		Logger:   logMiddleware,
		Database: postgresClient,
	})

	awsMiddleware := awsmiddleware.Connect(logMiddleware, gemininClient)

	srv := graph.Connnect(ctx, graph.GraphConnectProps{
		Logger:        logMiddleware,
		Database:      postgresClient,
		AWSMiddleware: awsMiddleware,
		Gemini:        gemininClient,
		Payments:      payments,
	})

	// Start listening for connections
	graphRouter := getGraphqlSrv()
	graphRouter.Handle("/", srv)

	if production == false {
		graphRouter.Handle("/playground", playground.Handler("GraphQL Playground", "/graph"))
		Logger.Info("[Graph] Connect to http://localhost:" + port + "/graph for GraphQL server")
		Logger.Info("[Graph] Connect to http://localhost:" + port + "/graph/playground for GraphQL playground")
	} else {
		Logger.Info("[Graph] Connect to https://api.guard.dev/graph for GraphQL server")
	}

	router := chi.NewRouter()
	router.Mount("/graph", graphRouter)
	router.Post("/payments", payments.HandleStripeWebhook)

	log.Fatal(http.ListenAndServe(":"+port, router))
}

func getGraphqlSrv() *chi.Mux {
	frontendAllowedOrigins := []string{
		"http://localhost:3000",
		"https://www.guard.dev",
		"https://guard.dev",
	}

	graphRouter := chi.NewRouter()
	graphRouter.Use(cors.New(cors.Options{
		AllowedOrigins:   frontendAllowedOrigins,
		AllowCredentials: true,
		AllowedHeaders:   []string{"Content-Type", "Authorization", "vercel_env"},
		Debug:            false,
	}).Handler)
	graphRouter.Use(auth.Middleware())

	return graphRouter
}
