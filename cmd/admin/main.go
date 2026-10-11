package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/membership"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/google/uuid"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("%s", usage())
	}
	command := os.Args[1]
	if command == "bootstrap-platform-admin" {
		return runBootstrap(os.Args[2:])
	}
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
		return fmt.Errorf("could not connect to DATABASE_URL")
	}
	defer p.Close()
	var operator bool
	if e = p.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users u
			WHERE u.id=$1 AND u.status='ACTIVE' AND u.email_verified_at IS NOT NULL
			AND EXISTS (
				SELECT 1 FROM user_platform_roles upr
				JOIN role_permissions rp ON rp.role_id=upr.role_id
				JOIN permissions p ON p.id=rp.permission_id AND p.scope='PLATFORM'
				WHERE upr.user_id=u.id AND p.code='platform_user.manage'
			)
		)
	`, *actor).Scan(&operator); e != nil {
		return e
	}
	if !operator {
		return fmt.Errorf("operator must be an active, verified platform administrator")
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
		return (user.Service{Pool: p}).Role(ctx, *actor, uid, "PLATFORM_ADMIN", "", false)
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

func runBootstrap(args []string) error {
	f := flag.NewFlagSet("bootstrap-platform-admin", flag.ContinueOnError)
	database := f.String("database", "", "expected PostgreSQL database name (required)")
	confirmDatabase := f.String("confirm-database", "", "repeat the expected database name (required)")
	email := f.String("email", "", "email of an existing verified user (required)")
	confirmEmail := f.String("confirm-email", "", "repeat the target email (required)")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 || validateBootstrapConfirmation(*database, *confirmDatabase, *email, *confirmEmail) != nil {
		return fmt.Errorf("usage: %s", usage())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := db.Open(ctx, os.Getenv("DATABASE_URL"), 2)
	if err != nil {
		return fmt.Errorf("could not connect to DATABASE_URL")
	}
	defer p.Close()
	userID, err := bootstrapPlatformAdmin(ctx, p, *database, *email)
	if err != nil {
		return err
	}
	fmt.Printf("Platform administrator assigned to existing verified account %s. No password was changed.\n", userID)
	return nil
}

func usage() string {
	return "usage: admin create-batch|create-membership|assign-role|assign-first-batch-rep|assign-platform-admin --actor USER_UUID [--batch SLUG --student NUMBER --role CODE --name NAME --year YEAR]\n       admin bootstrap-platform-admin --database DB --confirm-database DB --email EMAIL --confirm-email EMAIL"
}
