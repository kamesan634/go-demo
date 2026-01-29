-- 刪除觸發器
DROP TRIGGER IF EXISTS update_friendships_updated_at ON friendships;
DROP TRIGGER IF EXISTS update_direct_messages_updated_at ON direct_messages;
DROP TRIGGER IF EXISTS update_messages_updated_at ON messages;
DROP TRIGGER IF EXISTS update_rooms_updated_at ON rooms;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- 刪除觸發器函數
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 刪除表（按依賴順序）
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS friendships;
DROP TABLE IF EXISTS blocked_users;
DROP TABLE IF EXISTS direct_messages;
DROP TABLE IF EXISTS message_attachments;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS room_members;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS users;
