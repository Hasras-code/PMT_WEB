ALTER TABLE users
 DROP CONSTRAINT users_combination_valid,
 ADD CONSTRAINT users_combination_valid
  CHECK (combination IN ('PMT-ICT', 'PMT-CS', 'PMT-STAT', 'PMT-MAN', 'PMT-CHEM'));
