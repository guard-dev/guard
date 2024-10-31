package paymentsmiddleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"guarddev/database/postgres"
	"io"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/price"
	sub "github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/webhook"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (p *Payments) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("paymentsmiddleware/HandleStripeWebhook")
	ctx, span := tracer.Start(r.Context(), "HandleStripeWebhook")
	defer span.End()

	const MaxBodyBytes = int64(65536)
	bodyReader := http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(bodyReader)
	if err != nil {
		span.RecordError(err)
		p.RespondWithError(ctx, w, http.StatusBadRequest, "Error reading request body")
		return
	}

	event := stripe.Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		span.RecordError(err)
		p.RespondWithError(ctx, w, http.StatusBadRequest, "Error parsing webhook JSON")
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_ENDPOINT_SECRET")
	sigHeader := r.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/HandleStripeWebhook] Could not verify webhook signature",
			zap.Error(err),
			zap.String("Webhook Secret", endpointSecret),
			zap.String("Stripe Signature", sigHeader))
		p.RespondWithError(ctx, w, http.StatusBadRequest, "Error verifying webhook signature")
		return
	}

	// Add event type to span attributes
	span.SetAttributes(attribute.String("event.type", string(event.Type)))

	p.logger.Logger(ctx).Info(
		"[paymentsmiddleware/HandleStripeWebhook] Processing webhook event",
		zap.String("event_type", string(event.Type)))

	switch event.Type {

	case "customer.deleted":
		customer, err := p.parseCustomerBody(ctx, event.Data.Raw)
		if err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing customer data: "+err.Error())
			return
		}
		if err := p.DeleteCustomerFromDB(ctx, customer.ID); err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling customer deletion: "+err.Error())
			return
		}

	case "customer.subscription.deleted":
		subscription, err := p.parseSubscriptionBody(ctx, event.Data.Raw)
		if err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing subscription data: "+err.Error())
			return
		}
		if err := p.handleSubscriptionDeleted(ctx, *subscription); err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling subscription deletion: "+err.Error())
			return
		}

	case "customer.subscription.updated":
		subscription, err := p.parseSubscriptionBody(ctx, event.Data.Raw)
		if err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing subscription data: "+err.Error())
			return
		}
		if err := p.handleSubscriptionUpdated(ctx, *subscription); err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling subscription update: "+err.Error())
			return
		}

	case "customer.subscription.created":
		subscription, err := p.parseSubscriptionBody(ctx, event.Data.Raw)
		if err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing subscription data: "+err.Error())
			return
		}
		if err := p.handleSubscriptionCreated(ctx, *subscription); err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling subscription creation: "+err.Error())
			return
		}

	case "invoice.paid":
		invoice, err := p.parseInvoiceBody(ctx, event.Data.Raw)
		if err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing invoice data: "+err.Error())
			return
		}
		if err := p.handleInvoicePaid(ctx, *invoice); err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling invoice paid event: "+err.Error())
			return
		}

	case "invoice.payment_failed":
		invoice, err := p.parseInvoiceBody(ctx, event.Data.Raw)
		if err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing invoice data: "+err.Error())
			return
		}
		if err := p.handleInvoicePaymentFailed(ctx, *invoice); err != nil {
			span.RecordError(err)
			p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling invoice payment failed event: "+err.Error())
			return
		}

	default:
		msg := "Event type not handled: " + string(event.Type)
		p.logger.Logger(ctx).Info(
			"[paymentsmiddleware/HandleStripeWebhook] Unhandled event type",
			zap.String("event_type", string(event.Type)))
		p.RespondWithError(ctx, w, http.StatusOK, msg)
		return
	}

	RespondWithJSON(ctx, w, http.StatusOK, map[string]string{"status": "success"})
}

func (p *Payments) handleCustomerDeleted(ctx context.Context, event stripe.Event, w http.ResponseWriter) error {
	tracer := otel.Tracer("paymentsmiddleware/handleCustomerDeleted")
	ctx, span := tracer.Start(ctx, "handleCustomerDeleted")
	defer span.End()

	customer, err := p.parseCustomerBody(ctx, event.Data.Raw)
	if err != nil {
		span.RecordError(err)
		p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error parsing customer data: "+err.Error())
		return err
	}

	if err := p.DeleteCustomerFromDB(ctx, customer.ID); err != nil {
		span.RecordError(err)
		p.RespondWithError(ctx, w, http.StatusInternalServerError, "Error handling customer deletion: "+err.Error())
		return err
	}

	return nil
}

func (p *Payments) parseCustomerBody(ctx context.Context, jsonMessage json.RawMessage) (*stripe.Customer, error) {
	tracer := otel.Tracer("paymentsmiddleware/parseCustomerBody")
	ctx, span := tracer.Start(ctx, "parseCustomerBody")
	defer span.End()

	var customer stripe.Customer
	if err := json.Unmarshal(jsonMessage, &customer); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("could not parse customer body data: %w", err)
	}
	return &customer, nil
}

func (p *Payments) parseInvoiceBody(ctx context.Context, jsonMessage json.RawMessage) (*stripe.Invoice, error) {
	tracer := otel.Tracer("paymentsmiddleware/parseInvoiceBody")
	ctx, span := tracer.Start(ctx, "parseInvoiceBody")
	defer span.End()

	var invoice stripe.Invoice
	if err := json.Unmarshal(jsonMessage, &invoice); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("could not parse invoice body data: %w", err)
	}
	return &invoice, nil
}

func (p *Payments) parseSubscriptionBody(ctx context.Context, jsonMessage json.RawMessage) (*stripe.Subscription, error) {
	tracer := otel.Tracer("paymentsmiddleware/parseSubscriptionBody")
	ctx, span := tracer.Start(ctx, "parseSubscriptionBody")
	defer span.End()

	var subscription stripe.Subscription
	if err := json.Unmarshal(jsonMessage, &subscription); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("could not parse subscription data: %w", err)
	}
	return &subscription, nil
}

func (p *Payments) handleSubscriptionDeleted(ctx context.Context, subscription stripe.Subscription) error {
	tracer := otel.Tracer("paymentsmiddleware/handleSubscriptionDeleted")
	ctx, span := tracer.Start(ctx, "handleSubscriptionDeleted")
	defer span.End()

	customerId := subscription.Customer.ID
	p.logCustomer(ctx, customerId, "Customer Subscription Deleted")

	sub, err := p.database.DeleteSubscriptionByStripeSubscriptionId(ctx, sql.NullString{Valid: true, String: subscription.ID})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("could not delete subscription with id %s: %w", subscription.ID, err)
	}

	_, err = p.database.UpdateTeamStripeCustomerIdByTeamId(ctx, postgres.UpdateTeamStripeCustomerIdByTeamIdParams{
		TeamID:           sub.TeamID,
		StripeCustomerID: sql.NullString{Valid: false, String: ""},
	})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("could not delete stripe customer ID from DB. Team ID: %d, Subscription ID: %s: %w",
			sub.TeamID, subscription.ID, err)
	}

	p.logger.Logger(ctx).Info(
		"[paymentsmiddleware/handleSubscriptionDeleted] Deleted subscription from database",
		zap.String("subscription_id", subscription.ID))

	return nil
}

func (p *Payments) handleSubscriptionCreated(ctx context.Context, subscription stripe.Subscription) error {
	tracer := otel.Tracer("paymentsmiddleware/handleSubscriptionCreated")
	ctx, span := tracer.Start(ctx, "handleSubscriptionCreated")
	defer span.End()

	customerId := subscription.Customer.ID
	p.logCustomer(ctx, customerId, "Customer Subscription Created")

	team, err := p.database.GetTeamByStripeCustomerId(ctx, sql.NullString{Valid: true, String: customerId})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("could not fetch team from DB using stripe customer ID %s: %w", customerId, err)
	}

	// Identify which base product was purchased and get its usage price ID
	var usagePrice string
	for _, item := range subscription.Items.Data {
		if item.Price.LookupKey == "guard_pro_monthly" {
			usagePrice = "guard_pro_monthly_usage"
			break
		} else if item.Price.LookupKey == "guard_basic_monthly" {
			usagePrice = "guard_basic_monthly_usage"
			break
		}
	}

	if usagePrice == "" {
		p.logger.Logger(ctx).Error("Subscription does not match known products", zap.String("subscription_id", subscription.ID))
		return fmt.Errorf("unknown product in subscription: %s", subscription.ID)
	}

	// Fetch the product details to get the resources_included metadata
	prod, err := p.GetSubscriptionProduct(ctx, subscription.ID)
	if err != nil {
		return fmt.Errorf("Could not fetch subscription %s products: %s", subscription.ID, err.Error())
	}

	resources, ok := prod.Metadata["resources_included"]
	if !ok {
		return fmt.Errorf("No resources_included field in the product %s %s", prod.ID, prod.Name)
	}

	var resourcesIncluded int32
	if _, err := fmt.Sscanf(resources, "%d", &resourcesIncluded); err != nil {
		return fmt.Errorf(
			"Unable to parse included resources string in product (%s %s) '%s': %s",
			prod.ID, prod.Name, resources, err.Error())
	}

	// Get the price ID using the lookup key
	prices := stripe.PriceListParams{
		LookupKeys: stripe.StringSlice([]string{usagePrice}),
	}
	priceIter := price.List(&prices)
	var priceId string
	for priceIter.Next() {
		priceId = priceIter.Price().ID
		break
	}
	if priceId == "" {
		return fmt.Errorf("could not find price with lookup key: %s", usagePrice)
	}

	// Update subscription with the correct price ID
	_, err = sub.Update(
		subscription.ID,
		&stripe.SubscriptionParams{
			Items: []*stripe.SubscriptionItemsParams{
				{
					Price: stripe.String(priceId),
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to add usage-based price to subscription: %v", err)
	}

	// Create a new subscription entry in the database with resources included
	_, err = p.database.CreateSubscription(ctx, postgres.CreateSubscriptionParams{
		TeamID:               team.TeamID,
		StripeSubscriptionID: sql.NullString{Valid: true, String: subscription.ID},
		ResourcesIncluded:    resourcesIncluded,
	})
	if err != nil {
		return fmt.Errorf("failed to create subscription entry in the database: %v", err)
	}

	p.logger.Logger(ctx).Info("Added usage-based price to subscription and created subscription entry in DB", zap.String("subscription_id", subscription.ID))
	return nil
}

func (p *Payments) handleSubscriptionUpdated(ctx context.Context, subscription stripe.Subscription) error {
	tracer := otel.Tracer("paymentsmiddleware/handleSubscriptionUpdated")
	ctx, span := tracer.Start(ctx, "handleSubscriptionUpdated")
	defer span.End()

	customerId := subscription.Customer.ID
	p.logCustomer(ctx, customerId, "Customer Subscription Updated")

	// Add any additional subscription update logic here
	span.SetAttributes(attribute.String("subscription.id", subscription.ID))

	return nil
}

func (p *Payments) handleInvoicePaid(ctx context.Context, invoice stripe.Invoice) error {
	tracer := otel.Tracer("paymentsmiddleware/handleInvoicePaid")
	ctx, span := tracer.Start(ctx, "handleInvoicePaid")
	defer span.End()

	customerId := invoice.Customer.ID
	p.logCustomer(ctx, customerId, "Subscription Invoice Paid")

	// Only process if this is a subscription invoice
	if invoice.Subscription == nil {
		return nil
	}

	// Get the team associated with this customer
	team, err := p.database.GetTeamByStripeCustomerId(ctx, sql.NullString{
		Valid:  true,
		String: customerId,
	})
	if err != nil {
		return fmt.Errorf("could not fetch team for customer %s: %w", customerId, err)
	}

	// Check if subscription exists
	subscription, err := p.database.GetSubscriptionByTeamId(ctx, team.TeamID)
	if err != nil {
		if err == sql.ErrNoRows {
			p.logger.Logger(ctx).Warn(
				"[paymentsmiddleware/handleInvoicePaid] Subscription not found in database, possibly processing invoice.paid before subscription.created",
				zap.String("customer_id", customerId),
				zap.Int64("team_id", team.TeamID),
				zap.String("invoice_id", invoice.ID))
			// Return error to trigger Stripe retry
			return fmt.Errorf("subscription not found for team %d, waiting for retry", team.TeamID)
		}
		return fmt.Errorf("error checking subscription existence: %w", err)
	}

	// Reset the resources_used counter for the subscription
	_, err = p.database.ResetSubscriptionResourcesUsed(ctx, team.TeamID)
	if err != nil {
		return fmt.Errorf("could not reset resources_used for team %d: %w", team.TeamID, err)
	}

	p.logger.Logger(ctx).Info(
		"[paymentsmiddleware/handleInvoicePaid] Reset resources_used counter",
		zap.String("customer_id", customerId),
		zap.Int64("team_id", team.TeamID),
		zap.String("subscription_id", subscription.StripeSubscriptionID.String))

	return nil
}

func (p *Payments) handleInvoicePaymentFailed(ctx context.Context, invoice stripe.Invoice) error {
	tracer := otel.Tracer("paymentsmiddleware/handleInvoicePaymentFailed")
	ctx, span := tracer.Start(ctx, "handleInvoicePaymentFailed")
	defer span.End()

	customerId := invoice.Customer.ID
	p.logCustomer(ctx, customerId, "Subscription Invoice Payment Failed")

	span.SetAttributes(
		attribute.String("invoice.id", invoice.ID),
		attribute.String("customer.id", customerId),
	)

	return nil
}

func (p *Payments) logCustomer(ctx context.Context, stripeCustomerId string, event string) {
	tracer := otel.Tracer("paymentsmiddleware/logCustomer")
	ctx, span := tracer.Start(ctx, "logCustomer")
	defer span.End()

	customer, err := p.getCustomer(ctx, stripeCustomerId)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/logCustomer] Could not fetch stripe customer",
			zap.String("stripe_customer_id", stripeCustomerId),
			zap.Error(err))
		return
	}

	p.logger.Logger(ctx).Info(
		fmt.Sprintf("[paymentsmiddleware/logCustomer] %s", event),
		zap.String("customer_id", customer.ID),
		zap.String("customer_name", customer.Name),
		zap.String("customer_email", customer.Email))
}

// RespondWithError sends an error response with a given HTTP status code and error message.
func (p *Payments) RespondWithError(ctx context.Context, w http.ResponseWriter, code int, message string) {
	tracer := otel.Tracer("paymentsmiddleware/RespondWithError")
	ctx, span := tracer.Start(ctx, "RespondWithError")
	defer span.End()

	p.logger.Logger(ctx).Error(
		"[paymentsmiddleware/RespondWithError] Error occurred while handling stripe webhook",
		zap.String("error", message))
	RespondWithJSON(ctx, w, code, map[string]string{"error": message})
}

// RespondWithJSON sends a JSON response.
func RespondWithJSON(ctx context.Context, w http.ResponseWriter, code int, payload interface{}) {
	tracer := otel.Tracer("paymentsmiddleware/RespondWithJSON")
	ctx, span := tracer.Start(ctx, "RespondWithJSON")
	defer span.End()

	response, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	span.SetAttributes(
		attribute.Int("response.code", code),
		attribute.String("response.type", "json"),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
