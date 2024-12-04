package main

import (
	"context"
	"guarddev/auth"
	"guarddev/awsmiddleware"
	"guarddev/database/postgres"
	"guarddev/graph"
	"guarddev/logger"
	"guarddev/modelapi"
	"guarddev/modelapi/anthropicapi"
	"guarddev/modelapi/geminiapi"
	"guarddev/modelapi/grokapi"
	"guarddev/paymentsmiddleware"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"go.uber.org/zap"

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
	selfhosting := os.Getenv("SELF_HOSTING") != ""

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

	Logger := logMiddleware.Logger(ctx)

	// Initialize the selected model based on environment variable
	var selectedModel modelapi.ModelAPI
	modelType := os.Getenv("MODEL_TYPE")
	switch modelType {
	case "anthropic":
		Logger.Info("[ModelAPI] Initializing Anthropic model")
		selectedModel = anthropicapi.Connect(ctx, anthropicapi.AnthropicConnectProps{
			Logger: logMiddleware,
		})
	case "grok":
		Logger.Info("[ModelAPI] Initializing Grok model")
		selectedModel = grokapi.Connect(ctx, grokapi.GrokConnectProps{
			Logger: logMiddleware,
		})
	default:
		if modelType == "" {
			Logger.Info("[ModelAPI] No model type specified, defaulting to Gemini")
		} else {
			Logger.Info("[ModelAPI] Unknown model type, defaulting to Gemini", zap.String("specified_type", modelType))
		}
		selectedModel = geminiapi.Connect(ctx, geminiapi.GeminiConnectProps{
			Logger: logMiddleware,
		})
	}

	var payments *paymentsmiddleware.Payments = nil

	if !selfhosting {
		payments = paymentsmiddleware.Connect(paymentsmiddleware.PaymentsConnectProps{
			Logger:   logMiddleware,
			Database: postgresClient,
		})
	}

	awsMiddleware := awsmiddleware.Connect(logMiddleware, selectedModel)

	srv := graph.Connnect(ctx, graph.GraphConnectProps{
		Logger:        logMiddleware,
		Database:      postgresClient,
		AWSMiddleware: awsMiddleware,
		Gemini:        selectedModel, // Note: This might need to be updated if the graph package expects specifically Gemini
		Payments:      payments,
	})

	// Start listening for connections
	graphRouter := getGraphqlSrv()
	graphRouter.Handle("/", srv)

	if !production {
		graphRouter.Handle("/playground", playground.Handler("GraphQL Playground", "/graph"))
		Logger.Info("[Graph] Connect to http://localhost:" + port + "/graph for GraphQL server")
		Logger.Info("[Graph] Connect to http://localhost:" + port + "/graph/playground for GraphQL playground")
	} else {
		Logger.Info("[Graph] Connect to https://api.guard.dev/graph for GraphQL server")
	}

	router := chi.NewRouter()
	router.Mount("/graph", graphRouter)

	if !selfhosting {
		router.Post("/payments", payments.HandleStripeWebhook)
	}

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
