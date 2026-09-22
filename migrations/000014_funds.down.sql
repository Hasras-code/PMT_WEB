DROP TABLE funds;
ALTER TABLE events DROP CONSTRAINT events_batch_id_id_unique;
DELETE FROM role_permissions rp USING permissions p WHERE rp.permission_id=p.id AND p.code = ANY(ARRAY['fund.view','fund.create','fund.update','fund.close','fund.transaction.create','fund.transaction.reverse','fund.manager.assign','fund.manager.remove','fund.transfer.create','fund.transfer.reverse','birthday_fund.manage','birthday_contribution.record']);
DELETE FROM permissions WHERE code = ANY(ARRAY['fund.view','fund.create','fund.update','fund.close','fund.transaction.create','fund.transaction.reverse','fund.manager.assign','fund.manager.remove','fund.transfer.create','fund.transfer.reverse','birthday_fund.manage','birthday_contribution.record']);
