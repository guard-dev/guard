package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/clerkinc/clerk-sdk-go/clerk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var userCtxKey = &contextKey{"userId"}

type contextKey struct {
	name string
}

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracer := otel.Tracer("auth/Middleware")
			ctx, span := tracer.Start(r.Context(), "AuthMiddleware")
			defer span.End()

			vercelEnv := r.Header.Get("vercel_env")
			clientKey := os.Getenv("CLERK_SECRET_KEY")

			if vercelEnv == "preview" {
				clientKey = os.Getenv("CLERK_SECRET_KEY_ALTERNATIVE")
			}

			if clientKey == "" {
				log.Fatalln("ERROR: CANNOT FIND CLERK CLIENT KEY")
			}

			client, _ := clerk.NewClient(clientKey)
			header := r.Header.Get("Authorization")

			// User is unauthenticated.
			if header == "" {
				span.AddEvent("Unauthenticated user, no Authorization header")
				next.ServeHTTP(w, r)
				return
			}

			sessionToken := strings.Split(header, " ")[1]
			sessClaims, err := client.VerifyToken(sessionToken)
			if err != nil {
				span.RecordError(err)
				http.Error(w, "Invalid Authorization Token", http.StatusForbidden)
				return
			}

			user, err := client.Users().Read(sessClaims.Claims.Subject)
			if err != nil {
				span.RecordError(err)
				http.Error(w, "Malformed Authorization Token", http.StatusForbidden)
				return
			}

			span.SetAttributes(
				attribute.String("user.id", user.ID),
				attribute.String("user.email", user.EmailAddresses[0].EmailAddress),
			)

			ctx = AttachContext(ctx, user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func AttachContext(ctx context.Context, user *clerk.User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

func FromContext(ctx context.Context) *clerk.User {
	raw, _ := ctx.Value(userCtxKey).(*clerk.User)
	return raw
}

func EmailFromContext(ctx context.Context) (string, error) {
	tracer := otel.Tracer("auth/EmailFromContext")
	ctx, span := tracer.Start(ctx, "EmailFromContext")
	defer span.End()

	user := FromContext(ctx)
	return getEmail(ctx, user)
}

func getEmail(ctx context.Context, user *clerk.User) (string, error) {
	tracer := otel.Tracer("auth/getEmail")
	_, span := tracer.Start(ctx, "getEmail")
	defer span.End()

	if user == nil {
		err := fmt.Errorf("not logged in")
		span.RecordError(err)
		return "", err
	}
	for _, emailAddr := range user.EmailAddresses {
		if emailAddr.ID == *user.PrimaryEmailAddressID {
			return emailAddr.EmailAddress, nil
		}
	}
	return user.EmailAddresses[0].EmailAddress, nil
}

func FullnameFromContext(ctx context.Context) (string, error) {
	tracer := otel.Tracer("auth/FullnameFromContext")
	ctx, span := tracer.Start(ctx, "FullnameFromContext")
	defer span.End()

	user := FromContext(ctx)
	if user == nil {
		err := fmt.Errorf("not logged in")
		span.RecordError(err)
		return "", err
	}
	return fmt.Sprintf("%s %s", *user.FirstName, *user.LastName), nil
}
