CREATE FUNCTION protect_fund_transfer() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'fund transfers are immutable' USING ERRCODE='23514'; END IF;
 IF (NEW.batch_id,NEW.from_fund_id,NEW.to_fund_id,NEW.type,NEW.amount_minor,NEW.created_by,COALESCE(NEW.parent_transfer_id,'00000000-0000-0000-0000-000000000000')) IS DISTINCT FROM
    (OLD.batch_id,OLD.from_fund_id,OLD.to_fund_id,OLD.type,OLD.amount_minor,OLD.created_by,COALESCE(OLD.parent_transfer_id,'00000000-0000-0000-0000-000000000000'))
 THEN RAISE EXCEPTION 'fund transfers are immutable' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER fund_transfers_immutable BEFORE UPDATE OR DELETE ON fund_transfers FOR EACH ROW EXECUTE FUNCTION protect_fund_transfer();
