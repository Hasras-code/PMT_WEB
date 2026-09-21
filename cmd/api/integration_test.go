package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/membership"
	"github.com/Hasras-code/PMT_WEB.git/internal/notification"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	appstore "github.com/Hasras-code/PMT_WEB.git/internal/store"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type mailbox struct {
	mu     sync.Mutex
	tokens map[string]string
	fail   bool
}

func (m *mailbox) Send(_ context.Context, to, purpose, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[to+":"+purpose] = token
	if m.fail {
		return fmt.Errorf("mail unavailable")
	}
	return nil
}
func (m *mailbox) token(email, purpose string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tokens[email+":"+purpose]
}

type fixture struct {
	t     *testing.T
	p     *pgxpool.Pool
	as    *auth.Service
	mail  *mailbox
	h     http.Handler
	store *storage.Local
}

func setup(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec(ctx, `CREATE SCHEMA `+schema); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		_, e := admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		if e != nil {
			t.Error(e)
		}
		admin.Close()
	})
	pc, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	pc.ConnConfig.RuntimeParams["search_path"] = schema
	pc.MaxConns = 8
	p, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(p.Close)
	files, e := filepath.Glob("../../migrations/*.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(files)
	for _, path := range files {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = p.Exec(ctx, string(b)); e != nil {
			t.Fatalf("migration %s: %v", path, e)
		}
	}
	secret := strings.Repeat("s", 48)
	store, e := storage.Open(t.TempDir(), secret)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = store.Close() })
	mail := &mailbox{tokens: map[string]string{}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	as := &auth.Service{Pool: p, Store: appstore.NewStorage(p), Signer: auth.Signer{Secret: []byte(secret), Issuer: "test", Audience: "test"}, Mail: mail, Log: log, Cost: 4}
	a := &app{
		cfg:     config.Config{BaseURL: "http://api.test", Secret: secret, Origins: []string{"http://client.test"}},
		pool:    p,
		auth:    as,
		files:   store,
		uploads: upload.Service{Pool: p, Store: store, BaseURL: "http://api.test"},
		logger:  log,
	}
	return &fixture{t: t, p: p, as: as, mail: mail, h: a.mount(), store: store}
}
func (f *fixture) request(method, path, token string, body any, expected int) []byte {
	f.t.Helper()
	var b []byte
	if body != nil {
		var e error
		b, e = json.Marshal(body)
		if e != nil {
			f.t.Fatal(e)
		}
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.RemoteAddr = "127.0.0.1:12345"
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	if w.Code != expected {
		f.t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, expected, w.Body.String())
	}
	return w.Body.Bytes()
}
func object(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal(b, &v); e != nil {
		t.Fatal(e, string(b))
	}
	return v
}
func idOf(t *testing.T, b []byte) string { return object(t, b)["id"].(string) }
func (f *fixture) newUser(label string) (string, string, string) {
	f.t.Helper()
	ctx := context.Background()
	email := strings.ToLower(label) + "@example.test"
	in := auth.RegisterInput{StudentNumber: label, Combination: "PMT-ICT", FirstName: "Test", LastName: "User", DisplayName: label, Email: email, Password: "correct horse battery"}
	if e := f.as.Register(ctx, in); e != nil {
		f.t.Fatal(e)
	}
	if e := f.as.ConsumeToken(ctx, f.mail.token(email, "EMAIL_VERIFY"), "EMAIL_VERIFY", ""); e != nil {
		f.t.Fatal(e)
	}
	tokens, e := f.as.Login(ctx, email, in.Password, "token", "test")
	if e != nil {
		f.t.Fatal(e)
	}
	var uid string
	if e = f.p.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, email).Scan(&uid); e != nil {
		f.t.Fatal(e)
	}
	return uid, tokens.AccessToken, tokens.RefreshToken
}
func (f *fixture) newBatch(actor, slug string) string {
	id, e := (batch.Service{Pool: f.p}).Create(context.Background(), actor, batch.Input{Name: slug, Slug: slug, EntryYear: 2026}, true)
	if e != nil {
		f.t.Fatal(e)
	}
	return id
}
func (f *fixture) member(actor, bid, uid, role string) string {
	m := membership.Service{Pool: f.p}
	id, e := m.Create(context.Background(), actor, bid, uid, true)
	if e != nil {
		f.t.Fatal(e)
	}
	if role != "" {
		if e = m.Role(context.Background(), actor, bid, id, role, false, true); e != nil {
			f.t.Fatal(e)
		}
	}
	return id
}
func (f *fixture) sql(query string, args ...any) {
	f.t.Helper()
	if _, e := f.p.Exec(context.Background(), query, args...); e != nil {
		f.t.Fatal(e)
	}
}
func (f *fixture) upload(path, token, name, mime string, b []byte) string {
	f.t.Helper()
	v := object(f.t, f.request("POST", path, token, map[string]any{"file_name": name, "mime_type": mime, "size_bytes": len(b)}, 201))
	r := httptest.NewRequest("PUT", v["upload_url"].(string), bytes.NewReader(b))
	r.Header.Set("Content-Type", mime)
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	if w.Code != 204 {
		f.t.Fatalf("upload %d %s", w.Code, w.Body.String())
	}
	return v["upload_id"].(string)
}
func TestAuthLifecycle(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.request("POST", "/v1/auth/register", "", map[string]any{"student_number": "ST-MISSING", "first_name": "A", "last_name": "B", "display_name": "AB", "email": "missing-combination@example.test", "password": "correct horse battery"}, 422)
	f.request("POST", "/v1/auth/register", "", map[string]any{"student_number": "ST-INVALID", "combination": "PMT-MATH", "first_name": "A", "last_name": "B", "display_name": "AB", "email": "invalid-combination@example.test", "password": "correct horse battery"}, 422)
	body := map[string]any{"student_number": "ST001", "combination": "pmt-ict", "first_name": "A", "last_name": "B", "display_name": "AB", "email": "User@Example.test", "password": "correct horse battery"}
	b := f.request("POST", "/v1/auth/register", "", body, 202)
	if bytes.Contains(b, []byte("token")) {
		t.Fatal("verification token exposed")
	}
	f.request("POST", "/v1/auth/register", "", body, 202)
	f.request("POST", "/v1/auth/login", "", map[string]any{"email": "user@example.test", "password": "correct horse battery"}, 401)
	old := f.mail.token("user@example.test", "EMAIL_VERIFY")
	f.request("POST", "/v1/auth/resend-verification", "", map[string]any{"email": "user@example.test"}, 202)
	raw := f.mail.token("user@example.test", "EMAIL_VERIFY")
	f.request("POST", "/v1/auth/verify-email", "", map[string]any{"token": old}, 422)
	f.request("POST", "/v1/auth/verify-email", "", map[string]any{"token": raw}, 204)
	f.request("POST", "/v1/auth/verify-email", "", map[string]any{"token": raw}, 422)
	login := object(t, f.request("POST", "/v1/auth/login", "", map[string]any{"email": "user@example.test", "password": "correct horse battery"}, 200))
	access := login["access_token"].(string)
	refresh := login["refresh_token"].(string)
	me := object(t, f.request("GET", "/v1/me", access, nil, 200))
	if me["combination"] != "PMT-ICT" {
		t.Fatalf("combination = %v, want PMT-ICT", me["combination"])
	}
	f.request("POST", "/v1/auth/login", "", map[string]any{"email": "user@example.test", "password": "wrong password"}, 401)
	rotated := object(t, f.request("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": refresh}, 200))
	f.request("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": refresh}, 401)
	f.request("GET", "/v1/me", rotated["access_token"].(string), nil, 401)
	tok, e := f.as.Login(ctx, "user@example.test", "correct horse battery", "token", "")
	if e != nil {
		t.Fatal(e)
	}
	f.request("POST", "/v1/auth/forgot-password", "", map[string]any{"email": "user@example.test"}, 202)
	reset := f.mail.token("user@example.test", "PASSWORD_RESET")
	f.request("POST", "/v1/auth/reset-password", "", map[string]any{"token": reset, "password": "a new long password"}, 204)
	f.request("GET", "/v1/me", tok.AccessToken, nil, 401)
	f.request("POST", "/v1/auth/reset-password", "", map[string]any{"token": reset, "password": "another long password"}, 422)
	tok, e = f.as.Login(ctx, "user@example.test", "a new long password", "token", "")
	if e != nil {
		t.Fatal(e)
	}
	f.request("POST", "/v1/auth/logout", "", map[string]any{"refresh_token": tok.RefreshToken}, 204)
	f.request("GET", "/v1/me", tok.AccessToken, nil, 401)
	f.mail.fail = true
	if e = f.as.Register(ctx, auth.RegisterInput{StudentNumber: "MAILFAIL", Combination: "PMT-CS", FirstName: "A", LastName: "B", DisplayName: "AB", Email: "failure@example.test", Password: "correct horse battery"}); e != nil {
		t.Fatal(e)
	}
	var status string
	if e = f.p.QueryRow(ctx, `SELECT status FROM users WHERE email='failure@example.test'`).Scan(&status); e != nil || status != "PENDING_VERIFICATION" {
		t.Fatalf("mail failure rollback %s %v", status, e)
	}
	if _, e = f.p.Exec(ctx, `INSERT INTO users(student_number,combination,first_name,last_name,display_name,email,password_hash) VALUES('BAD-COMBINATION','PMT-MATH','A','B','AB','bad-combination@example.test','hash')`); e == nil {
		t.Fatal("database accepted an unsupported student combination")
	}
	if _, e = f.p.Exec(ctx, `INSERT INTO users(student_number,first_name,last_name,display_name,email,password_hash) VALUES('MISSING-COMBINATION','A','B','AB','missing-db-combination@example.test','hash')`); e == nil {
		t.Fatal("database accepted a new user without a student combination")
	}
}
func TestTenantRolesAndContent(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	rep, repToken, _ := f.newUser("REP001")
	student, studentToken, _ := f.newUser("ST002")
	other, otherToken, _ := f.newUser("OTHER003")
	bid := f.newBatch(rep, "cohort-a")
	bid2 := f.newBatch(rep, "cohort-b")
	f.member(rep, bid, rep, "BATCH_REP")
	mid := f.member(rep, bid, student, "")
	otherMid := f.member(rep, bid2, other, "BATCH_REP")
	f.member(rep, bid2, student, "")
	base := "/v1/batches/" + bid
	access := object(t, f.request("GET", "/v1/me/access", studentToken, nil, 200))
	if memberships, ok := access["memberships"].([]any); !ok || len(memberships) != 2 {
		t.Fatalf("access memberships = %#v", access["memberships"])
	}
	candidates := f.request("GET", base+"/members/candidates?query=OTHER", repToken, nil, 200)
	if !bytes.Contains(candidates, []byte(other)) {
		t.Fatalf("candidate search did not return eligible account: %s", candidates)
	}
	f.request("GET", base+"/members/candidates?query=O", repToken, nil, 422)
	f.request("GET", base+"/members/candidates?query=OTHER", studentToken, nil, 403)
	summary := object(t, f.request("GET", base+"/summary", studentToken, nil, 200))
	if summary["members"].(float64) != 2 {
		t.Fatalf("summary members = %#v", summary["members"])
	}
	f.request("POST", base+"/members/"+mid+"/roles", studentToken, map[string]any{"role": "BATCH_REP"}, 403)
	f.request("POST", base+"/members/"+mid+"/roles", repToken, map[string]any{"role": "BATCH_REP"}, 200)
	f.request("POST", base+"/members/"+mid+"/roles", repToken, map[string]any{"role": "BATCH_REP"}, 200)
	var n int
	if e := f.p.QueryRow(ctx, `SELECT count(*) FROM membership_roles WHERE membership_id=$1`, mid).Scan(&n); e != nil || n != 2 {
		t.Fatalf("base role missing: %d %v", n, e)
	}
	f.request("POST", base+"/members/"+otherMid+"/roles", repToken, map[string]any{"role": "BATCH_REP"}, 404)
	semester := idOf(t, f.request("POST", base+"/semesters", studentToken, map[string]any{"semester_number": 1, "name": "Semester 1", "academic_year": "2026"}, 201))
	f.request("POST", "/v1/batches/"+bid2+"/semesters", studentToken, map[string]any{"semester_number": 1, "name": "No", "academic_year": "2026"}, 403)
	f.request("DELETE", base+"/members/"+mid+"/roles/BATCH_REP", repToken, nil, 204)
	f.request("POST", base+"/semesters", studentToken, map[string]any{"semester_number": 2, "name": "No", "academic_year": "2026"}, 403)
	f.request("DELETE", base+"/members/"+mid+"/roles/STUDENT", repToken, nil, 409)
	module := idOf(t, f.request("POST", base+"/modules", repToken, map[string]any{"semester_id": semester, "module_code": "CS101", "name": "Computing"}, 201))
	f.request("POST", "/v1/batches/"+bid2+"/modules", otherToken, map[string]any{"semester_id": semester, "module_code": "CS101", "name": "Cross batch"}, 409)
	for _, path := range []string{"semesters", "modules"} {
		f.request("GET", base+"/"+path, studentToken, nil, 200)
	}
	ann := idOf(t, f.request("POST", base+"/announcements", repToken, map[string]any{"module_id": module, "title": "Welcome", "body": "Announcement"}, 201))
	f.request("GET", base+"/announcements/"+ann, studentToken, nil, 404)
	f.request("GET", "/v1/batches/"+bid2+"/announcements/"+ann, otherToken, nil, 404)
	f.request("POST", base+"/announcements/"+ann+"/publish", repToken, nil, 204)
	f.request("POST", base+"/announcements/"+ann+"/publish", repToken, nil, 204)
	f.request("GET", base+"/announcements/"+ann, studentToken, nil, 200)
	if _, e := (notification.Service{Pool: f.p}).Dispatch(ctx); e != nil {
		t.Fatal(e)
	}
	f.request("GET", "/v1/me/notifications", studentToken, nil, 200)
	up := f.upload(base+"/resources/uploads", repToken, "notes.pdf", "application/pdf", []byte("%PDF-1.7\nnotes"))
	resource := idOf(t, f.request("POST", base+"/resources", repToken, map[string]any{"upload_id": up, "module_id": module, "type": "HANDWRITTEN_NOTE", "title": "Notes"}, 201))
	f.request("GET", base+"/resources/"+resource+"/download", studentToken, nil, 404)
	f.request("POST", base+"/resources/"+resource+"/publish", repToken, nil, 204)
	f.request("GET", "/v1/batches/"+bid2+"/resources/"+resource+"/download", otherToken, nil, 404)
	download := object(t, f.request("GET", base+"/resources/"+resource+"/download", studentToken, nil, 200))
	got := f.request("GET", download["url"].(string), "", nil, 200)
	if string(got) != "%PDF-1.7\nnotes" {
		t.Fatal("download bytes")
	}
	up = f.upload(base+"/resources/"+resource+"/versions/uploads", repToken, "v2.pdf", "application/pdf", []byte("%PDF-1.7\nversion2"))
	f.request("POST", base+"/resources/"+resource+"/versions", repToken, map[string]any{"upload_id": up, "change_note": "Second version"}, 204)
	f.request("GET", base+"/resources/"+resource+"/versions", studentToken, nil, 200)
	f.request("PUT", base+"/resources/"+resource+"/bookmark", studentToken, nil, 204)
	f.request("PUT", base+"/resources/"+resource+"/bookmark", studentToken, nil, 204)
	f.request("GET", "/v1/me/bookmarks", studentToken, nil, 200)
	up = f.upload(base+"/announcements/"+ann+"/attachments/uploads", repToken, "attachment.pdf", "application/pdf", []byte("%PDF-1.7\nattachment"))
	aid := idOf(t, f.request("POST", base+"/announcements/"+ann+"/attachments", repToken, map[string]any{"upload_id": up}, 201))
	f.request("GET", base+"/announcements/"+ann+"/attachments/"+aid+"/download", studentToken, nil, 200)
	for _, x := range []struct {
		path string
		body map[string]any
	}{{"lessons", map[string]any{"module_id": module, "title": "Lesson", "youtube_video_id": "dQw4w9WgXcQ"}}, {"links", map[string]any{"title": "Reference", "url": "https://example.com"}}, {"events", map[string]any{"title": "Orientation", "starts_at": "2026-09-20T10:00:00Z", "visibility": "PUBLIC"}}} {
		id := idOf(t, f.request("POST", base+"/"+x.path, repToken, x.body, 201))
		f.request("GET", base+"/"+x.path+"/"+id, repToken, nil, 200)
		f.request("POST", base+"/"+x.path+"/"+id+"/publish", repToken, nil, 204)
		f.request("GET", base+"/"+x.path, studentToken, nil, 200)
		if x.path == "events" {
			f.request("GET", "/v1/public/batches/cohort-a/events", "", nil, 200)
		}
	}
	pos := idOf(t, f.request("POST", base+"/positions", repToken, map[string]any{"title": "President", "is_public": true}, 201))
	f.request("POST", base+"/positions/"+pos+"/assignments", repToken, map[string]any{"user_id": student, "starts_at": "2026-01-01T00:00:00Z"}, 201)
	f.request("POST", base+"/members/"+mid+"/roles", studentToken, map[string]any{"role": "BATCH_REP"}, 403)
	f.request("GET", "/v1/public/batches/cohort-a/positions", "", nil, 200)
	var auditCount int
	if e := f.p.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action IN ('ROLE_ASSIGNED','ROLE_REMOVED')`, mid).Scan(&auditCount); e != nil || auditCount != 2 {
		t.Fatalf("audit count %d %v", auditCount, e)
	}
}
func TestAnonymousComplaintsAndFeedback(t *testing.T) {
	f := setup(t)
	rep, repToken, _ := f.newUser("REP004")
	a, at, _ := f.newUser("ST005")
	b, bt, _ := f.newUser("ST006")
	bid := f.newBatch(rep, "support")
	f.member(rep, bid, rep, "BATCH_REP")
	am := f.member(rep, bid, a, "")
	f.member(rep, bid, b, "")
	base := "/v1/batches/" + bid
	id := idOf(t, f.request("POST", base+"/complaints", at, map[string]any{"category": "academic", "subject": "Concern", "message": "Private complaint", "is_anonymous": true}, 201))
	var owner *string
	if e := f.p.QueryRow(context.Background(), `SELECT submitted_by FROM complaints WHERE id=$1`, id).Scan(&owner); e != nil || owner != nil {
		t.Fatalf("anonymity violated %v %v", owner, e)
	}
	f.request("GET", base+"/complaints/"+id, at, nil, 404)
	f.request("GET", base+"/complaints/"+id, bt, nil, 404)
	f.request("GET", base+"/complaints", repToken, nil, 403)
	f.request("POST", base+"/members/"+am+"/roles", repToken, map[string]any{"role": "COMPLAINT_MANAGER"}, 200)
	f.request("GET", base+"/complaints/"+id, at, nil, 200)
	f.request("PATCH", base+"/complaints/"+id+"/status", at, map[string]any{"status": "CLOSED"}, 409)
	for _, status := range []string{"IN_REVIEW", "RESOLVED", "CLOSED"} {
		f.request("PATCH", base+"/complaints/"+id+"/status", at, map[string]any{"status": status}, 204)
	}
	identified := idOf(t, f.request("POST", base+"/complaints", bt, map[string]any{"category": "academic", "subject": "Named", "message": "My complaint"}, 201))
	f.request("POST", base+"/complaints/"+identified+"/messages", bt, map[string]any{"message": "Follow-up"}, 204)
	f.request("GET", base+"/complaints/"+identified+"/messages", at, nil, 200)
	fid := idOf(t, f.request("POST", base+"/feedback", bt, map[string]any{"message": "Feedback", "is_anonymous": true, "rating": 4}, 201))
	if e := f.p.QueryRow(context.Background(), `SELECT submitted_by FROM feedback WHERE id=$1`, fid).Scan(&owner); e != nil || owner != nil {
		t.Fatal("feedback anonymity")
	}
	f.request("GET", base+"/feedback", repToken, nil, 200)
}
func TestGlobalGallery(t *testing.T) {
	f := setup(t)
	admin, token, _ := f.newUser("ADMIN007")
	rep, rt, _ := f.newUser("REP008")
	bid := f.newBatch(admin, "gallery-test")
	f.member(admin, bid, rep, "BATCH_REP")
	f.request("POST", "/v1/admin/gallery/uploads", rt, map[string]any{"file_name": "x.png", "mime_type": "image/png", "size_bytes": 100}, 403)
	f.sql(`INSERT INTO user_platform_roles(user_id,role_id) SELECT $1,id FROM roles WHERE code='PLATFORM_ADMIN'`, admin)
	var imageBytes bytes.Buffer
	if e := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 16, 12))); e != nil {
		t.Fatal(e)
	}
	var ids []string
	for i := 0; i < 3; i++ {
		display := f.upload("/v1/admin/gallery/uploads", token, "display.png", "image/png", imageBytes.Bytes())
		thumb := f.upload("/v1/admin/gallery/uploads", token, "thumb.png", "image/png", imageBytes.Bytes())
		id := idOf(t, f.request("POST", "/v1/admin/gallery", token, map[string]any{"title": "Photo", "alt_text": "Campus", "display_upload_id": display, "thumbnail_upload_id": thumb, "taken_at": fmt.Sprintf("2026-09-%02dT10:00:00Z", i+1)}, 201))
		ids = append(ids, id)
		f.request("GET", "/v1/public/gallery/"+id, "", nil, 404)
		f.request("GET", "/v1/public/gallery/"+id+"/files/display", "", nil, 404)
		f.request("POST", "/v1/admin/gallery/"+id+"/publish", token, nil, 204)
	}
	v := object(t, f.request("GET", "/v1/public/gallery?month=2026-09&limit=2", "", nil, 200))
	if len(v["data"].([]any)) != 2 || v["data"].([]any)[0].(map[string]any)["id"] != ids[2] {
		t.Fatal("gallery ordering")
	}
	cursor := v["next_cursor"].(string)
	next := object(t, f.request("GET", "/v1/public/gallery?month=2026-09&limit=2&cursor="+cursor, "", nil, 200))
	if len(next["data"].([]any)) != 1 {
		t.Fatal("pagination")
	}
	f.request("GET", "/v1/public/gallery?month=2026-08&cursor="+cursor, "", nil, 422)
	f.request("GET", "/v1/public/gallery?limit=21", "", nil, 422)
	f.request("GET", "/v1/public/gallery/"+ids[0]+"/files/display", "", nil, 200)
	f.request("DELETE", "/v1/admin/gallery/"+ids[0], token, nil, 204)
	f.request("GET", "/v1/public/gallery/"+ids[0]+"/files/display", "", nil, 404)
	var n int
	if e := f.p.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='gallery_images' AND column_name='batch_id'`).Scan(&n); e != nil || n != 0 {
		t.Fatal("gallery tenant column")
	}
}
func TestConcurrentRefresh(t *testing.T) {
	f := setup(t)
	_, _, refresh := f.newUser("STCONC")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := f.as.Refresh(context.Background(), refresh, "token"); results <- e }()
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("rotations %d", success)
	}
	var revoked bool
	if e := f.p.QueryRow(context.Background(), `SELECT revoked_at IS NOT NULL FROM auth_sessions LIMIT 1`).Scan(&revoked); e != nil || !revoked {
		t.Fatalf("replay not revoked %v", e)
	}
}
func TestExpiredTokenAndCookieCSRF(t *testing.T) {
	f := setup(t)
	uid, token, _ := f.newUser("COOKIE01")
	f.sql(`UPDATE users SET status='SUSPENDED' WHERE id=$1`, uid)
	f.request("GET", "/v1/me", token, nil, 401)
	f.sql(`UPDATE users SET status='ACTIVE' WHERE id=$1`, uid)
	body := `{"email":"COOKIE01@example.test","password":"correct horse battery","transport":"cookie"}`
	for _, origin := range []string{"", "http://evil.test", "http://client.test"} {
		r := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		f.h.ServeHTTP(w, r)
		if origin == "http://client.test" {
			if w.Code != 200 || len(w.Result().Cookies()) != 1 {
				t.Fatalf("cookie login %d %s", w.Code, w.Body.String())
			}
			if bytes.Contains(w.Body.Bytes(), []byte("refresh_token")) {
				t.Fatal("cookie token exposed")
			}
			cookie := w.Result().Cookies()[0]
			if !cookie.HttpOnly {
				t.Fatal("not HttpOnly")
			}
			r = httptest.NewRequest("POST", "/v1/auth/refresh", nil)
			r.AddCookie(cookie)
			w = httptest.NewRecorder()
			f.h.ServeHTTP(w, r)
			if w.Code != 403 {
				t.Fatal("missing origin accepted")
			}
		} else if w.Code != 403 {
			t.Fatal("bad origin accepted")
		}
	}
	if e := f.as.RequestToken(context.Background(), "cookie01@example.test", "PASSWORD_RESET"); e != nil {
		t.Fatal(e)
	}
	raw := f.mail.token("cookie01@example.test", "PASSWORD_RESET")
	f.sql(`UPDATE verification_tokens SET expires_at=now()-interval '1 second' WHERE token_hash=$1`, auth.Hash(raw))
	if e := f.as.ConsumeToken(context.Background(), raw, "PASSWORD_RESET", "another long password"); e == nil {
		t.Fatal("expired reset")
	}
}

var _ = time.Second

func TestImagesAndPartialUpdates(t *testing.T) {
	f := setup(t)
	uid, token, _ := f.newUser("IMAGES01")
	bid := f.newBatch(uid, "image-cohort")
	f.member(uid, bid, uid, "BATCH_REP")
	base := "/v1/batches/" + bid
	var imageBytes bytes.Buffer
	if e := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 10, 10))); e != nil {
		t.Fatal(e)
	}
	up := f.upload("/v1/me/profile/image/uploads", token, "profile.png", "image/png", imageBytes.Bytes())
	f.request("PATCH", "/v1/me/profile", token, map[string]any{"upload_id": up}, 204)
	f.request("GET", "/v1/me/profile/image", token, nil, 200)
	up = f.upload(base+"/profile/uploads", token, "hero.png", "image/png", imageBytes.Bytes())
	f.request("PATCH", base+"/profile", token, map[string]any{"headline": "Our cohort", "about_text": "About us", "upload_id": up}, 204)
	f.request("PATCH", base+"/profile", token, map[string]any{"mission_text": "Learn"}, 204)
	profile := object(t, f.request("GET", base+"/profile", token, nil, 200))
	if profile["headline"] != "Our cohort" || profile["about_text"] != "About us" {
		t.Fatal("PATCH erased fields")
	}
	f.request("GET", "/v1/public/batches/image-cohort/image", "", nil, 200)
	id := idOf(t, f.request("POST", base+"/events", token, map[string]any{"title": "Private event", "starts_at": "2026-09-01T10:00:00Z", "visibility": "PUBLIC"}, 201))
	up = f.upload(base+"/events/"+id+"/uploads", token, "cover.png", "image/png", imageBytes.Bytes())
	f.request("POST", base+"/events/"+id+"/image", token, map[string]any{"upload_id": up}, 204)
	f.request("GET", "/v1/public/batches/image-cohort/events/"+id+"/image", "", nil, 404)
	f.request("POST", base+"/events/"+id+"/publish", token, nil, 204)
	f.request("GET", "/v1/public/batches/image-cohort/events/"+id+"/image", "", nil, 200)
	pos := idOf(t, f.request("POST", base+"/positions", token, map[string]any{"title": "Secretary", "is_public": true}, 201))
	f.request("PATCH", base+"/positions/"+pos, token, map[string]any{"description": "Committee"}, 204)
	p := object(t, f.request("GET", base+"/positions/"+pos, token, nil, 200))
	if p["title"] != "Secretary" || p["is_public"] != true {
		t.Fatal("position PATCH erased fields")
	}
}
func TestConstraintsAndSessionRevocation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	uid, token, _ := f.newUser("GUARDS01")
	bid := f.newBatch(uid, "guards")
	if _, e := f.p.Exec(ctx, `INSERT INTO batch_memberships(batch_id,user_id) VALUES($1,$2)`, bid, uid); e == nil {
		t.Fatal("active membership without STUDENT committed")
	}
	mid := f.member(uid, bid, uid, "BATCH_REP")
	if _, e := f.p.Exec(ctx, `DELETE FROM membership_roles WHERE membership_id=$1 AND role_id=(SELECT id FROM roles WHERE code='STUDENT')`, mid); e == nil {
		t.Fatal("base role removal committed")
	}
	if _, e := f.p.Exec(ctx, `UPDATE audit_logs SET action='TAMPER'`); e == nil {
		t.Fatal("audit history mutable")
	}
	second, e := f.as.Login(ctx, "guards01@example.test", "correct horse battery", "token", "")
	if e != nil {
		t.Fatal(e)
	}
	f.request("DELETE", "/v1/me/sessions/"+second.SessionID, token, nil, 204)
	f.request("GET", "/v1/me", second.AccessToken, nil, 401)
	third, e := f.as.Login(ctx, "guards01@example.test", "correct horse battery", "token", "")
	if e != nil {
		t.Fatal(e)
	}
	f.request("POST", "/v1/auth/logout-all", token, nil, 204)
	f.request("GET", "/v1/me", third.AccessToken, nil, 401)
}

func TestMigrationRoundTrip(t *testing.T) {
	f := setup(t)
	files, e := filepath.Glob("../../migrations/*.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	if len(files) == 0 {
		t.Fatal("no down migrations found")
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	for _, path := range files {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.p.Exec(context.Background(), string(b)); e != nil {
			t.Fatalf("down migration %s: %v", path, e)
		}
	}
	files, e = filepath.Glob("../../migrations/*.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	if len(files) == 0 {
		t.Fatal("no up migrations found")
	}
	sort.Strings(files)
	for _, path := range files {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.p.Exec(context.Background(), string(b)); e != nil {
			t.Fatalf("reapply %s: %v", path, e)
		}
	}
}

func TestSeedPlatformAdminMigrationMapsUserAndRoleIDs(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	var userID string
	if e := f.p.QueryRow(ctx, `
		INSERT INTO users(
			student_number,combination,first_name,last_name,display_name,email,
			password_hash,email_verified_at,status
		)
		VALUES('as123123','PMT-CS','Platform','Admin','Platform Admin',
			'test@example.com','unused',now(),'ACTIVE')
		RETURNING id
	`).Scan(&userID); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("../../migrations/000011_seed_platform_admin.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.p.Exec(ctx, string(b)); e != nil {
		t.Fatal(e)
	}
	var assignedUserID, assignedRoleID, scope string
	if e = f.p.QueryRow(ctx, `
		SELECT upr.user_id,upr.role_id,upr.scope
		FROM user_platform_roles upr
		JOIN roles r ON (r.id,r.scope)=(upr.role_id,upr.scope)
		WHERE upr.user_id=$1 AND r.code='PLATFORM_ADMIN'
	`, userID).Scan(&assignedUserID, &assignedRoleID, &scope); e != nil {
		t.Fatal(e)
	}
	if assignedUserID != userID || assignedRoleID == userID || scope != "PLATFORM" {
		t.Fatalf("invalid platform role mapping: user=%s role=%s scope=%s", assignedUserID, assignedRoleID, scope)
	}
}

func TestExpiredVerificationAndInactiveAccounts(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.as.Register(ctx, auth.RegisterInput{StudentNumber: "EXPIRED", Combination: "PMT-CS", FirstName: "A", LastName: "B", DisplayName: "AB", Email: "expired@example.test", Password: "correct horse battery"}); e != nil {
		t.Fatal(e)
	}
	raw := f.mail.token("expired@example.test", "EMAIL_VERIFY")
	f.sql(`UPDATE verification_tokens SET expires_at=now()-interval '1 second' WHERE token_hash=$1`, auth.Hash(raw))
	if e := f.as.ConsumeToken(ctx, raw, "EMAIL_VERIFY", ""); e == nil {
		t.Fatal("expired verification accepted")
	}
	uid, token, _ := f.newUser("INACTIVE")
	for _, status := range []string{"SUSPENDED", "ARCHIVED"} {
		f.sql(`UPDATE users SET status=$2 WHERE id=$1`, uid, status)
		if _, e := f.as.Login(ctx, "inactive@example.test", "correct horse battery", "token", ""); e == nil {
			t.Fatal("inactive login")
		}
		if _, e := f.as.Authenticate(ctx, token); e == nil {
			t.Fatal("inactive access")
		}
	}
}
