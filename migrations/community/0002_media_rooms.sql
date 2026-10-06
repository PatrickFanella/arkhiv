-- Existing clients and stored discussion remain in the video room.
ALTER TABLE chat_messages ADD COLUMN media text NOT NULL DEFAULT 'video' CHECK (media IN ('video','audio'));
ALTER TABLE chat_messages ADD COLUMN item text NOT NULL DEFAULT '';
ALTER TABLE chat_messages ADD COLUMN representation text NOT NULL DEFAULT '';
ALTER TABLE chat_messages DROP CONSTRAINT chat_messages_deployment_user_id_idempotency_key_key;
ALTER TABLE chat_messages ADD UNIQUE(deployment,media,user_id,idempotency_key);
CREATE INDEX chat_messages_media_room ON chat_messages(deployment,collection,media,seq);
ALTER TABLE chat_requests ADD COLUMN media text NOT NULL DEFAULT 'video' CHECK (media IN ('video','audio'));
ALTER TABLE chat_requests DROP CONSTRAINT chat_requests_pkey;
ALTER TABLE chat_requests ADD PRIMARY KEY(deployment,media,user_id,key);
ALTER TABLE chat_restrictions ADD COLUMN media text NOT NULL DEFAULT 'video' CHECK (media IN ('video','audio'));
ALTER TABLE chat_restrictions DROP CONSTRAINT chat_restrictions_pkey;
ALTER TABLE chat_restrictions ADD PRIMARY KEY(deployment,media,user_id);
ALTER TABLE chat_moderation ADD COLUMN media text NOT NULL DEFAULT 'video' CHECK (media IN ('video','audio'));
