# Payment Processing Planning

## Overview
Comprehensive payment processing system supporting both Stripe and PayPal for subscriptions, one-time payments, and comprehensive webhook handling for payment lifecycle management.

## Payment Providers

### Stripe Integration

#### Subscription Management
```go
type StripeService struct {
    client    *client.API
    webhookSecret string
}

func (s *StripeService) CreateCustomer(email, name string) (*stripe.Customer, error) {
    params := &stripe.CustomerParams{
        Email: stripe.String(email),
        Name:  stripe.String(name),
    }
    return customer.New(params)
}

func (s *StripeService) CreateSubscription(customerID, priceID string) (*stripe.Subscription, error) {
    params := &stripe.SubscriptionParams{
        Customer: stripe.String(customerID),
        Items: []*stripe.SubscriptionItemsParams{
            {
                Price: stripe.String(priceID),
            },
        },
        PaymentBehavior: stripe.String("default_incomplete"),
        PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
            SaveDefaultPaymentMethod: stripe.String("on_subscription"),
        },
    }
    params.AddExpand("latest_invoice.payment_intent")
    return subscription.New(params)
}

func (s *StripeService) CreatePaymentIntent(amount int64, currency, customerID string) (*stripe.PaymentIntent, error) {
    params := &stripe.PaymentIntentParams{
        Amount:   stripe.Int64(amount),
        Currency: stripe.String(currency),
        Customer: stripe.String(customerID),
        AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
            Enabled: stripe.Bool(true),
        },
    }
    return paymentintent.New(params)
}
```

#### Webhook Handling
```go
func (s *StripeService) HandleWebhook(payload []byte, signature string) error {
    event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
    if err != nil {
        return fmt.Errorf("webhook signature verification failed: %v", err)
    }

    switch event.Type {
    case "customer.subscription.created":
        return s.handleSubscriptionCreated(event.Data.Object)
    case "customer.subscription.updated":
        return s.handleSubscriptionUpdated(event.Data.Object)
    case "customer.subscription.deleted":
        return s.handleSubscriptionDeleted(event.Data.Object)
    case "invoice.payment_succeeded":
        return s.handlePaymentSucceeded(event.Data.Object)
    case "invoice.payment_failed":
        return s.handlePaymentFailed(event.Data.Object)
    case "customer.subscription.trial_will_end":
        return s.handleTrialWillEnd(event.Data.Object)
    default:
        log.Printf("Unhandled event type: %s", event.Type)
    }

    return nil
}

func (s *StripeService) handleSubscriptionCreated(obj map[string]interface{}) error {
    var sub stripe.Subscription
    if err := mapstructure.Decode(obj, &sub); err != nil {
        return err
    }

    // Update database with subscription details
    return s.db.CreateSubscription(&Subscription{
        ID:                   generateUUID(),
        UserID:              s.getUserIDByCustomerID(sub.Customer.ID),
        StripeSubscriptionID: sub.ID,
        Status:               string(sub.Status),
        CurrentPeriodStart:   time.Unix(sub.CurrentPeriodStart, 0),
        CurrentPeriodEnd:     time.Unix(sub.CurrentPeriodEnd, 0),
        CreatedAt:           time.Now(),
    })
}
```

### PayPal Integration

#### Order Creation
```go
type PayPalService struct {
    client       *paypal.Client
    clientID     string
    clientSecret string
    environment  string // sandbox or live
}

func (s *PayPalService) CreateOrder(amount, currency string) (*paypal.Order, error) {
    order := paypal.Order{
        Intent: "CAPTURE",
        PurchaseUnits: []paypal.PurchaseUnitRequest{
            {
                Amount: &paypal.PurchaseUnitAmount{
                    Currency: currency,
                    Value:    amount,
                },
            },
        },
        ApplicationContext: &paypal.ApplicationContext{
            ReturnURL: fmt.Sprintf("%s/payments/paypal/success", s.frontendURL),
            CancelURL: fmt.Sprintf("%s/payments/paypal/cancel", s.frontendURL),
        },
    }

    return s.client.CreateOrder(context.Background(), order)
}

func (s *PayPalService) CaptureOrder(orderID string) (*paypal.CaptureOrderResponse, error) {
    return s.client.CaptureOrder(context.Background(), orderID, paypal.CaptureOrderRequest{})
}

func (s *PayPalService) CreateSubscription(planID string, subscriberEmail string) (*paypal.Subscription, error) {
    subscription := paypal.SubscriptionBase{
        PlanID: planID,
        Subscriber: &paypal.Subscriber{
            EmailAddress: subscriberEmail,
        },
        ApplicationContext: &paypal.ApplicationContext{
            ReturnURL: fmt.Sprintf("%s/subscriptions/paypal/success", s.frontendURL),
            CancelURL: fmt.Sprintf("%s/subscriptions/paypal/cancel", s.frontendURL),
        },
    }

    return s.client.CreateSubscription(context.Background(), subscription)
}
```

#### PayPal Webhook Handling
```go
func (s *PayPalService) HandleWebhook(payload []byte, headers map[string]string) error {
    // Verify webhook signature
    if !s.verifyWebhookSignature(payload, headers) {
        return errors.New("invalid webhook signature")
    }

    var event PayPalWebhookEvent
    if err := json.Unmarshal(payload, &event); err != nil {
        return err
    }

    switch event.EventType {
    case "BILLING.SUBSCRIPTION.CREATED":
        return s.handleSubscriptionCreated(event.Resource)
    case "BILLING.SUBSCRIPTION.ACTIVATED":
        return s.handleSubscriptionActivated(event.Resource)
    case "BILLING.SUBSCRIPTION.CANCELLED":
        return s.handleSubscriptionCancelled(event.Resource)
    case "PAYMENT.SALE.COMPLETED":
        return s.handlePaymentCompleted(event.Resource)
    case "PAYMENT.SALE.DENIED":
        return s.handlePaymentDenied(event.Resource)
    default:
        log.Printf("Unhandled PayPal event type: %s", event.EventType)
    }

    return nil
}
```

## Database Schema

### Subscriptions Table
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Stripe fields
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_price_id VARCHAR(255),

    -- PayPal fields
    paypal_subscription_id VARCHAR(255) UNIQUE,
    paypal_plan_id VARCHAR(255),

    -- Common fields
    status VARCHAR(50) NOT NULL, -- active, canceled, past_due, etc.
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    canceled_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT subscription_provider_check CHECK (
        (stripe_subscription_id IS NOT NULL AND paypal_subscription_id IS NULL) OR
        (stripe_subscription_id IS NULL AND paypal_subscription_id IS NOT NULL)
    )
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_stripe_customer ON subscriptions(stripe_customer_id);
CREATE INDEX idx_subscriptions_stripe_subscription ON subscriptions(stripe_subscription_id);
CREATE INDEX idx_subscriptions_paypal_subscription ON subscriptions(paypal_subscription_id);
```

### Payments Table
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    subscription_id UUID REFERENCES subscriptions(id),

    -- Payment details
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL, -- pending, succeeded, failed, canceled
    payment_type VARCHAR(20) NOT NULL, -- subscription, one_time

    -- Provider specific fields
    provider VARCHAR(20) NOT NULL, -- stripe, paypal
    stripe_payment_intent_id VARCHAR(255),
    stripe_invoice_id VARCHAR(255),
    paypal_order_id VARCHAR(255),
    paypal_capture_id VARCHAR(255),

    -- Metadata
    description TEXT,
    metadata JSONB,

    -- Timestamps
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT payment_provider_check CHECK (
        (provider = 'stripe' AND (stripe_payment_intent_id IS NOT NULL OR stripe_invoice_id IS NOT NULL)) OR
        (provider = 'paypal' AND paypal_order_id IS NOT NULL)
    )
);

CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX idx_payments_stripe_intent ON payments(stripe_payment_intent_id);
CREATE INDEX idx_payments_paypal_order ON payments(paypal_order_id);
```

### Pricing Plans Table
```sql
CREATE TABLE pricing_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Pricing
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    interval VARCHAR(20) NOT NULL, -- month, year
    interval_count INTEGER DEFAULT 1,

    -- Trial settings
    trial_period_days INTEGER DEFAULT 0,

    -- Provider IDs
    stripe_price_id VARCHAR(255),
    paypal_plan_id VARCHAR(255),

    -- Features (JSON)
    features JSONB,

    -- Status
    active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Frontend Payment Integration

### Stripe Elements Implementation
```tsx
// components/payments/StripeCheckout.tsx
import { createSignal, onMount } from 'solid-js';
import { loadStripe } from '@stripe/stripe-js';

interface StripeCheckoutProps {
  planId: string;
  amount: number;
  currency: string;
  onSuccess: () => void;
  onError: (error: string) => void;
}

export default function StripeCheckout(props: StripeCheckoutProps) {
  const [stripe, setStripe] = createSignal(null);
  const [elements, setElements] = createSignal(null);
  const [clientSecret, setClientSecret] = createSignal('');
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal('');

  onMount(async () => {
    // Initialize Stripe
    const stripeInstance = await loadStripe(import.meta.env.PUBLIC_STRIPE_PUBLISHABLE_KEY);
    setStripe(stripeInstance);

    // Create payment intent
    try {
      const response = await api.post('/payments/stripe/create-intent', {
        amount: props.amount,
        currency: props.currency,
        plan_id: props.planId
      });

      const { client_secret } = response.data.data;
      setClientSecret(client_secret);

      // Initialize Elements
      const elementsInstance = stripeInstance.elements({
        clientSecret: client_secret,
        appearance: {
          theme: 'stripe',
          variables: {
            colorPrimary: '#0570de',
          }
        }
      });

      const paymentElement = elementsInstance.create('payment');
      paymentElement.mount('#payment-element');

      setElements(elementsInstance);
    } catch (error) {
      setError('Failed to initialize payment');
    }
  });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();

    if (!stripe() || !elements()) return;

    setLoading(true);
    setError('');

    const { error: submitError } = await elements().submit();
    if (submitError) {
      setError(submitError.message);
      setLoading(false);
      return;
    }

    const { error: confirmError } = await stripe().confirmPayment({
      elements: elements(),
      confirmParams: {
        return_url: `${window.location.origin}/dashboard/billing/success`
      }
    });

    if (confirmError) {
      setError(confirmError.message);
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} class="space-y-6">
      <div id="payment-element" class="p-4 border rounded-lg">
        {/* Stripe Elements will mount here */}
      </div>

      {error() && (
        <div class="text-red-600 text-sm p-3 bg-red-50 rounded-lg">
          {error()}
        </div>
      )}

      <button
        type="submit"
        disabled={loading() || !stripe() || !clientSecret()}
        class="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-700"
      >
        {loading() ? 'Processing...' : `Pay $${props.amount / 100}`}
      </button>
    </form>
  );
}
```

### PayPal Button Implementation
```tsx
// components/payments/PayPalCheckout.tsx
import { createSignal, onMount } from 'solid-js';

interface PayPalCheckoutProps {
  planId: string;
  amount: string;
  currency: string;
  onSuccess: () => void;
  onError: (error: string) => void;
}

export default function PayPalCheckout(props: PayPalCheckoutProps) {
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal('');

  onMount(() => {
    // Load PayPal SDK
    const script = document.createElement('script');
    script.src = `https://www.paypal.com/sdk/js?client-id=${import.meta.env.PUBLIC_PAYPAL_CLIENT_ID}&currency=${props.currency}`;
    script.onload = initializePayPal;
    document.head.appendChild(script);
  });

  const initializePayPal = () => {
    window.paypal.Buttons({
      createOrder: async () => {
        try {
          const response = await api.post('/payments/paypal/create-order', {
            amount: props.amount,
            currency: props.currency,
            plan_id: props.planId
          });
          return response.data.data.order_id;
        } catch (error) {
          setError('Failed to create PayPal order');
          throw error;
        }
      },

      onApprove: async (data) => {
        try {
          setLoading(true);
          await api.post('/payments/paypal/capture-order', {
            order_id: data.orderID
          });
          props.onSuccess();
        } catch (error) {
          setError('Payment capture failed');
          props.onError('Payment capture failed');
        } finally {
          setLoading(false);
        }
      },

      onError: (err) => {
        console.error('PayPal error:', err);
        setError('PayPal payment failed');
        props.onError('PayPal payment failed');
      }
    }).render('#paypal-button-container');
  };

  return (
    <div class="space-y-4">
      {error() && (
        <div class="text-red-600 text-sm p-3 bg-red-50 rounded-lg">
          {error()}
        </div>
      )}

      <div id="paypal-button-container" class={loading() ? 'opacity-50 pointer-events-none' : ''}>
        {/* PayPal buttons will render here */}
      </div>

      {loading() && (
        <div class="text-center text-gray-600">
          Processing payment...
        </div>
      )}
    </div>
  );
}
```

## Subscription Management

### Backend Service
```go
type SubscriptionService struct {
    db           Database
    stripeService *StripeService
    paypalService *PayPalService
}

func (s *SubscriptionService) CreateSubscription(userID, planID, provider string) (*Subscription, error) {
    plan, err := s.db.GetPricingPlan(planID)
    if err != nil {
        return nil, err
    }

    switch provider {
    case "stripe":
        return s.createStripeSubscription(userID, plan)
    case "paypal":
        return s.createPayPalSubscription(userID, plan)
    default:
        return nil, errors.New("unsupported payment provider")
    }
}

func (s *SubscriptionService) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error {
    subscription, err := s.db.GetSubscription(subscriptionID)
    if err != nil {
        return err
    }

    if subscription.StripeSubscriptionID != "" {
        return s.stripeService.CancelSubscription(subscription.StripeSubscriptionID, cancelAtPeriodEnd)
    } else if subscription.PayPalSubscriptionID != "" {
        return s.paypalService.CancelSubscription(subscription.PayPalSubscriptionID)
    }

    return errors.New("invalid subscription provider")
}

func (s *SubscriptionService) GetUsageLimits(userID string) (*UsageLimits, error) {
    subscription, err := s.db.GetActiveSubscription(userID)
    if err != nil {
        return s.getFreeTierLimits(), nil // Default to free tier
    }

    plan, err := s.db.GetPricingPlan(subscription.PlanID)
    if err != nil {
        return nil, err
    }

    return s.extractUsageLimitsFromPlan(plan), nil
}
```

### Frontend Subscription Management
```tsx
// components/subscription/SubscriptionManager.tsx
import { createSignal, createEffect } from 'solid-js';
import { createStore } from 'solid-js/store';

interface Subscription {
  id: string;
  status: string;
  currentPeriodEnd: string;
  cancelAtPeriodEnd: boolean;
  provider: string;
  plan: PricingPlan;
}

export default function SubscriptionManager() {
  const [subscriptions, setSubscriptions] = createStore<Subscription[]>([]);
  const [loading, setLoading] = createSignal(true);

  createEffect(() => {
    loadSubscriptions();
  });

  const loadSubscriptions = async () => {
    try {
      const response = await api.get('/subscriptions');
      setSubscriptions(response.data.data);
    } catch (error) {
      console.error('Failed to load subscriptions:', error);
    } finally {
      setLoading(false);
    }
  };

  const cancelSubscription = async (subscriptionId: string, cancelAtPeriodEnd: boolean) => {
    try {
      await api.put(`/subscriptions/${subscriptionId}/cancel`, {
        cancel_at_period_end: cancelAtPeriodEnd
      });
      await loadSubscriptions(); // Refresh data
    } catch (error) {
      console.error('Failed to cancel subscription:', error);
    }
  };

  return (
    <div class="space-y-6">
      <h2 class="text-2xl font-bold">Subscription Management</h2>

      {loading() ? (
        <div>Loading subscriptions...</div>
      ) : (
        <div class="space-y-4">
          {subscriptions.map((sub) => (
            <div class="border rounded-lg p-6">
              <div class="flex justify-between items-start">
                <div>
                  <h3 class="text-lg font-semibold">{sub.plan.name}</h3>
                  <p class="text-gray-600">Status: {sub.status}</p>
                  <p class="text-gray-600">
                    Next billing: {new Date(sub.currentPeriodEnd).toLocaleDateString()}
                  </p>
                  <p class="text-gray-600">Provider: {sub.provider}</p>
                </div>

                <div class="space-x-2">
                  {sub.status === 'active' && !sub.cancelAtPeriodEnd && (
                    <button
                      onClick={() => cancelSubscription(sub.id, true)}
                      class="px-4 py-2 text-red-600 border border-red-600 rounded hover:bg-red-50"
                    >
                      Cancel at Period End
                    </button>
                  )}

                  {sub.cancelAtPeriodEnd && (
                    <button
                      onClick={() => cancelSubscription(sub.id, false)}
                      class="px-4 py-2 text-green-600 border border-green-600 rounded hover:bg-green-50"
                    >
                      Resume Subscription
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

## Security Considerations

### Webhook Security
- Verify webhook signatures from both Stripe and PayPal
- Use HTTPS for webhook endpoints
- Implement idempotency for webhook processing
- Log all webhook events for audit trails

### Payment Data Security
- Never store credit card information
- Use tokenization for payment methods
- Implement PCI DSS compliance where applicable
- Secure API key management

### Financial Reconciliation
- Regular payment reconciliation with provider reports
- Automated alerts for failed payments
- Comprehensive logging for all payment operations
- Regular audit trails and reporting