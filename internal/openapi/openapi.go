// Package openapi derives the endpoint inventory from the real Chi router.
// Explicit request schemas are attached here to keep wire contracts reviewable.
package openapi

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/announcement"
	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/complaint"
	"github.com/Hasras-code/PMT_WEB.git/internal/event"
	"github.com/Hasras-code/PMT_WEB.git/internal/feedback"
	"github.com/Hasras-code/PMT_WEB.git/internal/fund"
	"github.com/Hasras-code/PMT_WEB.git/internal/gallery"
	"github.com/Hasras-code/PMT_WEB.git/internal/kuppi"
	"github.com/Hasras-code/PMT_WEB.git/internal/link"
	"github.com/Hasras-code/PMT_WEB.git/internal/module"
	"github.com/Hasras-code/PMT_WEB.git/internal/position"
	"github.com/Hasras-code/PMT_WEB.git/internal/resource"
	"github.com/Hasras-code/PMT_WEB.git/internal/semester"
	"github.com/Hasras-code/PMT_WEB.git/internal/student"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
)

type M = map[string]any

func schema(t reflect.Type) M {
	if t.Kind() == reflect.Pointer {
		return schema(t.Elem())
	}
	if t == reflect.TypeFor[time.Time]() {
		return M{"type": "string", "format": "date-time"}
	}
	if t == reflect.TypeFor[student.Combination]() {
		return M{"type": "string", "enum": student.CombinationValues()}
	}
	switch t.Kind() {
	case reflect.String:
		return M{"type": "string"}
	case reflect.Bool:
		return M{"type": "boolean"}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return M{"type": "integer"}
	case reflect.Slice:
		return M{"type": "array", "items": schema(t.Elem())}
	case reflect.Struct:
		props := M{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" || name == "" {
				continue
			}
			props[name] = schema(f.Type)
		}
		return M{"type": "object", "additionalProperties": false, "properties": props}
	default:
		return M{"type": "object"}
	}
}
func body(fields ...string) M {
	p := M{}
	for _, f := range fields {
		p[f] = M{"type": "string"}
	}
	return M{"type": "object", "additionalProperties": false, "properties": p}
}

func operationMetadata(method, path string) (string, string, string) {
	tag := "System"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, part := range parts {
		switch part {
		case "auth":
			tag = "Authentication"
		case "admin":
			tag = "Administration"
		case "batches":
			tag = "Batches"
		case "gallery":
			tag = "Gallery"
		case "funds", "fund-transfers", "birthday-fund":
			tag = "Funds"
		case "notifications":
			tag = "Notifications"
		case "complaints", "feedback":
			tag = "Support"
		case "me":
			tag = "Profile"
		}
	}

	name := strings.Trim(parts[len(parts)-1], "{}")
	name = strings.ReplaceAll(name, "-", " ")
	if name == "" {
		name = "resource"
	}
	summary := strings.ToUpper(method[:1]) + strings.ToLower(method[1:]) + " " + name
	description := "Manage " + name + "."
	switch path {
	case "/v1/auth/register":
		summary, description = "Register a user", "Create a pending account for a selected cohort and send an email verification message."
	case "/v1/auth/login":
		summary, description = "Log in", "Authenticate with email and password. Use token transport for JSON refresh credentials or cookie transport for browser sessions."
	case "/v1/auth/refresh":
		summary, description = "Refresh an access token", "Rotate the refresh credential and issue a new access token."
	case "/v1/auth/logout":
		summary, description = "Log out", "Revoke the current refresh session."
	case "/v1/auth/verify-email":
		summary, description = "Verify an email address", "Activate a pending account and add it to its selected cohort as a student."
	case "/v1/auth/reset-password":
		summary, description = "Reset a password", "Set a new password using a password reset token."
	}
	return tag, summary, description
}

func Request(method, path string) M {
	if method != "POST" && method != "PATCH" {
		return nil
	}
	if strings.HasSuffix(path, "/uploads") {
		return schema(reflect.TypeFor[upload.Input]())
	}
	if strings.Contains(path, "/funds") || strings.Contains(path, "/fund-transfers") || strings.Contains(path, "/birthday-fund") {
		switch {
		case strings.HasSuffix(path, "/reverse"), strings.HasSuffix(path, "/post"), strings.HasSuffix(path, "/close"), strings.HasSuffix(path, "/archive"):
			return nil
		case strings.HasSuffix(path, "/repayments"):
			return schema(reflect.TypeFor[fund.RepaymentInput]())
		case strings.HasSuffix(path, "/payments"):
			return schema(reflect.TypeFor[fund.PaymentInput]())
		case strings.HasSuffix(path, "/managers"):
			return schema(reflect.TypeFor[fund.ManagerInput]())
		case strings.HasSuffix(path, "/attachments"):
			return schema(reflect.TypeFor[fund.AttachmentInput]())
		case strings.HasSuffix(path, "/transactions"):
			return schema(reflect.TypeFor[fund.TransactionInput]())
		case strings.Contains(path, "/transactions/{transactionID}") && method == "PATCH":
			return schema(reflect.TypeFor[fund.TransactionUpdate]())
		case strings.HasSuffix(path, "/fund-transfers"):
			return schema(reflect.TypeFor[fund.TransferInput]())
		case strings.HasSuffix(path, "/birthday-fund/periods"):
			return schema(reflect.TypeFor[fund.PeriodInput]())
		case strings.HasSuffix(path, "/funds"):
			return schema(reflect.TypeFor[fund.FundInput]())
		case strings.Contains(path, "/funds/{fundID}") && method == "PATCH":
			return schema(reflect.TypeFor[fund.FundUpdate]())
		}
	}
	if strings.Contains(path, "/auth/") {
		switch path {
		case "/v1/auth/register":
			return schema(reflect.TypeFor[auth.RegisterInput]())
		case "/v1/auth/login":
			return body("email", "password", "transport")
		case "/v1/auth/refresh", "/v1/auth/logout":
			return body("refresh_token")
		case "/v1/auth/verify-email":
			return body("token")
		case "/v1/auth/reset-password":
			return body("token", "password")
		case "/v1/auth/forgot-password", "/v1/auth/resend-verification":
			return body("email")
		}
		return nil
	}
	if strings.HasSuffix(path, "/publish") || strings.HasSuffix(path, "/archive") || strings.HasSuffix(path, "/suspend") || strings.HasSuffix(path, "/reactivate") || strings.HasSuffix(path, "/graduate") || strings.HasSuffix(path, "/leave") || strings.HasSuffix(path, "/end") || strings.HasSuffix(path, "/set-current") || strings.HasSuffix(path, "/read-all") {
		return nil
	}
	if path == "/v1/me/profile" {
		return schema(reflect.TypeFor[user.Profile]())
	}
	if strings.Contains(path, "/notifications") {
		return nil
	}
	if path == "/v1/admin/batches" {
		return schema(reflect.TypeFor[batch.Input]())
	}
	if strings.HasSuffix(path, "/profile") {
		return schema(reflect.TypeFor[batch.Profile]())
	}
	if strings.HasSuffix(path, "/roles") {
		return body("role")
	}
	if strings.HasSuffix(path, "/members") {
		return body("user_id")
	}
	if strings.HasSuffix(path, "/messages") {
		return body("message")
	}
	if strings.HasSuffix(path, "/status") {
		return body("status")
	}
	if strings.HasSuffix(path, "/assignee") {
		return body("user_id")
	}
	if strings.HasSuffix(path, "/assignments") {
		return M{"type": "object", "additionalProperties": false, "required": []string{"user_id", "starts_at"}, "properties": M{"user_id": M{"type": "string", "format": "uuid"}, "starts_at": M{"type": "string", "format": "date-time"}, "ends_at": M{"type": "string", "format": "date-time", "nullable": true}}}
	}
	if strings.HasSuffix(path, "/image") && method == "POST" {
		return body("upload_id")
	}
	if strings.HasSuffix(path, "/attachments") {
		return body("upload_id")
	}
	switch {
	case strings.Contains(path, "/gallery"):
		if method == "PATCH" {
			return schema(reflect.TypeFor[gallery.Update]())
		}
		return schema(reflect.TypeFor[gallery.Input]())
	case strings.Contains(path, "/semesters"):
		return schema(reflect.TypeFor[semester.Input]())
	case strings.Contains(path, "/modules"):
		return schema(reflect.TypeFor[module.Input]())
	case strings.Contains(path, "/announcements"):
		return schema(reflect.TypeFor[announcement.Input]())
	case strings.Contains(path, "/resources"):
		if method == "PATCH" {
			return schema(reflect.TypeFor[resource.Update]())
		}
		return schema(reflect.TypeFor[resource.Input]())
	case strings.Contains(path, "/kuppis"):
		return schema(reflect.TypeFor[kuppi.Input]())
	case strings.Contains(path, "/links"):
		return schema(reflect.TypeFor[link.Input]())
	case strings.Contains(path, "/events"):
		return schema(reflect.TypeFor[event.Input]())
	case strings.Contains(path, "/positions"):
		return schema(reflect.TypeFor[position.Input]())
	case strings.Contains(path, "/complaints"):
		return schema(reflect.TypeFor[complaint.Input]())
	case strings.Contains(path, "/feedback"):
		return schema(reflect.TypeFor[feedback.Input]())
	case strings.HasSuffix(path, "{batchID}"):
		return body("name", "description")
	}
	return nil
}
func Generate(r chi.Routes) (M, error) {
	paths := M{}
	e := chi.Walk(r, func(method, path string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// Documentation assets are not application API operations.
		if path == "/swagger" || strings.HasPrefix(path, "/swagger/") {
			return nil
		}
		path = strings.TrimSuffix(path, "/")
		tag, summary, description := operationMetadata(method, path)
		op := M{"tags": []string{tag}, "summary": summary, "description": description, "operationId": strings.ToLower(method) + regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(path, "_"), "responses": M{}}
		params := []any{}
		for _, m := range regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(path, -1) {
			s := M{"type": "string"}
			if strings.HasSuffix(m[1], "ID") {
				s["format"] = "uuid"
			}
			params = append(params, M{"name": m[1], "in": "path", "required": true, "schema": s})
		}
		public := strings.HasPrefix(path, "/health/") || strings.Contains(path, "/public/") || strings.Contains(path, "/files/") || (strings.HasPrefix(path, "/v1/auth/") && path != "/v1/auth/logout-all")
		if !public {
			op["security"] = []any{M{"bearerAuth": []string{}}}
		}
		if path == "/v1/auth/refresh" || path == "/v1/auth/logout" {
			op["description"] = "Provide exactly one refresh credential: HttpOnly cookie with allowlisted Origin, or JSON refresh_token. Cookie sessions cannot switch transport."
			op["security"] = []any{M{"refreshCookie": []string{}}, M{}}
		}
		list := method == "GET" && !strings.HasSuffix(path, "}") && !strings.Contains(path, "/download") && !strings.Contains(path, "/files/") && !strings.HasSuffix(path, "/image") && !strings.HasPrefix(path, "/health/") && path != "/v1/me" && !strings.HasSuffix(path, "/profile")
		if list {
			max, def := 100, 20
			if path == "/v1/public/gallery" {
				max, def = 20, 12
			}
			params = append(params, M{"name": "limit", "in": "query", "schema": M{"type": "integer", "minimum": 1, "maximum": max, "default": def}})
			if path == "/v1/public/gallery" {
				for _, n := range []string{"cursor", "month"} {
					params = append(params, M{"name": n, "in": "query", "schema": M{"type": "string"}})
				}
			} else {
				params = append(params, M{"name": "offset", "in": "query", "schema": M{"type": "integer", "minimum": 0, "maximum": 100000, "default": 0}})
			}
		}
		if method == "GET" && path == "/v1/batches/{batchID}/resources" {
			params = append(params,
				M{"name": "module_id", "in": "query", "schema": M{"type": "string", "format": "uuid"}},
				M{"name": "type", "in": "query", "schema": M{"type": "string", "enum": []string{"LECTURE_NOTE", "PAST_PAPER", "ASSIGNMENT", "REFERENCE"}}},
				M{"name": "query", "in": "query", "schema": M{"type": "string", "maxLength": 200}},
			)
		}
		if method == "GET" && path == "/v1/batches/{batchID}/kuppis" {
			params = append(params,
				M{"name": "module_id", "in": "query", "schema": M{"type": "string", "format": "uuid"}},
				M{"name": "search", "in": "query", "schema": M{"type": "string", "maxLength": 200}},
				M{"name": "date_from", "in": "query", "schema": M{"type": "string", "format": "date-time"}},
				M{"name": "date_to", "in": "query", "schema": M{"type": "string", "format": "date-time"}},
				M{"name": "include_archived", "in": "query", "schema": M{"type": "boolean", "default": false}},
			)
		}
		if method == "GET" && strings.HasSuffix(path, "/funds/{fundID}/transactions") {
			params = append(params,
				M{"name": "type", "in": "query", "schema": M{"type": "string", "enum": []string{"CASH_IN", "EXPENSE", "TRANSFER_IN", "TRANSFER_OUT", "REVERSAL_IN", "REVERSAL_OUT"}}},
				M{"name": "source_type", "in": "query", "schema": M{"type": "string"}},
				M{"name": "date_from", "in": "query", "schema": M{"type": "string", "format": "date-time"}},
				M{"name": "date_to", "in": "query", "schema": M{"type": "string", "format": "date-time"}},
				M{"name": "include_drafts", "in": "query", "schema": M{"type": "boolean", "default": false}},
			)
		}
		if len(params) > 0 {
			op["parameters"] = params
		}
		req := Request(method, path)
		if req != nil {
			if required := requiredFields(method, path); len(required) > 0 {
				req["required"] = required
			}
		}
		if req != nil {
			op["requestBody"] = M{"required": !(path == "/v1/auth/refresh" || path == "/v1/auth/logout"), "content": M{"application/json": M{"schema": req}}}
		}
		status := "200"
		resp := M{"description": "Success"}
		if method == "DELETE" || method == "PATCH" || method == "PUT" || (method == "POST" && req == nil) {
			status = "204"
		}
		if method == "POST" && req != nil {
			status = "201"
		}
		if path == "/v1/auth/login" || path == "/v1/auth/refresh" || strings.HasSuffix(path, "/roles") {
			status = "200"
		}
		if strings.HasSuffix(path, "/verify-email") || strings.HasSuffix(path, "/reset-password") || strings.HasSuffix(path, "/logout") || strings.HasSuffix(path, "/messages") || (strings.HasSuffix(path, "/image") && method == "POST") || strings.HasSuffix(path, "/versions") && method == "POST" {
			status = "204"
		}
		if path == "/v1/auth/register" || path == "/v1/auth/forgot-password" || path == "/v1/auth/resend-verification" {
			status = "202"
		}
		if status != "204" {
			rs := M{"type": "object"}
			if list {
				rs = M{"type": "array", "items": M{"type": "object"}}
			}
			if status == "201" {
				rs = M{"type": "object", "properties": M{"id": M{"type": "string", "format": "uuid"}}}
			}
			if path == "/v1/public/gallery" {
				rs = schema(reflect.TypeFor[gallery.Page]())
			}
			if path == "/v1/public/gallery/{imageID}" {
				rs = schema(reflect.TypeFor[gallery.Image]())
			}
			if path == "/v1/me" {
				rs = schema(reflect.TypeFor[auth.User]())
			}
			if path == "/v1/auth/login" || path == "/v1/auth/refresh" {
				rs = schema(reflect.TypeFor[auth.Tokens]())
			}
			if strings.HasSuffix(path, "/uploads") {
				rs = M{"type": "object", "properties": M{"upload_id": M{"type": "string", "format": "uuid"}, "upload_url": M{"type": "string", "format": "uri"}, "method": M{"type": "string", "enum": []string{"PUT"}}, "headers": M{"type": "object", "additionalProperties": M{"type": "string"}}, "expires_in": M{"type": "integer"}}}
			}
			if strings.HasSuffix(path, "/download") {
				rs = M{"type": "object", "properties": M{"url": M{"type": "string", "format": "uri"}, "expires_in": M{"type": "integer"}}}
			}
			resp["content"] = M{"application/json": M{"schema": rs}}
		}
		if strings.Contains(path, "/files/uploads/") {
			op["requestBody"] = M{"required": true, "content": M{"application/pdf": M{"schema": M{"type": "string", "format": "binary"}}, "image/png": M{"schema": M{"type": "string", "format": "binary"}}, "image/jpeg": M{"schema": M{"type": "string", "format": "binary"}}, "image/webp": M{"schema": M{"type": "string", "format": "binary"}}}}
		}
		if method == "GET" && (strings.Contains(path, "/files/") || (strings.Contains(path, "/public/") && strings.HasSuffix(path, "/image"))) {
			resp["content"] = M{"application/octet-stream": M{"schema": M{"type": "string", "format": "binary"}}}
		}
		responses := op["responses"].(M)
		responses[status] = resp
		for _, code := range []string{"400", "401", "403", "404", "409", "413", "422", "429", "500"} {
			responses[code] = M{"description": http.StatusText(atoi(code)), "content": M{"application/json": M{"schema": M{"$ref": "#/components/schemas/Error"}}}}
		}
		entry, ok := paths[path].(M)
		if !ok {
			entry = M{}
			paths[path] = entry
		}
		entry[strings.ToLower(method)] = op
		return nil
	})
	if e != nil {
		return nil, e
	}
	return M{"openapi": "3.0.3", "info": M{"title": "PMT University LMS API", "version": "1.0.0", "description": "Backend-only modular monolith. Local PostgreSQL and immutable local PDF/image storage. All private batch operations resolve current database permissions. See README and docs/security.md for lifecycle and permission details."}, "servers": []M{{"url": "http://localhost:8080"}}, "paths": paths, "components": M{"securitySchemes": M{"bearerAuth": M{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}, "refreshCookie": M{"type": "apiKey", "in": "cookie", "name": "refresh_token"}}, "schemas": M{"Error": M{"type": "object", "required": []string{"error"}, "properties": M{"error": M{"type": "object", "required": []string{"code", "message", "request_id"}, "properties": M{"code": M{"type": "string"}, "message": M{"type": "string"}, "request_id": M{"type": "string"}}}}}}}}, nil
}
func atoi(s string) int { var n int; _, _ = fmt.Sscan(s, &n); return n }

func requiredFields(method, path string) []string {
	if method != "POST" {
		return nil
	}
	switch {
	case strings.HasSuffix(path, "/uploads"):
		return []string{"file_name", "mime_type", "size_bytes"}
	case path == "/v1/auth/register":
		return []string{"student_number", "combination", "batch_id", "first_name", "last_name", "display_name", "email", "password"}
	case path == "/v1/auth/login":
		return []string{"email", "password"}
	case path == "/v1/auth/verify-email":
		return []string{"token"}
	case path == "/v1/auth/reset-password":
		return []string{"token", "password"}
	case path == "/v1/auth/forgot-password" || path == "/v1/auth/resend-verification":
		return []string{"email"}
	case path == "/v1/admin/batches":
		return []string{"name", "slug", "entry_year"}
	case strings.HasSuffix(path, "/members"):
		return []string{"user_id"}
	case strings.HasSuffix(path, "/roles"):
		return []string{"role"}
	case strings.HasSuffix(path, "/fund-transfers"):
		return []string{"from_fund_id", "to_fund_id", "type", "amount_minor"}
	case strings.HasSuffix(path, "/repayments") || strings.HasSuffix(path, "/payments"):
		return []string{"amount_minor"}
	case strings.HasSuffix(path, "/managers"):
		return []string{"membership_id"}
	case strings.HasSuffix(path, "/birthday-fund/periods"):
		return []string{"year", "month", "amount_minor"}
	case strings.HasSuffix(path, "/funds"):
		return []string{"name", "type"}
	case strings.HasSuffix(path, "/transactions"):
		return []string{"type", "amount_minor", "description"}
	case strings.HasSuffix(path, "/semesters"):
		return []string{"semester_number", "name", "academic_year"}
	case strings.HasSuffix(path, "/modules"):
		return []string{"semester_id", "module_code", "name"}
	case strings.HasSuffix(path, "/announcements"):
		return []string{"title", "body"}
	case strings.HasSuffix(path, "/resources"):
		return []string{"upload_id", "module_id", "type", "title"}
	case strings.HasSuffix(path, "/versions") || strings.HasSuffix(path, "/attachments") || strings.HasSuffix(path, "/image"):
		return []string{"upload_id"}
	case strings.HasSuffix(path, "/kuppis"):
		return []string{"module_id", "title", "youtube_url"}
	case strings.HasSuffix(path, "/links"):
		return []string{"title", "url"}
	case strings.HasSuffix(path, "/events"):
		return []string{"title", "starts_at"}
	case strings.HasSuffix(path, "/positions"):
		return []string{"title"}
	case strings.HasSuffix(path, "/assignments"):
		return []string{"user_id", "starts_at"}
	case path == "/v1/admin/gallery":
		return []string{"display_upload_id", "thumbnail_upload_id", "alt_text"}
	case strings.HasSuffix(path, "/complaints"):
		return []string{"category", "subject", "message"}
	case strings.HasSuffix(path, "/feedback") || strings.HasSuffix(path, "/messages"):
		return []string{"message"}
	}
	return nil
}
