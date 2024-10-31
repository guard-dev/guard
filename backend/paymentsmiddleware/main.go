package paymentsmiddleware

import (
	"context"
	"database/sql"
	"fmt"
	"guarddev/auth"
	"guarddev/database/postgres"
	"guarddev/graph/model"
	"guarddev/logger"
	"os"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/paymentmethod"
	"github.com/stripe/stripe-go/v81/product"
	"github.com/stripe/stripe-go/v81/subscription"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type Payments struct {
	secretKey string
	database  *postgres.Database
	logger    *logger.LogMiddleware
}

type PaymentsConnectProps struct {
	Logger   *logger.LogMiddleware
	Database *postgres.Database
}

func Connect(args PaymentsConnectProps) *Payments {
	STRIPE_KEY := os.Getenv("STRIPE_SECRET_KEY")
	stripe.Key = STRIPE_KEY
	return &Payments{secretKey: STRIPE_KEY, database: args.Database, logger: args.Logger}
}

// Customer Management
func (p *Payments) createCustomer(ctx context.Context, email string, name string) (*stripe.Customer, error) {
	tracer := otel.Tracer("paymentsmiddleware/createCustomer")
	ctx, span := tracer.Start(ctx, "createCustomer")
	defer span.End()

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	customer, err := customer.New(params)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/createCustomer] Unable to create stripe customer",
			zap.Error(err),
			zap.String("email", email),
			zap.String("name", name))
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}
	return customer, nil
}

func (p *Payments) getCustomer(ctx context.Context, customerId string) (*stripe.Customer, error) {
	tracer := otel.Tracer("paymentsmiddleware/getCustomer")
	ctx, span := tracer.Start(ctx, "getCustomer")
	defer span.End()

	customer, err := customer.Get(customerId, nil)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/getCustomer] Unable to fetch stripe customer",
			zap.Error(err),
			zap.String("customer_id", customerId))
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	return customer, nil
}

func (p *Payments) GetLast4CardDigits(ctx context.Context, paymentMethodId string) (string, error) {
	tracer := otel.Tracer("paymentsmiddleware/GetLast4CardDigits")
	ctx, span := tracer.Start(ctx, "GetLast4CardDigits")
	defer span.End()

	pm, err := paymentmethod.Get(paymentMethodId, nil)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetLast4CardDigits] Unable to fetch payment method",
			zap.Error(err),
			zap.String("payment_method_id", paymentMethodId))
		return "", fmt.Errorf("failed to get payment method: %w", err)
	}
	return pm.Card.Last4, nil
}

func (p *Payments) UpdateCustomer(customerId string, newEmail string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(newEmail),
	}
	return customer.Update(customerId, params)
}

func (p *Payments) GetCustomerByTeamSlug(ctx context.Context, teamSlug string) (*stripe.Customer, error) {
	tracer := otel.Tracer("paymentsmiddleware/GetCustomerByTeamSlug")
	ctx, span := tracer.Start(ctx, "GetCustomerByTeamSlug")
	defer span.End()

	team, err := p.database.GetTeamByTeamSlug(ctx, teamSlug)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetCustomerByTeamSlug] Unable to fetch team from DB",
			zap.Error(err),
			zap.String("team_slug", teamSlug))
		return nil, fmt.Errorf("failed to fetch team: %w", err)
	}

	if team.StripeCustomerID.Valid {
		customer, err := p.getCustomer(ctx, team.StripeCustomerID.String)
		if err != nil {
			span.RecordError(err)
			p.logger.Logger(ctx).Error(
				"[paymentsmiddleware/GetCustomerByTeamSlug] Unable to fetch stripe customer",
				zap.Error(err),
				zap.String("stripe_customer_id", team.StripeCustomerID.String))
			return nil, fmt.Errorf("failed to fetch stripe customer: %w", err)
		}
		return customer, nil
	}

	userEmail, err := auth.EmailFromContext(ctx)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetCustomerByTeamSlug] Unable to fetch user email",
			zap.Error(err))
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}

	newCustomer, err := p.createCustomer(ctx, userEmail, team.TeamName)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetCustomerByTeamSlug] Unable to create stripe customer",
			zap.Error(err),
			zap.String("team_name", team.TeamName),
			zap.String("user_email", userEmail))
		return nil, fmt.Errorf("failed to create stripe customer: %w", err)
	}

	_, err = p.database.UpdateTeamStripeCustomerIdByTeamId(ctx, postgres.UpdateTeamStripeCustomerIdByTeamIdParams{
		TeamID:           team.TeamID,
		StripeCustomerID: sql.NullString{Valid: true, String: newCustomer.ID},
	})
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetCustomerByTeamSlug] Unable to update team stripe customer ID",
			zap.Error(err),
			zap.Int64("team_id", team.TeamID),
			zap.String("stripe_customer_id", newCustomer.ID))
		return nil, fmt.Errorf("failed to update team stripe customer ID: %w", err)
	}

	return newCustomer, nil
}

func (p *Payments) DeleteCustomer(customerId string) (*stripe.Customer, error) {
	return customer.Del(customerId, nil)
}

func (p *Payments) DeleteCustomerFromDB(ctx context.Context, customerId string) error {
	tracer := otel.Tracer("paymentsmiddleware/DeleteCustomerFromDB")
	ctx, span := tracer.Start(ctx, "DeleteCustomerFromDB")
	defer span.End()

	team, err := p.database.GetTeamByStripeCustomerId(ctx, sql.NullString{Valid: true, String: customerId})
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/DeleteCustomerFromDB] Unable to fetch team by stripe customer ID",
			zap.Error(err),
			zap.String("stripe_customer_id", customerId))
		return fmt.Errorf("failed to fetch team: %w", err)
	}

	team, err = p.database.UpdateTeamStripeCustomerIdByTeamId(ctx, postgres.UpdateTeamStripeCustomerIdByTeamIdParams{
		TeamID:           team.TeamID,
		StripeCustomerID: sql.NullString{Valid: false, String: ""},
	})
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/DeleteCustomerFromDB] Unable to update team stripe customer ID",
			zap.Error(err),
			zap.Int64("team_id", team.TeamID))
		return fmt.Errorf("failed to update team: %w", err)
	}

	p.logger.Logger(ctx).Info(
		"[paymentsmiddleware/DeleteCustomerFromDB] Deleted stripe customer ID for team",
		zap.String("team_name", team.TeamName),
		zap.Int64("team_id", team.TeamID))

	return nil
}

// Subscription Management
func (p *Payments) CreateSubscription(ctx context.Context, customerId string, priceId string) (*stripe.Subscription, error) {
	tracer := otel.Tracer("paymentsmiddleware/CreateSubscription")
	ctx, span := tracer.Start(ctx, "CreateSubscription")
	defer span.End()

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerId),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(priceId)},
		},
	}
	sub, err := subscription.New(params)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/CreateSubscription] Unable to create subscription",
			zap.Error(err),
			zap.String("customer_id", customerId),
			zap.String("price_id", priceId))
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}
	return sub, nil
}

func (p *Payments) GetSubscription(ctx context.Context, subscriptionId string) (*stripe.Subscription, error) {
	tracer := otel.Tracer("paymentsmiddleware/GetSubscription")
	ctx, span := tracer.Start(ctx, "GetSubscription")
	defer span.End()

	sub, err := subscription.Get(subscriptionId, nil)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetSubscription] Unable to fetch subscription",
			zap.Error(err),
			zap.String("subscription_id", subscriptionId))
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return sub, nil
}

func (p *Payments) GetSubscriptionProduct(ctx context.Context, subscriptionId string) (*stripe.Product, error) {
	sub, err := p.GetSubscription(ctx, subscriptionId)
	if err != nil {
		return nil, err
	}

	if len(sub.Items.Data) <= 0 {
		return nil, fmt.Errorf("No items found for subscription ID %s", subscriptionId)
	}

	firstItem := sub.Items.Data[0]
	productObject, err := product.Get(firstItem.Price.Product.ID, nil)
	if err != nil {
		return nil, err
	}

	return productObject, nil
}

func (p *Payments) GetSubscriptionAmountInUSDAndInterval(ctx context.Context, subscriptionId string) (int64, string, error) {
	sub, err := p.GetSubscription(ctx, subscriptionId)
	if err != nil {
		return 0, "", err
	}

	if len(sub.Items.Data) <= 0 {
		return 0, "", fmt.Errorf("No items found for subscription ID %s", subscriptionId)
	}

	firstItem := sub.Items.Data[0]
	interval := string(firstItem.Plan.Interval)

	// This assumes that firstItem.Plan.Currency will be USD
	// We are only listing out subscriptions in USD so this works for now
	// Will have to change in future when we list subsctipions in other currencies
	// currency := firstItem.Plan.Currency
	amount := firstItem.Plan.Amount

	return amount / 100, interval, nil
}

func (p *Payments) CancelSubscription(subscriptionId string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionCancelParams{
		InvoiceNow: stripe.Bool(true),
		Prorate:    stripe.Bool(true),
	}
	return subscription.Cancel(subscriptionId, params)
}

func (p *Payments) ListSubscriptions(ctx context.Context, customerId string) ([]*stripe.Subscription, error) {
	tracer := otel.Tracer("paymentsmiddleware/ListSubscriptions")
	ctx, span := tracer.Start(ctx, "ListSubscriptions")
	defer span.End()

	params := &stripe.SubscriptionListParams{}
	params.Filters.AddFilter("customer", "", customerId)
	i := subscription.List(params)
	var subscriptions []*stripe.Subscription
	for i.Next() {
		subscriptions = append(subscriptions, i.Subscription())
	}

	if err := i.Err(); err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/ListSubscriptions] Unable to list subscriptions",
			zap.Error(err),
			zap.String("customer_id", customerId))
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return subscriptions, nil
}

func (p *Payments) GetSubscriptionInterval(ctx context.Context, subscriptionId string) (stripe.PlanInterval, error) {
	subPlan, err := p.GetSubscription(ctx, subscriptionId)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", fmt.Errorf("No items found for subscription ID %s", subscriptionId)
	}

	firstItem := subPlan.Items.Data[0]
	interval := firstItem.Plan.Interval

	return interval, nil
}

// Payment Management
func (p *Payments) AttachPaymentMethod(ctx context.Context, customerId string, paymentMethodId string) (*stripe.PaymentMethod, error) {
	tracer := otel.Tracer("paymentsmiddleware/AttachPaymentMethod")
	ctx, span := tracer.Start(ctx, "AttachPaymentMethod")
	defer span.End()

	pm, err := paymentmethod.Attach(paymentMethodId, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerId),
	})
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/AttachPaymentMethod] Unable to attach payment method",
			zap.Error(err),
			zap.String("customer_id", customerId),
			zap.String("payment_method_id", paymentMethodId))
		return nil, fmt.Errorf("failed to attach payment method: %w", err)
	}
	return pm, nil
}

func (p *Payments) DetachPaymentMethod(ctx context.Context, paymentMethodId string) (*stripe.PaymentMethod, error) {
	tracer := otel.Tracer("paymentsmiddleware/DetachPaymentMethod")
	ctx, span := tracer.Start(ctx, "DetachPaymentMethod")
	defer span.End()

	pm, err := paymentmethod.Detach(paymentMethodId, nil)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/DetachPaymentMethod] Unable to detach payment method",
			zap.Error(err),
			zap.String("payment_method_id", paymentMethodId))
		return nil, fmt.Errorf("failed to detach payment method: %w", err)
	}
	return pm, nil
}

func (p *Payments) ListPaymentMethods(ctx context.Context, customerId string, paymentMethodType string) ([]*stripe.PaymentMethod, error) {
	tracer := otel.Tracer("paymentsmiddleware/ListPaymentMethods")
	ctx, span := tracer.Start(ctx, "ListPaymentMethods")
	defer span.End()

	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerId),
		Type:     stripe.String(paymentMethodType),
	}

	i := paymentmethod.List(params)
	var paymentMethods []*stripe.PaymentMethod
	for i.Next() {
		paymentMethods = append(paymentMethods, i.PaymentMethod())
	}

	if err := i.Err(); err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/ListPaymentMethods] Unable to list payment methods",
			zap.Error(err),
			zap.String("customer_id", customerId),
			zap.String("payment_method_type", paymentMethodType))
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	return paymentMethods, nil
}

func (p *Payments) GetSubscriptionPlanData(ctx context.Context, subscriptionId string) (*model.SubscriptionData, error) {
	tracer := otel.Tracer("paymentsmiddleware/GetSubscriptionPlanData")
	ctx, span := tracer.Start(ctx, "GetSubscriptionPlanData")
	defer span.End()

	subPlan, err := p.GetSubscription(ctx, subscriptionId)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetSubscriptionPlanData] Unable to fetch subscription",
			zap.Error(err),
			zap.String("subscription_id", subscriptionId))
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	if len(subPlan.Items.Data) <= 0 {
		err := fmt.Errorf("no items found for subscription ID %s", subscriptionId)
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetSubscriptionPlanData] No subscription items found",
			zap.String("subscription_id", subscriptionId))
		return nil, err
	}

	firstItem := subPlan.Items.Data[0]
	currentPeriodStart := time.Unix(subPlan.CurrentPeriodStart, 0).UTC()
	currentPeriodEnd := time.Unix(subPlan.CurrentPeriodEnd, 0).UTC()
	status := string(subPlan.Status)
	interval := string(firstItem.Plan.Interval)
	amount := firstItem.Plan.Amount
	costInUsd := (amount / 100)

	prod, err := p.GetSubscriptionProduct(ctx, subscriptionId)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetSubscriptionPlanData] Unable to fetch subscription product",
			zap.Error(err),
			zap.String("subscription_id", subscriptionId))
		return nil, fmt.Errorf("failed to fetch subscription product: %w", err)
	}

	paymentMethodId := subPlan.DefaultPaymentMethod.ID
	lastFourCardDigits, err := p.GetLast4CardDigits(ctx, paymentMethodId)
	if err != nil {
		span.RecordError(err)
		p.logger.Logger(ctx).Error(
			"[paymentsmiddleware/GetSubscriptionPlanData] Unable to fetch card details",
			zap.Error(err),
			zap.String("payment_method_id", paymentMethodId))
		return nil, fmt.Errorf("failed to fetch card details: %w", err)
	}

	subscriptionData := model.SubscriptionData{
		CurrentPeriodStart: currentPeriodStart.String(),
		CurrentPeriodEnd:   currentPeriodEnd.String(),
		Status:             status,
		Interval:           interval,
		PlanName:           prod.Name,
		CostInUsd:          costInUsd,
		LastFourCardDigits: lastFourCardDigits,
	}

	return &subscriptionData, nil
}
