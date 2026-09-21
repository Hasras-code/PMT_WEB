ALTER TABLE users
 ADD COLUMN combination text,
 ADD CONSTRAINT users_combination_valid CHECK (combination IN ('PMT-ICT', 'PMT-CS')),
 ADD CONSTRAINT users_combination_required CHECK (combination IS NOT NULL) NOT VALID;

COMMENT ON COLUMN users.combination IS 'Student-selected degree combination. Required by the application for registrations created after this migration.';
