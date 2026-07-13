ALTER TABLE notifications
DROP COLUMN IF EXISTS email_subject,
DROP COLUMN IF EXISTS email_cc,
DROP COLUMN IF EXISTS email_bcc;
