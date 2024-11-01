package graph

import (
	"context"
	"fmt"
	"guarddev/auth"
	"guarddev/awsmiddleware"
	"guarddev/database/postgres"
	"guarddev/logger"
	"guarddev/modelapi/geminiapi"
	"guarddev/paymentsmiddleware"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type GraphConnectProps struct {
	Logger        *logger.LogMiddleware
	Database      *postgres.Database
	AWSMiddleware *awsmiddleware.AWSMiddleware
	Gemini        *geminiapi.Gemini
	Payments      *paymentsmiddleware.Payments
}

func Connnect(ctx context.Context, args GraphConnectProps) *handler.Server {
	tracer := otel.Tracer("graph/Connect")
	_, span := tracer.Start(ctx, "Connect")
	defer span.End()

	gqlConfig := Config{Resolvers: &Resolver{
		Database:      args.Database,
		Logger:        args.Logger,
		AWSMiddleware: args.AWSMiddleware,
		Gemini:        args.Gemini,
		Payments:      args.Payments,
	}}
	gqlConfig.Directives.LoggedIn = func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		_, span := tracer.Start(ctx, "LoggedIn")
		defer span.End()

		if !isLoggedIn(ctx) {
			span.RecordError(fmt.Errorf("access denied"))
			return nil, fmt.Errorf("access denied")
		}
		return next(ctx)
	}

	gqlConfig.Directives.IsAdmin = func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		_, span := tracer.Start(ctx, "IsAdmin")
		defer span.End()

		if !isLoggedIn(ctx) {
			span.RecordError(fmt.Errorf("access denied"))
			return nil, fmt.Errorf("access denied")
		}
		if !isSuperAdmin(ctx) {
			span.RecordError(fmt.Errorf("access denied"))
			return nil, fmt.Errorf("access denied")
		}
		return next(ctx)
	}

	gqlConfig.Directives.MemberTeam = func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		_, span := tracer.Start(ctx, "MemberTeam")
		defer span.End()

		teamSlugField := obj.(map[string]interface{})["teamSlug"]
		if teamSlugField == nil {
			return next(ctx)
		}
		teamSlug := teamSlugField.(string)
		if !memberTeam(ctx, teamSlug, args.Database) {
			span.RecordError(fmt.Errorf("access denied"))
			return nil, fmt.Errorf("access denied")
		}
		return next(ctx)
	}

	gqlConfig.Directives.SubActive = func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		tracer := otel.Tracer("graph/SubActive")
		ctx, span := tracer.Start(ctx, "SubActive")
		defer span.End()

		selfhosting := os.Getenv("SELF_HOSTING") != ""
		if selfhosting {
			return next(ctx)
		}

		oc := graphql.GetOperationContext(ctx)
		vars := oc.Variables
		teamSlugField := vars["teamSlug"]

		if teamSlugField == nil {
			span.RecordError(fmt.Errorf("missing teamSlug"))
			return nil, fmt.Errorf("missing teamSlug")
		}

		teamSlug := teamSlugField.(string)
		active, err := subActive(ctx, teamSlug, args.Database)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("error checking subscription: %w", err)
		}

		if !active {
			span.RecordError(fmt.Errorf("subscription not active"))
			return nil, fmt.Errorf("subscription required")
		}

		return next(ctx)
	}

	var MB int64 = 1 << 20

	gqlServer := handler.New(NewExecutableSchema(gqlConfig))
	gqlServer.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	gqlServer.AddTransport(transport.Options{})
	gqlServer.AddTransport(transport.GET{})
	gqlServer.AddTransport(transport.POST{})
	gqlServer.AddTransport(transport.MultipartForm{MaxUploadSize: 1024 * MB, MaxMemory: 1024 * MB})

	gqlServer.SetQueryCache(lru.New[*ast.QueryDocument](100))

	gqlServer.Use(extension.Introspection{})
	gqlServer.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	gqlServer.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		_, span := tracer.Start(ctx, "AroundOperations")
		defer span.End()

		user := auth.FromContext(ctx)
		oc := graphql.GetOperationContext(ctx)
		logger := args.Logger.Logger(ctx)
		if user == nil {
			logger.Info("[Graph] Incoming Request", zap.String("operation_name", oc.OperationName), zap.String("user", "nil"))
		} else {
			emailAddr, err := auth.EmailFromContext(ctx)
			if err != nil {
				logger.Error("[Graph] Could Not Fetch User Email Address from Incoming Request", zap.Error(err), zap.String("operation_name", oc.OperationName))
				span.RecordError(err)
			}

			fullName, err := auth.FullnameFromContext(ctx)
			if err != nil {
				logger.Error("[Graph] Could Not Fetch User Full Name from Incoming Request", zap.Error(err), zap.String("operation_name", oc.OperationName))
				span.RecordError(err)
			}

			user, err := args.Database.GetUserByEmail(ctx, emailAddr)
			if err != nil {
				userPtr, err := args.Database.SetupNewUser(ctx, postgres.SetupNewUserProps{EmailAddr: emailAddr, FullName: fullName})
				if err == nil {
					user = *userPtr
				} else {
					span.RecordError(err)
				}
			}

			logger.Info("[Graph] Incoming Request", zap.String("operation_name", oc.OperationName), zap.String("user", user.Email))
		}
		return next(ctx)
	})

	return gqlServer
}

func subActive(ctx context.Context, teamSlug string, db *postgres.Database) (bool, error) {
	tracer := otel.Tracer("graph/subActive")
	ctx, span := tracer.Start(ctx, "subActive")
	defer span.End()

	team, err := db.GetTeamByTeamSlug(ctx, teamSlug)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to get team: %w", err)
	}

	sub, err := db.GetSubscriptionByTeamId(ctx, team.TeamID)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub.StripeSubscriptionID.Valid, nil
}

func memberTeam(ctx context.Context, teamSlug string, db *postgres.Database) bool {
	tracer := otel.Tracer("graph/memberTeam")
	ctx, span := tracer.Start(ctx, "memberTeam")
	defer span.End()

	if !isLoggedIn(ctx) {
		return false
	}

	if isSuperAdmin(ctx) {
		return true
	}

	userEmail, _ := auth.EmailFromContext(ctx)
	user, err := db.GetUserByEmail(ctx, userEmail)
	if err != nil {
		span.RecordError(err)
		return false
	}

	team, err := db.GetTeamByTeamSlug(ctx, teamSlug)
	if err != nil {
		span.RecordError(err)
		return false
	}

	_, err = db.GetTeamMembershipByTeamIdUserId(ctx, postgres.GetTeamMembershipByTeamIdUserIdParams{
		UserID: user.UserID,
		TeamID: team.TeamID,
	})
	if err != nil {
		span.RecordError(err)
	}

	return err == nil
}

func isLoggedIn(ctx context.Context) bool {
	tracer := otel.Tracer("graph/isLoggedIn")
	_, span := tracer.Start(ctx, "isLoggedIn")
	defer span.End()

	user := auth.FromContext(ctx)
	return user != nil
}

func isSuperAdmin(ctx context.Context) bool {
	tracer := otel.Tracer("graph/isSuperAdmin")
	_, span := tracer.Start(ctx, "isSuperAdmin")
	defer span.End()

	userEmail, _ := auth.EmailFromContext(ctx)
	return strings.Split(userEmail, "@")[1] == "guard.dev"
}
