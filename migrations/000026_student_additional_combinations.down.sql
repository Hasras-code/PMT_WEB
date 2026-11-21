ALTER TABLE users
 DROP CONSTRAINT IF EXISTS users_combination_valid,
 ADD CONSTRAINT users_combination_valid
  CHECK (combination IN ('PMT-ICT', 'PMT-CS'));
