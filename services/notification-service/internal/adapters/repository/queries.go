package repository

const INSERT_PROCESSED_EVENT_QUERY = `
INSERT INTO processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING`

const INSERT_NOTIFICATION_QUERY = `
INSERT INTO notifications (id, user_id, event_id, event_type, text, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`

const LIST_NOTIFICATIONS_QUERY = `
SELECT id, user_id, event_id, event_type, text, created_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3`
