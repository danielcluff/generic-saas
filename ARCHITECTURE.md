# Technical Architecture

## System Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend API   │    │   Database      │
│   (Astro +      │◄──►│   (Go)          │◄──►│   (PostgreSQL)  │
│    Solid.js)    │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │
         │              ┌─────────────────┐
         │              │   External      │
         └──────────────┤   Services      │
                        │   - Stripe      │
                        │   - PayPal      │
                        │   - Google OAuth│
                        └─────────────────┘
```

## Backend Architecture (Go)

### Directory Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes/
│   ├── auth/
│   ├── config/
│   ├── database/
│   ├── models/
│   ├── services/
│   │   ├── payment/
│   │   ├── user/
│   │   └── subscription/
│   └── utils/
├── pkg/
├── migrations/
├── sqlc/
│   ├── queries/
│   └── schema/
├── go.mod
└── go.sum
```

### Key Components

#### API Layer

-   **Router**: net/http package for HTTP routing
-   **Middleware**: Authentication, CORS, logging, rate limiting
-   **Handlers**: REST endpoint implementations
-   **Validation**: Request/response validation

#### Database Layer

-   **SQLC**: Type-safe SQL query generation
-   **Migrations**: Database schema versioning
-   **Connection Pool**: Optimized database connections

#### Service Layer

-   **User Service**: User management and profiles
-   **Auth Service**: Authentication and authorization
-   **Payment Service**: Stripe and PayPal integrations
-   **Subscription Service**: Subscription lifecycle management

#### External Integrations

-   **Stripe SDK**: Payment processing
-   **PayPal SDK**: Alternative payment processing
-   **Google OAuth**: Third-party authentication

## Frontend Architecture (Astro + Solid.js)

### Directory Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── ui/
│   │   ├── forms/
│   │   └── layout/
│   ├── pages/
│   │   ├── auth/
│   │   ├── dashboard/
│   │   └── settings/
│   ├── stores/
│   ├── utils/
│   ├── styles/
│   └── types/
├── public/
├── astro.config.mjs
└── package.json
```

### Key Features

-   **SSG/SSR**: Astro's hybrid rendering
-   **Solid.js Islands**: Interactive components
-   **State Management**: Solid stores for client state
-   **API Client**: HTTP client for backend communication

## Database Schema Design

### Core Tables

```sql
-- Users table
users (
  id UUID PRIMARY KEY,
  email VARCHAR UNIQUE NOT NULL,
  password_hash VARCHAR,
  google_id VARCHAR UNIQUE,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

-- Subscriptions table
subscriptions (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  stripe_subscription_id VARCHAR,
  paypal_subscription_id VARCHAR,
  status VARCHAR,
  current_period_start TIMESTAMP,
  current_period_end TIMESTAMP,
  created_at TIMESTAMP
)

-- Payments table
payments (
  id UUID PRIMARY KEY,
  subscription_id UUID REFERENCES subscriptions(id),
  amount DECIMAL,
  currency VARCHAR,
  provider VARCHAR, -- 'stripe' or 'paypal'
  external_id VARCHAR,
  status VARCHAR,
  created_at TIMESTAMP
)
```

## Security Considerations

### Authentication

-   JWT tokens with refresh mechanism
-   Password hashing with bcrypt
-   OAuth2 flow for Google integration
-   Session management and timeout

### API Security

-   CORS configuration
-   Rate limiting
-   Input validation and sanitization
-   SQL injection prevention (SQLC helps)

### Payment Security

-   PCI compliance considerations
-   Webhook signature verification
-   Secure API key management
-   Environment-based configuration

## Deployment Architecture

### Development Environment

-   Docker Compose for local development
-   Hot reload for both frontend and backend
-   Local PostgreSQL instance

### Production Environment

-   Container orchestration (Docker/Kubernetes)
-   Load balancer for high availability
-   Database connection pooling
-   CDN for static assets
-   SSL/TLS termination

## Technology Decisions

### Why Go for Backend?

-   Excellent performance and concurrency
-   Strong standard library
-   Great tooling and ecosystem
-   Easy deployment (single binary)

### Why Astro + Solid.js for Frontend?

-   Astro's partial hydration for optimal performance
-   Solid.js for reactive UI components
-   Great developer experience
-   SEO-friendly with SSG/SSR

### Why SQLC?

-   Type-safe database queries
-   Compile-time query validation
-   No ORM complexity
-   Performance benefits

## Scalability Considerations

-   Horizontal scaling of API servers
-   Database read replicas
-   Caching layer (Redis)
-   CDN for global content delivery
-   Microservices migration path
