# PMT_WEB - Codebase & Architecture Analysis

## 1. Backend Architecture & Routes
The backend is built in **Go (Golang)** following a clean, domain-driven directory layout.

- **Entrypoints (`cmd/`)**:
  - `api`: The main HTTP server.
  - `admin`: CLI tool for bootstrapping verified users/admins.
  - `jobs`: Background job runner.
  - `migrate`: Database migration runner.
- **Domain Logic (`internal/`)**: Modularized into domain packages such as `auth`, `user`, `batch`, `lesson`, `gallery`, `student`, `module`, `announcement`, `complaint`, etc.
- **Routing (`internal/httpapi`)**: Utilizes the `chi` router with a robust set of middleware (CORS, Rate Limiting, Authentication). 
  - **Key Routes**: `AuthRoutes`, `AdminRoutes`, `FileRoutes`, `GalleryAdminRoutes`, `SemesterRoutes`, `ModuleRoutes`, `LessonRoutes`, `EventRoutes`, `SupportRoutes`.
  - **Health Checks**: Included under `/health/live` and `/health/ready`.

## 2. Database Schema
The database is **PostgreSQL**, exclusively managed by raw SQL migration files (`migrations/`).

- **Core Identity Tables**: 
  - `users`: Core identity (student number, email, password hash, status).
  - `auth_sessions` & `spent_refresh_tokens`: Track JWT refresh sessions and device info.
  - `verification_tokens`: Handles email verification and password resets.
- **Organization & RBAC (Role-Based Access Control)**:
  - `batches`: Cohorts/Classes of students.
  - `batch_memberships`: Links users to batches.
  - Highly normalized RBAC system using `roles`, `permissions`, `role_permissions`, `membership_roles` (Batch scope), and `user_platform_roles` (Platform scope).
  - Predefined roles: `STUDENT`, `BATCH_REP`, `CONTENT_MANAGER`, `ACADEMIC_REP`, `COMPLAINT_MANAGER`, and `PLATFORM_ADMIN`.
- **Domain Tables**: Separate files define tables for resources, portfolios, galleries, notifications, and student combinations.
- **Audit Logging**: Handled via an immutable `audit_logs` table strictly enforced by PL/pgSQL database triggers (`protect_audit`).

## 3. Frontend Setup
The frontend is a modern Single Page Application (SPA) built with **React 19, TypeScript, and Vite**, stylized using **Tailwind CSS v4**.

- **State Management & Routing**: Utilizes `zustand` for global state (e.g., authentication) and `react-router-dom` v7 for routing.
- **Project Structure**: 
  - `src/pages/`: Contains all route components (Admin/Student Dashboards, Courses, Profiles, etc.).
  - `src/components/`: Reusable UI elements, including distinct layout wrappers (`Layout.tsx` for Admin, `StudentLayout.tsx` for Students).
  - `src/api/`: Handles `axios`-based HTTP requests to the backend.
- **Current UI Pages**: Login, Register, Verify, PasswordReset, Admin/Student Dashboards, Courses, CourseDetail, Assessments, Communication, Meetings, Monitoring, System, Administration, Gallery, Profile, Notifications.

## 4. Authentication Flow
Authentication is handled seamlessly via **JWT (JSON Web Tokens)** and database-backed sessions.

- **Backend**: Generates short-lived access tokens and long-lived refresh tokens. Refresh tokens are tracked in `auth_sessions` for revocation capabilities, and `spent_refresh_tokens` for replay protection.
- **Frontend**: The `useAuthStore` (zustand) keeps track of `isAuthenticated`. Protected routes are wrapped in `<RequireAuth>` (requires login) and `<RequireAdmin>` (requires elevated platform role).
- **Redirection Logic**: In `App.tsx`, a `<Landing>` component dynamically redirects logged-in users to either `/admin/dashboard` or `/student/dashboard` based on their role via the `useElevated()` hook.

## 5. Existing vs Missing Features

**✅ Currently Functional (Implemented):**
- **Authentication & Security**: Registration, Login, Email Verification, Password Reset, and secure JWT handling.
- **RBAC & Hierarchy**: Complex role assignments (Platform vs Batch scope) backed by strict database constraints.
- **Core Domain Scaffolding**: Extensive Go backend API endpoints and DB schemas for batches, semesters, modules, lessons, events, resources, announcements, and complaints.
- **Frontend Foundation**: Routing shell, Tailwind theming, state management, and distinct portal layouts for Admins vs Students.

**🚧 Missing / Next Steps (To be implemented):**
- **Frontend-Backend Integration**: Many frontend pages (like Assessments, Communication, Meetings, System Administration) exist as structural shells but need to be fully wired up to their corresponding backend endpoints.
- **Legacy Routes Cleanup**: The frontend `App.tsx` contains a `LEGACY` array redirecting older generic paths (e.g., `/users`, `/assessments`) to `/`, indicating an incomplete migration to the new Admin/Student split structure.
- **Background Jobs**: `cmd/jobs` exists, but queue/worker implementation needs expansion for robust asynchronous tasks like email notifications or data cleanup.
- **Testing**: End-to-end tests and extensive frontend unit tests seem minimal/missing and should be considered before large feature pushes.
