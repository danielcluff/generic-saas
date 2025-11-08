# Backend API Planning

## API Design Philosophy
- RESTful design principles
- Consistent response formats
- Proper HTTP status codes
- Clear error messaging
- API versioning strategy

## Base Configuration

### Server Setup
```go
// Configuration structure
type Config struct {
    Port            string
    DatabaseURL     string
    JWTSecret       string
    StripeKey       string
    PayPalClientID  string
    GoogleClientID  string
    Environment     string // dev, staging, prod
}
```

### Response Format
```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

## Authentication Endpoints

### POST /api/v1/auth/register
Register new user with email/password
```go
type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

type AuthResponse struct {
    Token        string `json:"token"`
    RefreshToken string `json:"refresh_token"`
    User         User   `json:"user"`
}
```

### POST /api/v1/auth/login
Login with email/password
```go
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

### POST /api/v1/auth/google
Google OAuth authentication
```go
type GoogleAuthRequest struct {
    Code string `json:"code" validate:"required"`
}
```

### POST /api/v1/auth/refresh
Refresh JWT token
```go
type RefreshRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
}
```

### POST /api/v1/auth/logout
Logout and invalidate tokens
```go
type LogoutRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
}
```

## User Management Endpoints

### GET /api/v1/user/profile
Get current user profile
```go
type UserProfile struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### PUT /api/v1/user/profile
Update user profile
```go
type UpdateProfileRequest struct {
    Email string `json:"email" validate:"omitempty,email"`
}
```

### DELETE /api/v1/user/account
Delete user account
```go
type DeleteAccountRequest struct {
    Password string `json:"password" validate:"required"`
}
```

## Subscription Management Endpoints

### GET /api/v1/subscriptions
Get user's subscriptions
```go
type Subscription struct {
    ID                   string    `json:"id"`
    Status               string    `json:"status"`
    CurrentPeriodStart   time.Time `json:"current_period_start"`
    CurrentPeriodEnd     time.Time `json:"current_period_end"`
    Provider             string    `json:"provider"` // stripe, paypal
    PlanID               string    `json:"plan_id"`
    CreatedAt           time.Time `json:"created_at"`
}
```

### POST /api/v1/subscriptions
Create new subscription
```go
type CreateSubscriptionRequest struct {
    PlanID   string `json:"plan_id" validate:"required"`
    Provider string `json:"provider" validate:"required,oneof=stripe paypal"`
}
```

### PUT /api/v1/subscriptions/{id}/cancel
Cancel subscription
```go
type CancelSubscriptionRequest struct {
    CancelAtPeriodEnd bool `json:"cancel_at_period_end"`
}
```

## Payment Processing Endpoints

### POST /api/v1/payments/stripe/create-intent
Create Stripe payment intent
```go
type CreatePaymentIntentRequest struct {
    Amount   int64  `json:"amount" validate:"required,min=1"`
    Currency string `json:"currency" validate:"required"`
    PlanID   string `json:"plan_id,omitempty"`
}

type PaymentIntentResponse struct {
    ClientSecret string `json:"client_secret"`
    IntentID     string `json:"intent_id"`
}
```

### POST /api/v1/payments/paypal/create-order
Create PayPal order
```go
type CreatePayPalOrderRequest struct {
    Amount   string `json:"amount" validate:"required"`
    Currency string `json:"currency" validate:"required"`
    PlanID   string `json:"plan_id,omitempty"`
}

type PayPalOrderResponse struct {
    OrderID string `json:"order_id"`
    Links   []Link `json:"links"`
}
```

### GET /api/v1/payments/history
Get payment history
```go
type Payment struct {
    ID           string    `json:"id"`
    Amount       float64   `json:"amount"`
    Currency     string    `json:"currency"`
    Provider     string    `json:"provider"`
    Status       string    `json:"status"`
    Description  string    `json:"description"`
    CreatedAt    time.Time `json:"created_at"`
}
```

## Webhook Endpoints

### POST /api/v1/webhooks/stripe
Handle Stripe webhooks
```go
type StripeWebhookHandler struct {
    // Event processing logic
}
```

### POST /api/v1/webhooks/paypal
Handle PayPal webhooks
```go
type PayPalWebhookHandler struct {
    // Event processing logic
}
```

## Middleware Components

### Authentication Middleware
```go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // JWT token validation
        // Set user context
    }
}
```

### Rate Limiting Middleware
```go
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Rate limiting logic
    }
}
```

### CORS Middleware
```go
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // CORS headers
    }
}
```

### Logging Middleware
```go
func LoggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Request/response logging
    }
}
```

## Error Handling

### Standard Error Codes
- `AUTH_REQUIRED`: Authentication required
- `INVALID_CREDENTIALS`: Invalid login credentials
- `USER_EXISTS`: User already exists
- `INVALID_TOKEN`: Invalid or expired token
- `PAYMENT_FAILED`: Payment processing failed
- `SUBSCRIPTION_INVALID`: Invalid subscription state
- `RATE_LIMITED`: Too many requests
- `INTERNAL_ERROR`: Server error

### Error Response Format
```go
func HandleError(c *gin.Context, err error, statusCode int) {
    response := APIResponse{
        Success: false,
        Error: &APIError{
            Code:    determineErrorCode(err),
            Message: err.Error(),
        },
    }
    c.JSON(statusCode, response)
}
```

## Database Queries (SQLC)

### User Queries
```sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateUser :exec
UPDATE users SET email = $2, updated_at = $3 WHERE id = $1;
```

### Subscription Queries
```sql
-- name: GetUserSubscriptions :many
SELECT * FROM subscriptions WHERE user_id = $1;

-- name: CreateSubscription :one
INSERT INTO subscriptions (id, user_id, stripe_subscription_id, status, current_period_start, current_period_end, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions SET status = $2, updated_at = $3 WHERE id = $1;
```

## Service Layer Structure

### User Service
```go
type UserService interface {
    CreateUser(ctx context.Context, email, password string) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    UpdateUser(ctx context.Context, userID string, updates UserUpdates) error
    DeleteUser(ctx context.Context, userID string) error
}
```

### Payment Service
```go
type PaymentService interface {
    CreateStripePaymentIntent(ctx context.Context, amount int64, currency string) (*stripe.PaymentIntent, error)
    CreatePayPalOrder(ctx context.Context, amount, currency string) (*PayPalOrder, error)
    ProcessWebhook(ctx context.Context, provider string, payload []byte) error
}
```

### Subscription Service
```go
type SubscriptionService interface {
    CreateSubscription(ctx context.Context, userID, planID, provider string) (*Subscription, error)
    GetUserSubscriptions(ctx context.Context, userID string) ([]*Subscription, error)
    CancelSubscription(ctx context.Context, subscriptionID string) error
    UpdateSubscriptionFromWebhook(ctx context.Context, externalID, status string) error
}
```