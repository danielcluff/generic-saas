# Database Schema Planning

## Overview
Comprehensive PostgreSQL database schema design for the SaaS platform, optimized for performance, scalability, and data integrity with proper indexing and relationships.

## Core Tables

### Users Table
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth-only users
    google_id VARCHAR(255) UNIQUE, -- NULL for email/password users
    email_verified BOOLEAN DEFAULT FALSE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url TEXT,
    timezone VARCHAR(50) DEFAULT 'UTC',
    locale VARCHAR(10) DEFAULT 'en',
    status VARCHAR(20) DEFAULT 'active', -- active, suspended, deleted
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Email Verification Table
```sql
CREATE TABLE email_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token);
CREATE INDEX idx_email_verifications_user_id ON email_verifications(user_id);
CREATE INDEX idx_email_verifications_expires_at ON email_verifications(expires_at);
```

### Password Reset Table
```sql
CREATE TABLE password_resets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_password_resets_token ON password_resets(token);
CREATE INDEX idx_password_resets_user_id ON password_resets(user_id);
CREATE INDEX idx_password_resets_expires_at ON password_resets(expires_at);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

## Subscription and Payment Tables

### Pricing Plans Table
```sql
CREATE TABLE pricing_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Pricing
    amount DECIMAL(10,2) NOT NULL, -- Price in smallest currency unit
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    interval VARCHAR(20) NOT NULL, -- month, year, week, day
    interval_count INTEGER DEFAULT 1,

    -- Trial settings
    trial_period_days INTEGER DEFAULT 0,

    -- Usage limits and features (JSON)
    features JSONB DEFAULT '{}',
    usage_limits JSONB DEFAULT '{}',

    -- Provider IDs
    stripe_price_id VARCHAR(255),
    paypal_plan_id VARCHAR(255),

    -- Plan metadata
    plan_type VARCHAR(20) DEFAULT 'subscription', -- subscription, one_time
    billing_scheme VARCHAR(20) DEFAULT 'per_unit', -- per_unit, tiered

    -- Status
    active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_pricing_plans_active ON pricing_plans(active);
CREATE INDEX idx_pricing_plans_currency ON pricing_plans(currency);
CREATE INDEX idx_pricing_plans_interval ON pricing_plans(interval);
CREATE INDEX idx_pricing_plans_stripe_price ON pricing_plans(stripe_price_id);
CREATE INDEX idx_pricing_plans_paypal_plan ON pricing_plans(paypal_plan_id);

CREATE TRIGGER update_pricing_plans_updated_at
    BEFORE UPDATE ON pricing_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Subscriptions Table
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES pricing_plans(id),

    -- Stripe fields
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_price_id VARCHAR(255),

    -- PayPal fields
    paypal_subscription_id VARCHAR(255) UNIQUE,
    paypal_plan_id VARCHAR(255),

    -- Subscription details
    status VARCHAR(50) NOT NULL, -- active, canceled, past_due, unpaid, trialing, incomplete
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,

    -- Cancellation
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    cancel_at TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT subscription_provider_check CHECK (
        (stripe_subscription_id IS NOT NULL AND paypal_subscription_id IS NULL) OR
        (stripe_subscription_id IS NULL AND paypal_subscription_id IS NOT NULL)
    ),
    CONSTRAINT subscription_periods_check CHECK (
        current_period_start IS NULL OR current_period_end IS NULL OR
        current_period_start < current_period_end
    ),
    CONSTRAINT trial_periods_check CHECK (
        trial_start IS NULL OR trial_end IS NULL OR
        trial_start < trial_end
    )
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_plan_id ON subscriptions(plan_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_stripe_customer ON subscriptions(stripe_customer_id);
CREATE INDEX idx_subscriptions_stripe_subscription ON subscriptions(stripe_subscription_id);
CREATE INDEX idx_subscriptions_paypal_subscription ON subscriptions(paypal_subscription_id);
CREATE INDEX idx_subscriptions_current_period_end ON subscriptions(current_period_end);
CREATE INDEX idx_subscriptions_trial_end ON subscriptions(trial_end);

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Payments Table
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    subscription_id UUID REFERENCES subscriptions(id),
    plan_id UUID REFERENCES pricing_plans(id),

    -- Payment details
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL, -- pending, succeeded, failed, canceled, refunded
    payment_type VARCHAR(20) NOT NULL, -- subscription, one_time, usage

    -- Provider details
    provider VARCHAR(20) NOT NULL, -- stripe, paypal

    -- Stripe fields
    stripe_payment_intent_id VARCHAR(255),
    stripe_invoice_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),

    -- PayPal fields
    paypal_order_id VARCHAR(255),
    paypal_capture_id VARCHAR(255),
    paypal_payment_id VARCHAR(255),

    -- Payment metadata
    description TEXT,
    failure_reason TEXT,
    metadata JSONB DEFAULT '{}',

    -- Refund information
    refunded_amount DECIMAL(10,2) DEFAULT 0,
    refunded_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT payment_provider_check CHECK (
        (provider = 'stripe' AND (stripe_payment_intent_id IS NOT NULL OR stripe_invoice_id IS NOT NULL)) OR
        (provider = 'paypal' AND paypal_order_id IS NOT NULL)
    ),
    CONSTRAINT payment_refund_check CHECK (refunded_amount >= 0 AND refunded_amount <= amount)
);

CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_provider ON payments(provider);
CREATE INDEX idx_payments_stripe_intent ON payments(stripe_payment_intent_id);
CREATE INDEX idx_payments_stripe_invoice ON payments(stripe_invoice_id);
CREATE INDEX idx_payments_paypal_order ON payments(paypal_order_id);
CREATE INDEX idx_payments_created_at ON payments(created_at);
CREATE INDEX idx_payments_paid_at ON payments(paid_at);

CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Invoices Table
```sql
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    subscription_id UUID REFERENCES subscriptions(id),

    -- Invoice details
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    amount_subtotal DECIMAL(10,2) NOT NULL,
    amount_tax DECIMAL(10,2) DEFAULT 0,
    amount_total DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,

    -- Status
    status VARCHAR(50) NOT NULL, -- draft, open, paid, void, uncollectible

    -- Provider details
    stripe_invoice_id VARCHAR(255) UNIQUE,
    paypal_invoice_id VARCHAR(255) UNIQUE,

    -- Dates
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,

    -- URLs and metadata
    hosted_invoice_url TEXT,
    invoice_pdf_url TEXT,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_invoices_user_id ON invoices(user_id);
CREATE INDEX idx_invoices_subscription_id ON invoices(subscription_id);
CREATE INDEX idx_invoices_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_stripe_id ON invoices(stripe_invoice_id);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_created_at ON invoices(created_at);

CREATE TRIGGER update_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

## Usage Tracking and Limits

### Usage Metrics Table
```sql
CREATE TABLE usage_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    subscription_id UUID REFERENCES subscriptions(id),

    -- Metric details
    metric_name VARCHAR(100) NOT NULL, -- api_calls, storage_gb, users, etc.
    metric_value DECIMAL(15,4) NOT NULL,
    metric_unit VARCHAR(20), -- calls, gb, users, etc.

    -- Time period
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_usage_metrics_user_id ON usage_metrics(user_id);
CREATE INDEX idx_usage_metrics_subscription_id ON usage_metrics(subscription_id);
CREATE INDEX idx_usage_metrics_name ON usage_metrics(metric_name);
CREATE INDEX idx_usage_metrics_period ON usage_metrics(period_start, period_end);
CREATE INDEX idx_usage_metrics_recorded_at ON usage_metrics(recorded_at);

-- Unique constraint for user + metric + period
CREATE UNIQUE INDEX idx_usage_metrics_unique ON usage_metrics(user_id, metric_name, period_start, period_end);
```

### Usage Events Table (for real-time tracking)
```sql
CREATE TABLE usage_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),

    -- Event details
    event_type VARCHAR(100) NOT NULL, -- api_call, storage_upload, user_invite, etc.
    event_value DECIMAL(15,4) DEFAULT 1,
    event_unit VARCHAR(20),

    -- Context
    resource_id VARCHAR(255), -- API endpoint, file ID, etc.
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_usage_events_user_id ON usage_events(user_id);
CREATE INDEX idx_usage_events_type ON usage_events(event_type);
CREATE INDEX idx_usage_events_created_at ON usage_events(created_at);

-- Partition by month for better performance
CREATE TABLE usage_events_y2024m01 PARTITION OF usage_events
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## Webhook Events Table

### Webhook Events Table
```sql
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Provider details
    provider VARCHAR(20) NOT NULL, -- stripe, paypal
    event_id VARCHAR(255) NOT NULL, -- Provider's event ID
    event_type VARCHAR(100) NOT NULL,

    -- Processing status
    status VARCHAR(20) DEFAULT 'pending', -- pending, processed, failed, skipped
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,

    -- Event data
    raw_data JSONB NOT NULL,
    processed_data JSONB,

    -- Error handling
    error_message TEXT,
    last_attempt_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    provider_created_at TIMESTAMP WITH TIME ZONE,
    received_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_provider ON webhook_events(provider);
CREATE INDEX idx_webhook_events_event_id ON webhook_events(provider, event_id);
CREATE INDEX idx_webhook_events_type ON webhook_events(event_type);
CREATE INDEX idx_webhook_events_status ON webhook_events(status);
CREATE INDEX idx_webhook_events_created_at ON webhook_events(created_at);

-- Unique constraint to prevent duplicate processing
CREATE UNIQUE INDEX idx_webhook_events_unique ON webhook_events(provider, event_id);
```

## Audit and Logging Tables

### Audit Log Table
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Actor information
    user_id UUID REFERENCES users(id),
    admin_id UUID REFERENCES users(id), -- For admin actions

    -- Action details
    action VARCHAR(100) NOT NULL, -- login, logout, subscription_created, payment_failed, etc.
    resource_type VARCHAR(50), -- user, subscription, payment, etc.
    resource_id VARCHAR(255),

    -- Context
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),

    -- Change details
    old_values JSONB,
    new_values JSONB,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Partition by month for better performance
ALTER TABLE audit_logs PARTITION BY RANGE (created_at);
```

### System Notifications Table
```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),

    -- Notification details
    type VARCHAR(50) NOT NULL, -- email, in_app, push
    category VARCHAR(50) NOT NULL, -- billing, security, feature, etc.
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,

    -- Delivery
    status VARCHAR(20) DEFAULT 'pending', -- pending, sent, delivered, failed, read
    sent_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
```

## Database Maintenance

### Cleanup Functions
```sql
-- Function to clean up expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Clean up expired refresh tokens
    DELETE FROM refresh_tokens WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    -- Clean up expired email verifications
    DELETE FROM email_verifications WHERE expires_at < NOW() AND verified_at IS NULL;

    -- Clean up expired password resets
    DELETE FROM password_resets WHERE expires_at < NOW() AND used_at IS NULL;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to archive old usage events
CREATE OR REPLACE FUNCTION archive_old_usage_events(days_to_keep INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := NOW() - (days_to_keep || ' days')::INTERVAL;

    -- Move old events to archive table (create if not exists)
    CREATE TABLE IF NOT EXISTS usage_events_archive (LIKE usage_events INCLUDING ALL);

    WITH moved_events AS (
        DELETE FROM usage_events
        WHERE created_at < cutoff_date
        RETURNING *
    )
    INSERT INTO usage_events_archive SELECT * FROM moved_events;

    GET DIAGNOSTICS archived_count = ROW_COUNT;
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;
```

### Scheduled Maintenance
```sql
-- Create scheduled jobs (using pg_cron extension)
-- Clean up expired tokens daily
SELECT cron.schedule('cleanup-tokens', '0 2 * * *', 'SELECT cleanup_expired_tokens();');

-- Archive old usage events monthly
SELECT cron.schedule('archive-usage', '0 3 1 * *', 'SELECT archive_old_usage_events(90);');
```

## Performance Optimizations

### Materialized Views
```sql
-- Monthly revenue summary
CREATE MATERIALIZED VIEW monthly_revenue AS
SELECT
    DATE_TRUNC('month', paid_at) as month,
    currency,
    COUNT(*) as payment_count,
    SUM(amount) as total_revenue,
    AVG(amount) as avg_payment_amount
FROM payments
WHERE status = 'succeeded'
    AND paid_at IS NOT NULL
GROUP BY DATE_TRUNC('month', paid_at), currency
ORDER BY month DESC;

CREATE UNIQUE INDEX idx_monthly_revenue_month_currency ON monthly_revenue(month, currency);

-- Refresh monthly revenue view daily
SELECT cron.schedule('refresh-revenue', '0 4 * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY monthly_revenue;');
```

### Database Configuration Recommendations
```sql
-- Recommended PostgreSQL settings for SaaS workload
-- postgresql.conf settings:

-- Memory settings
-- shared_buffers = 25% of RAM
-- effective_cache_size = 75% of RAM
-- work_mem = 4MB - 8MB
-- maintenance_work_mem = 256MB

-- Checkpoint settings
-- checkpoint_completion_target = 0.9
-- wal_buffers = 16MB
-- default_statistics_target = 100

-- Logging
-- log_statement = 'mod'
-- log_duration = on
-- log_min_duration_statement = 1000
```