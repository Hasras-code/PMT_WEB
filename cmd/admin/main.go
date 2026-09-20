package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/membership"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: admin create-batch|create-membership|assign-role|assign-first-batch-rep|assign-platform-admin --actor USER_UUID [--batch SLUG --student NUMBER --role CODE --name NAME --year YEAR]")
	}
	command := os.Args[1]
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	actor := f.String("actor", "", "account UUID of the operator (required)")
	slug := f.String("batch", "", "batch slug")
	student := f.String("student", "", "student number")
	role := f.String("role", "BATCH_REP", "batch role")
	name := f.String("name", "", "batch name")
	year := f.Int("year", time.Now().Year(), "entry year")
	if e := f.Parse(os.Args[2:]); e != nil {
		return e
	}
	if _, e := uuid.Parse(*actor); e != nil {
		return fmt.Errorf("--actor must be a registered active account UUID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, e := db.Open(ctx, os.Getenv("DATABASE_URL"), 2)
	if e != nil {
		return e
	}
	defer p.Close()
	var active bool
	if e = p.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND status='ACTIVE')`, *actor).Scan(&active); e != nil {
		return e
	}
	if !active {
		return fmt.Errorf("operator account is not active")
	}
	if command == "create-batch" {
		id, e := (batch.Service{Pool: p}).Create(ctx, *actor, batch.Input{Name: *name, Slug: *slug, EntryYear: *year}, true)
		if e == nil {
			fmt.Println(id)
		}
		return e
	}
	var uid string
	if e = p.QueryRow(ctx, `SELECT id FROM users WHERE student_number=$1 AND status='ACTIVE'`, *student).Scan(&uid); e != nil {
		return fmt.Errorf("resolve active student: %w", e)
	}
	if command == "assign-platform-admin" {
		return db.Tx(ctx, p, func(tx pgx.Tx) error {
			tag, e := tx.Exec(ctx, `INSERT INTO user_platform_roles(user_id,role_id,assigned_by) SELECT $1,id,$2 FROM roles WHERE code='PLATFORM_ADMIN' ON CONFLICT DO NOTHING`, uid, *actor)
			if e != nil {
				return e
			}
			if tag.RowsAffected() == 0 {
				return nil
			}
			return audit.Record(ctx, tx, "", *actor, "PLATFORM_ROLE_ASSIGNED", "user", uid, map[string]string{"role": "PLATFORM_ADMIN", "source": "operator_cli"})
		})
	}
	var bid string
	if e = p.QueryRow(ctx, `SELECT id FROM batches WHERE slug=$1 AND status='ACTIVE'`, *slug).Scan(&bid); e != nil {
		return fmt.Errorf("resolve active batch: %w", e)
	}
	m := membership.Service{Pool: p}
	if command == "create-membership" {
		id, e := m.Create(ctx, *actor, bid, uid, true)
		if e == nil {
			fmt.Println(id)
		}
		return e
	}
	if command != "assign-role" && command != "assign-first-batch-rep" {
		return fmt.Errorf("unknown command")
	}
	var mid string
	if e = p.QueryRow(ctx, `SELECT id FROM batch_memberships WHERE batch_id=$1 AND user_id=$2 AND status='ACTIVE'`, bid, uid).Scan(&mid); e != nil {
		return fmt.Errorf("create active membership first: %w", e)
	}
	if command == "assign-first-batch-rep" {
		*role = "BATCH_REP"
	}
	return m.Role(ctx, *actor, bid, mid, *role, false, true)
}
