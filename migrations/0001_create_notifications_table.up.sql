CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    channel SMALLINT NOT NULL CHECK (channel IN (0, 1)),
    recipient TEXT NOT NULL,
    message TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    status SMALLINT NOT NULL CHECK (status IN (0, 1, 2, 3)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_status_scheduled_at
    ON notifications (status, scheduled_at);

CREATE TABLE IF NOT EXISTS telegram_receivers (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    chat_id BIGINT NOT NULL UNIQUE
);
