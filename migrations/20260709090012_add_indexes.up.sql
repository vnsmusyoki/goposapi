CREATE INDEX IF NOT EXISTS idx_stores_business_id
    ON stores(business_id);

CREATE INDEX IF NOT EXISTS idx_users_business_id
    ON users(business_id);

CREATE INDEX IF NOT EXISTS idx_users_store_id
    ON users(store_id);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id
    ON user_sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_user_sessions_refresh_token_hash
    ON user_sessions(refresh_token_hash);

CREATE INDEX IF NOT EXISTS idx_package_features_feature_id
    ON package_features(feature_id);

CREATE INDEX IF NOT EXISTS idx_business_subscriptions_business_id
    ON business_subscriptions(business_id);

CREATE INDEX IF NOT EXISTS idx_business_subscriptions_package_id
    ON business_subscriptions(package_id);

CREATE INDEX IF NOT EXISTS idx_business_subscriptions_business_status
    ON business_subscriptions(business_id, status);

CREATE INDEX IF NOT EXISTS idx_subscription_payments_business_subscription_id
    ON subscription_payments(business_subscription_id);

CREATE INDEX IF NOT EXISTS idx_subscription_payments_business_id
    ON subscription_payments(business_id);

CREATE INDEX IF NOT EXISTS idx_subscription_payments_status
    ON subscription_payments(status);

CREATE INDEX IF NOT EXISTS idx_subscription_payments_created_at
    ON subscription_payments(created_at);
