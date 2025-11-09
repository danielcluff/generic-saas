-- Email tokens table for secure token management
-- Run this migration to add email token support

-- Create email_tokens table
CREATE TABLE IF NOT EXISTS email_tokens (
    id SERIAL PRIMARY KEY,
    token VARCHAR(64) NOT NULL, -- Stores hashed token (SHA-256 = 32 bytes = 64 hex chars)
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- password_reset, email_verification, magic_link
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    request_ip VARCHAR(45), -- Support IPv6 addresses
    user_agent TEXT
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_email_tokens_email_type ON email_tokens(email, type);
CREATE INDEX IF NOT EXISTS idx_email_tokens_token ON email_tokens(token);
CREATE INDEX IF NOT EXISTS idx_email_tokens_expires_at ON email_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_email_tokens_user_id ON email_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_email_tokens_created_at ON email_tokens(created_at);

-- Add email_verified_at column to users table if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'email_verified_at'
    ) THEN
        ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP WITH TIME ZONE;
    END IF;
END $$;

-- Create index on email verification status
CREATE INDEX IF NOT EXISTS idx_users_email_verified_at ON users(email_verified_at);

-- Add some constraints for data integrity
ALTER TABLE email_tokens
ADD CONSTRAINT chk_email_tokens_type
CHECK (type IN ('password_reset', 'email_verification', 'magic_link'));

-- Add constraint to ensure expires_at is in the future when created
ALTER TABLE email_tokens
ADD CONSTRAINT chk_email_tokens_expires_future
CHECK (expires_at > created_at);

-- Create a function to automatically cleanup expired tokens
-- This can be called periodically by a cron job or application scheduler
CREATE OR REPLACE FUNCTION cleanup_expired_email_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired tokens or used tokens older than 7 days
    DELETE FROM email_tokens
    WHERE expires_at < NOW()
       OR (used = TRUE AND created_at < NOW() - INTERVAL '7 days');

    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE email_tokens IS 'Stores secure tokens for email-based authentication flows';
COMMENT ON COLUMN email_tokens.token IS 'Hashed token (SHA-256) - never store plaintext tokens';
COMMENT ON COLUMN email_tokens.type IS 'Type of token: password_reset, email_verification, magic_link';
COMMENT ON COLUMN email_tokens.expires_at IS 'Token expiration time - tokens are invalid after this time';
COMMENT ON COLUMN email_tokens.used IS 'Whether the token has been consumed (single-use tokens)';
COMMENT ON COLUMN email_tokens.request_ip IS 'IP address of the request that generated this token';
COMMENT ON COLUMN email_tokens.user_agent IS 'User agent of the request for security logging';

COMMENT ON FUNCTION cleanup_expired_email_tokens() IS 'Cleanup function to remove expired and old used tokens';

-- Example usage for setting up periodic cleanup (adjust schedule as needed):
--
-- To run cleanup manually:
-- SELECT cleanup_expired_email_tokens();
--
-- To schedule with pg_cron (if available):
-- SELECT cron.schedule('cleanup-email-tokens', '0 2 * * *', 'SELECT cleanup_expired_email_tokens();');
--
-- Or use your application scheduler to call this function periodically