# Dental Laboratory Management System
## Implementation Plan, Software Architecture & Security Blueprint

**Version:** 1.0  
**Date:** 2026-10-23  
**Status:** Approved Discovery Baseline; Business Decisions Confirmed & Architecture Ready  
**Backend:** Go + Fiber + GORM  
**Database:** PostgreSQL  
**Frontend:** Web application (recommended: Next.js + React + TypeScript)  
**Deployment baseline:** Linux VPS, reverse proxy with HTTPS, systemd or containers

---

## 1. Executive Summary

Build a secure, maintainable Dental Laboratory Management System (DLMS) to manage inventory, denture production, sales and invoicing, customer relationships, staff/task assignment, reporting, online access, and backups.

The design uses a **modular monolith** initially: one Go API deployment organized by business domain, one PostgreSQL database, and background workers for notifications and scheduled jobs. This avoids the operational complexity of microservices while preserving clear module boundaries so components can be separated later if scale or team structure requires it.

### 1.1 Goals

- Track materials, products, stock movements, batches, expiry dates, suppliers, and minimum-stock alerts.
- Track a production order from intake through technical stages, QC, completion, and delivery.
- Link production orders to customers and invoices.
- Record payments, expenses, production costs, and sales reports.
- Maintain doctor/clinic profiles and order history.
- Assign work to technicians and measure operational performance.
- Provide role-based access, audit history, secure online access, and recoverable backups.
- Keep the system extensible for WhatsApp/email notifications, accounting integration, and QR/barcode workflows.

### 1.2 Scope from the supplied requirement

| Module | Initial priority | Included capabilities |
|---|---|---|
| Inventory & stock | High | Materials/products, suppliers, units, purchase/usage movements, stock thresholds, batches/expiry, valuation and usage reports |
| Production | High | Production order, doctor/patient reference, denture type, intake/due dates, stage tracking, technician PIC, status, QC photos/report |
| Sales & finance | High | Doctor/clinic customers, invoice from production, payment status, sales reports, expenses and production costs |
| Dashboard & reports | High | Production trends, top customers/products, costs/revenue, overdue production, low stock and overdue invoices |
| Online access & backup | High | Multi-user login, roles, web/mobile-responsive access, backup and restore plan |
| Security | Medium | Activity logs and role/permission controls; foundational security is mandatory from day one |
| Staff & tasks | Medium | Staff profiles, skills, task assignment, KPI, attendance/overtime as a later increment |
| Doctor/clinic CRM | Low | Preferences, notes, order/feedback history, reminders, loyalty/discount rules |
| Integrations | Low / optional | WhatsApp/email, accounting, barcode/QR |

The supplied requirement does not define detailed accounting rules, tax handling, multi-branch tenancy, exact production stages per product, or regulatory retention requirements. Treat these as discovery decisions, not assumed facts.

### 1.3 Non-goals for the first release

- Microservices or Kubernetes.
- Full double-entry accounting/general ledger unless separately approved.
- Automatic WhatsApp messaging before a provider and consent policy are chosen.
- Patient clinical-record management beyond the minimum production reference required.
- Multi-tenant SaaS isolation unless confirmed as a requirement.
- Automatic inventory valuation/accounting policy without owner approval.

---

## 2. Architecture Principles

1. **Modular monolith first.** Business modules have clear packages and interfaces.
2. **PostgreSQL is the source of truth.** Do not treat in-memory state or frontend values as authoritative.
3. **Explicit business transactions.** Inventory movement, production completion, invoice creation, and payment posting must be atomic where appropriate.
4. **Append-only financial and stock ledgers.** Correct errors with reversal/adjustment records rather than silently rewriting historical movements.
5. **Least privilege.** Users receive only the roles and permissions required for their jobs.
6. **Secure by default.** HTTPS, input validation, safe file uploads, secret management, rate limits, and audit logging are baseline controls.
7. **Migration-safe releases.** Expand schema first, deploy compatible code, backfill, then remove obsolete fields only in a later release.
8. **Observable and recoverable.** Structured logs, health checks, metrics, backups, and tested restore procedures.
9. **API-first.** Web UI and future mobile clients use the same versioned API.
10. **No secrets in source control.** Configuration comes from environment or a secret manager.

---

## 3. Proposed System Architecture

```mermaid
flowchart TB
    U[Owner / Admin / Technician / Cashier] --> HTTPS[HTTPS Reverse Proxy]
    HTTPS --> FE[Web UI: Next.js + React + TypeScript]
    FE --> API[Go Fiber REST API]
    API --> AUTH[Authentication & Authorization]
    API --> DOM[Domain Modules]
    DOM --> DB[(PostgreSQL)]
    DOM --> OBJ[Private Object Storage / Protected File Store]
    API --> JOBQ[PostgreSQL-backed Job Queue / Outbox]
    JOBQ --> WORKER[Background Worker]
    WORKER --> NOTIF[Email / WhatsApp Provider - optional]
    WORKER --> DB
    MON[Logs / Metrics / Alerts] --- API
    MON --- WORKER
    BACKUP[Encrypted Backup + Retention] --> DB
    BACKUP --> OBJ
```

### 3.1 Runtime components

- **Web frontend:** responsive user interface for desktop, tablet, and mobile browsers.
- **Go Fiber API:** validation, authentication, authorization, application services, and HTTP endpoints.
- **GORM:** persistence mapping and query composition. Use explicit SQL migrations as the authoritative schema evolution mechanism.
- **PostgreSQL:** transactional relational database, indexes, constraints, and ledger records.
- **Background worker:** scheduled reminders, notification delivery, report generation, and retryable jobs.
- **File storage:** private storage for QC photos and reports. Store metadata in PostgreSQL; store bytes in private object storage or a protected directory outside the web root.
- **Reverse proxy:** TLS termination, request size limits, security headers, and optional request throttling.
- **Backup process:** encrypted database and file backups, retention policy, off-site copy, and restore drills.

### 3.2 Deployment topology

**Small initial deployment**
- One Linux VPS.
- Reverse proxy (Nginx or Caddy) with HTTPS.
- Frontend and Go API as separate services or containers.
- PostgreSQL bound to localhost/private network only.
- Worker as a separate process using the same codebase.
- Private file directory or object storage.
- Encrypted backup copied to a separate storage location.

**Production hardening**
- Separate database host/managed PostgreSQL when operationally justified.
- Separate object storage for uploaded files.
- Monitoring and alerting independent of the application server.
- Staging environment with masked/synthetic data.
- Restrict administrative access through VPN or IP allowlisting where practical.

Do not expose PostgreSQL directly to the public internet. Do not store uploads under a public static directory.

---

## 4. Backend Technology and Dependency Policy

| Concern | Recommendation |
|---|---|
| Language | Go, current supported stable release pinned in CI |
| HTTP | Fiber v2 or current stable major; pin the selected major and review migration notes |
| ORM | GORM for normal persistence operations |
| Database | Supported PostgreSQL release |
| Schema migrations | Versioned SQL migrations using a dedicated migration tool; do not rely on AutoMigrate alone in production |
| Validation | Explicit request DTO validation; validate again in domain/application services |
| Authentication | Short-lived access token or secure server-side session; refresh/session revocation required |
| Password hashing | Argon2id or bcrypt with current recommended parameters |
| Logging | Structured JSON logs with request/correlation ID; redact secrets and personal data |
| API contract | OpenAPI 3.1, generated/validated in CI |
| Testing | Go unit tests, integration tests against real PostgreSQL, API tests, security tests |
| Observability | Health/readiness endpoints, metrics, centralized logs and alerting |
| Frontend (recommended) | Next.js + React + TypeScript; enforce server/client security boundaries |

Pin dependency versions, use `go.mod`/`go.sum`, run dependency vulnerability checks, and update dependencies on a defined schedule. Do not use unmaintained packages for authentication, cryptography, or file processing.

---

## 5. Domain Modules and Responsibilities

### 5.1 Identity and access

- User accounts, role assignment, permissions, session/token revocation.
- Password reset and account status.
- Optional MFA for owner/admin accounts.
- Login throttling and suspicious-login logging.

### 5.2 Customer / doctor / clinic CRM

- Doctor and clinic profile, contacts, address, preferences, notes.
- Customer order history and feedback.
- Discount policy with effective dates and approval constraints.
- Reminder scheduling; communication provider integration is optional.

### 5.3 Catalog and suppliers

- Material/product catalog, SKU/code, description, unit of measure.
- Supplier records and purchase references.
- Reorder/minimum stock thresholds.
- Active/inactive lifecycle; avoid deleting records referenced by transactions.

### 5.4 Inventory

- Stock locations if more than one store/room is required.
- Stock receipt, issue/consumption, adjustment, return, and transfer.
- Batch/lot and expiry date tracking where applicable.
- Stock movement ledger and current balance.
- Low-stock and expiry alerts.
- Valuation method must be chosen by the business (for example, weighted average or FIFO) before financial reporting is finalized.

**Inventory invariants**
- Never allow an unapproved client request to directly set a stock balance.
- Every stock change creates a movement record.
- Use a database transaction and row-level locking or an equivalent concurrency-safe strategy when consuming stock.
- Reject negative stock by default; any exception requires an explicit business rule and audit trail.
- For expiry-tracked material, prevent issuing expired lots unless a controlled exception is approved.
- Use a stable, documented costing method. Do not mix costing methods across reports.

### 5.5 Production

- Production order intake and customer/patient reference.
- Product/denture type, order specification, received date, promised due date.
- Configurable workflow stages; the requirement gives examples: gypsum model, design, wax-up, casting, finishing, and QC.
- Stage assignment, technician PIC, timestamps, status and notes.
- Material consumption associated with a production order.
- QC checklist, QC report and protected photo attachments.
- Completion and delivery tracking.

A stage transition should be validated by the server. The workflow must define which roles can move an order to each stage, whether a stage can be reopened, and how failed QC is handled.

### 5.6 Sales and finance

- Invoice generated from a production order, with explicit duplicate prevention.
- Invoice lines, amount, discount, due date and payment status.
- Payment records with payment method/reference.
- Expenses and production cost components (materials, labor, operations).
- Daily/monthly sales and gross-margin reports.
- Reversal/void flow with reason and permission controls.

**Financial invariants**
- Use `NUMERIC/DECIMAL` in PostgreSQL for currency; never use floating-point for money.
- Store currency code where future multi-currency support may be needed.
- Invoice number is unique and generated server-side.
- Payment posting is idempotent and transaction-safe.
- Do not equate invoice issuance with payment receipt.
- Define whether edits are allowed after an invoice is issued; prefer void-and-reissue or controlled credit/debit adjustments for posted documents.
- Tax, rounding, invoice legal format, and accounting treatment need owner/accountant approval.

### 5.7 Staff and tasks

- Staff profile, skills, availability and task assignment.
- Production KPI: completed jobs, on-time completion and QC outcomes.
- Attendance/overtime can be a later module; payroll is not included unless specified.

### 5.8 Dashboard and reporting

- Production count by week/month and stage.
- Delayed production orders.
- Low-stock and near-expiry materials.
- Sales, outstanding invoices, expenses, and gross margin.
- Top customers and products.
- KPI metrics must have written definitions and use consistent date/time boundaries.

Use paginated API queries and indexed aggregate queries. For heavy reports, use summary tables or background-generated reports only after profiling shows a need.

### 5.9 Notifications and integrations

- Internal notification/event record.
- Background jobs with retry count, next attempt, status and error category.
- Optional WhatsApp/email provider adapter.
- Do not put API keys in database logs or client-side code.
- Store delivery outcome and provider message ID.
- Apply retry backoff and idempotency to prevent duplicate notifications.
- Barcode/QR should contain opaque identifiers, not patient names or other sensitive details.

---

## 6. Suggested Database Design

Use UUID primary keys for public-facing entities where appropriate. Use `timestamptz` for timestamps and store instants in UTC; convert to the configured business timezone in the UI/report layer. Add `created_at`, `updated_at` and, where justified, `created_by`, `updated_by`. Avoid soft deletion for immutable ledger records.

### 6.1 Core tables

| Area | Tables (baseline) |
|---|---|
| Identity | `users`, `roles`, `permissions`, `user_roles`, `role_permissions`, `sessions` or `refresh_tokens`, `password_reset_tokens` |
| CRM | `customers`, `customer_contacts`, `customer_preferences`, `customer_notes`, `customer_feedback`, `discount_rules` |
| Catalog | `items`, `units_of_measure`, `suppliers`, `supplier_items`, `stock_locations` |
| Inventory | `inventory_batches`, `stock_movements`, `stock_movement_lines` or a single movement table, `stock_balances` (optional derived/cache table), `purchase_receipts`, `purchase_receipt_lines` |
| Production | `production_orders`, `production_order_items`, `production_stages`, `production_stage_runs`, `production_assignments`, `material_consumptions`, `quality_checks`, `quality_check_attachments`, `deliveries` |
| Finance | `invoices`, `invoice_lines`, `payments`, `expenses`, `expense_categories`, `production_cost_entries`, `invoice_sequences` |
| Staff | `staff_profiles`, `staff_skills`, `tasks`, `task_assignments`, `attendance_records` (later) |
| Platform | `audit_logs`, `attachments`, `notifications`, `background_jobs`, `idempotency_keys`, `system_settings` |

This is a logical baseline, not a final migration set. Confirm whether items, denture types, stock locations, and customer records need more specialized attributes before freezing migrations.

### 6.2 Key relationships

```mermaid
erDiagram
    USERS ||--o{ PRODUCTION_ASSIGNMENTS : assigned
    CUSTOMERS ||--o{ PRODUCTION_ORDERS : places
    PRODUCTION_ORDERS ||--o{ PRODUCTION_STAGE_RUNS : has
    PRODUCTION_ORDERS ||--o{ QUALITY_CHECKS : checked
    PRODUCTION_ORDERS ||--o{ INVOICES : billed
    INVOICES ||--o{ INVOICE_LINES : contains
    INVOICES ||--o{ PAYMENTS : receives
    ITEMS ||--o{ INVENTORY_BATCHES : batched
    INVENTORY_BATCHES ||--o{ STOCK_MOVEMENTS : moved
    PRODUCTION_ORDERS ||--o{ MATERIAL_CONSUMPTIONS : consumes
    USERS ||--o{ AUDIT_LOGS : performs
```

### 6.3 Schema rules

- Use foreign keys and explicit `ON DELETE` behavior. Prefer `RESTRICT` for financial, production, and stock history.
- Unique constraints for usernames/emails where applicable, SKU, invoice number, and idempotency key scope.
- Check constraints for nonnegative quantities, valid amounts and date relationships where practical.
- Index foreign keys used in joins, common filters, status, due dates, created timestamps, batch expiry and customer references.
- Use partial indexes for common active/open records when beneficial.
- Use `NUMERIC(precision, scale)` for money and quantity precision appropriate to the unit.
- Do not store multiple unrelated values in a comma-separated column.
- Use JSONB only for genuinely flexible metadata, not core relational fields.
- Use optimistic concurrency (`version` column) or guarded updates for records likely to be edited concurrently.
- Use database transactions for multi-table business operations.

### 6.4 Stock ledger strategy

`stock_movements` should record a movement type, item, location, batch (when tracked), quantity delta, unit cost where applicable, source document, actor, timestamp and reason. The exact schema can be normalized into header/line tables if a movement contains multiple items.

The current balance should be derived from the ledger or maintained as a transactionally updated balance table. If using a balance table, enforce that balance changes and ledger inserts occur in the same database transaction and periodically reconcile balance against the ledger.

### 6.5 Production workflow strategy

Use configurable `production_stages` and append stage history in `production_stage_runs`. Each run records stage, status, assigned technician, start/end timestamps, notes and transition actor. A current stage/status field on `production_orders` may be maintained for fast listing, but stage history remains the source of truth.

### 6.6 Audit history

`audit_logs` should capture actor, action, entity type/id, timestamp, request/correlation ID, outcome and a carefully redacted change summary. Never log passwords, access tokens, session cookies, full payment credentials or unnecessary patient information. Restrict audit-log access and prevent ordinary users from editing/deleting audit entries.

---

## 7. API Design

Base path: `/api/v1`

### 7.1 API conventions

- JSON request/response except file uploads and downloads.
- Standard HTTP status codes.
- Pagination: `page`, `page_size` with a strict maximum; consider cursor pagination for high-volume ledgers.
- Filters and sorting use allowlisted fields; never interpolate raw client input into SQL identifiers.
- Return a consistent error envelope with a stable machine-readable code, safe message and request ID.
- Use UTC ISO-8601 timestamps.
- Validate body, query, path parameters and domain invariants.
- Use idempotency keys for payment posting and other retry-sensitive write endpoints.
- OpenAPI contract is versioned and checked in CI.

Example error envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request contains invalid fields.",
    "request_id": "server-generated-correlation-id",
    "details": [
      { "field": "due_date", "code": "INVALID_DATE" }
    ]
  }
}
```

### 7.2 Endpoint outline

| Domain | Method and endpoint | Purpose |
|---|---|---|
| Auth | `POST /auth/login` | Authenticate and create session |
| Auth | `POST /auth/refresh` | Rotate refresh token/session |
| Auth | `POST /auth/logout` | Revoke current session |
| Auth | `GET /auth/me` | Current user and permissions |
| Users | `GET/POST /users` | List/create users (authorized roles only) |
| Users | `GET/PATCH /users/{id}` | View/update user |
| Roles | `GET/POST /roles` | Manage roles (owner/admin only) |
| Customers | `GET/POST /customers` | List/create doctor or clinic |
| Customers | `GET/PATCH /customers/{id}` | View/update customer |
| Items | `GET/POST /items` | Catalog |
| Suppliers | `GET/POST /suppliers` | Supplier management |
| Inventory | `GET /inventory/balances` | Current stock |
| Inventory | `GET/POST /inventory/movements` | List/create controlled movements |
| Inventory | `GET/POST /inventory/receipts` | Receive purchased material |
| Inventory | `GET /inventory/alerts` | Low-stock and expiry alerts |
| Production | `GET/POST /production-orders` | List/create orders |
| Production | `GET/PATCH /production-orders/{id}` | View/update permitted fields |
| Production | `POST /production-orders/{id}/transitions` | Validated stage/status transition |
| Production | `POST /production-orders/{id}/assignments` | Assign technician |
| Production | `POST /production-orders/{id}/material-consumptions` | Record material use |
| Production | `POST /production-orders/{id}/quality-checks` | Record QC outcome |
| Production | `POST /production-orders/{id}/attachments` | Upload QC files |
| Production | `POST /production-orders/{id}/deliveries` | Record delivery |
| Finance | `POST /production-orders/{id}/invoice` | Create invoice once eligible |
| Finance | `GET/POST /invoices` | List/create permitted invoices |
| Finance | `GET /invoices/{id}` | Invoice detail |
| Finance | `POST /invoices/{id}/payments` | Record payment |
| Finance | `POST /invoices/{id}/void` | Controlled void with reason |
| Finance | `GET/POST /expenses` | Record/list expenses |
| Reports | `GET /reports/dashboard` | Summary KPIs |
| Reports | `GET /reports/sales` | Sales report |
| Reports | `GET /reports/production` | Production report |
| Reports | `GET /reports/inventory` | Inventory report |
| Audit | `GET /audit-logs` | Restricted audit search |
| Notifications | `GET /notifications` | User notifications |

This endpoint list is a contract outline. Final request schemas, response schemas, permissions and state transitions must be documented in OpenAPI before implementation.

### 7.3 Example production transition request

```json
{
  "to_stage_code": "finishing",
  "note": "Casting completed and passed review",
  "expected_version": 4
}
```

The server must verify the user's permission, current state, allowed transition, required prior stage completion, order version and any QC/material conditions. Never trust a frontend-supplied status as authorization.

---

## 8. Authentication and Authorization

### 8.1 Roles

Suggested starting roles:
- **Owner:** all business functions, security configuration, reports and user administration.
- **Admin:** operational configuration and permitted management functions; sensitive owner-only actions can be separated.
- **Technician:** assigned production work, stage updates, material consumption within permissions, QC input as authorized.
- **Cashier/Finance:** invoices, payments, expenses and finance reports; no unrestricted user/security administration.
- **Viewer/Auditor:** read-only, explicitly scoped views.

Do not rely on role names alone. Define permissions such as `production.order.create`, `production.stage.transition`, `inventory.adjust`, `finance.payment.record`, `finance.invoice.void`, `users.manage`, `audit.read`.

### 8.2 Authorization requirements

- Enforce authorization in backend middleware and application services, not just by hiding UI buttons.
- Deny by default when no permission matches.
- Check record-level scope for assigned jobs and sensitive records.
- Require re-authentication or stronger verification for critical actions such as changing admin roles, voiding posted invoices, and large stock adjustments.
- Prevent users from granting themselves privileges.
- Log permission changes and privileged actions.

### 8.3 Session/token approach

Prefer secure, HttpOnly, SameSite cookies for browser sessions when frontend and API deployment allow it. Add CSRF protection for cookie-authenticated state-changing requests. If bearer tokens are used, keep access tokens short-lived and avoid persistent browser local storage for long-lived credentials. Refresh tokens must be rotated, hashed at rest, revocable, and reuse-detected.

- Passwords: hash with Argon2id or bcrypt; never encrypt passwords reversibly.
- Password reset: random, single-use, short-lived token stored hashed; return generic responses to avoid account enumeration.
- Login: rate-limit by account and source, use progressive delay, and log security events.
- Owner/admin: strongly recommend MFA.
- Logout: revoke server-side session/refresh token.
- Session expiry and idle timeout must be configurable.

---

## 9. Security Blueprint

Use OWASP ASVS as a verification reference and OWASP API Security Top 10 as a threat checklist.

### 9.1 Network and transport

- HTTPS only; redirect HTTP to HTTPS.
- TLS certificates auto-renewed and monitored.
- PostgreSQL bound to localhost/private network; firewall deny-by-default.
- SSH key-based access; disable password SSH where operationally feasible; restrict admin access.
- Separate app and database credentials; no superuser credentials in the application.
- Security headers: HSTS after HTTPS is verified, `X-Content-Type-Options`, appropriate `Content-Security-Policy`, and frame protection.
- Configure reverse-proxy body-size and timeout limits.

### 9.2 Application security

- Validate and normalize all input on the server.
- Use parameterized queries/GORM placeholders; never concatenate untrusted values into SQL.
- Protect against broken object-level authorization (BOLA/IDOR) by checking permissions and ownership on every object request.
- Use allowlists for sorting/filtering and strict pagination limits.
- Apply rate limits to login, password reset, upload, export and expensive reports.
- CORS: allow only known frontend origins; never combine wildcard origins with credentials.
- Protect cookie sessions against CSRF.
- Use safe output encoding in the frontend; avoid unsafe HTML rendering.
- Avoid leaking stack traces, SQL errors, secrets or internal paths to clients.
- Configure request timeouts, concurrency limits and graceful shutdown.
- Use idempotency and database uniqueness to resist duplicate requests.

### 9.3 File upload security (QC photos/reports)

- Accept only required formats, such as JPEG/PNG and PDF, after business approval.
- Check extension, MIME type and file signature; do not trust the browser-provided content type.
- Set strict file-size and image-dimension limits.
- Generate random server-side filenames; never use user-supplied paths.
- Store outside the web root or in private object storage.
- Serve through an authorized download endpoint or short-lived signed URL.
- Consider malware scanning before files become available to other users.
- Strip EXIF metadata from images when appropriate.
- Prevent path traversal, archive bombs and executable content.
- Log upload/download metadata without logging file contents.
- Enforce per-order and per-user authorization on downloads.

### 9.4 Data protection

- Minimize patient data collected; only store what production operations require.
- Encrypt backups and storage volumes; use managed encryption where available.
- Use TLS for network traffic.
- Keep secrets in environment files readable only by the service account or use a secret manager.
- Redact personal data in logs.
- Define data retention, deletion and access policies with the business.
- If Indonesia's Personal Data Protection Law (UU PDP) applies, have the owner/legal advisor validate notices, lawful basis, data-subject handling, retention and breach response. This plan is technical guidance, not legal advice.

### 9.5 Audit and monitoring

Audit at minimum:
- Login success/failure, logout, password reset and session revocation.
- User/role/permission changes.
- Stock receipt, issue, adjustment, return and transfer.
- Production stage transitions, assignment, QC decision and delivery.
- Invoice creation/void, payment recording/reversal, expense changes.
- Sensitive exports, attachment access where appropriate, and configuration changes.

Audit entries should be append-only for application users. Restrict database access so normal application credentials cannot arbitrarily rewrite historical audit data where practical. Send security alerts for repeated login failures, privilege changes, unexpected admin access, backup failure and suspicious export activity.

### 9.6 Threat model

| Threat | Mitigation |
|---|---|
| Stolen credentials | Strong password hashing, MFA for privileged users, rate limits, session revocation |
| IDOR/BOLA | Object-level authorization on every read/write/download |
| SQL injection | Parameterized queries, strict allowlists, least-privilege DB role |
| Cross-site request forgery | SameSite cookies plus CSRF token/origin checks |
| Malicious upload | Type/signature validation, size limits, private storage, scanning |
| Duplicate payment or invoice | DB constraints, transactions, idempotency keys |
| Concurrent stock issue | Row locks/guarded updates and transactional ledger writes |
| Insider changes to finance/stock | Least privilege, audit logs, approval/reversal workflow |
| VPS compromise | Firewall, patching, SSH hardening, isolated DB, off-site encrypted backups |
| Data loss/ransomware | Off-site immutable or versioned backups and tested restoration |
| Secret leakage | Secret management, log redaction, repository scanning and rotation |
| Resource exhaustion | Request limits, rate limits, pagination caps, timeouts and monitoring |

---

## 10. Go Project Structure

Suggested structure:

```text
dental-lab/
├── cmd/
│   ├── api/main.go
│   └── worker/main.go
├── internal/
│   ├── config/
│   ├── platform/
│   │   ├── database/
│   │   ├── logger/
│   │   ├── httpserver/
│   │   ├── auth/
│   │   ├── authorization/
│   │   ├── audit/
│   │   ├── storage/
│   │   ├── jobs/
│   │   └── observability/
│   ├── modules/
│   │   ├── users/
│   │   ├── customers/
│   │   ├── catalog/
│   │   ├── suppliers/
│   │   ├── inventory/
│   │   ├── production/
│   │   ├── finance/
│   │   ├── staff/
│   │   ├── reports/
│   │   └── notifications/
│   └── shared/
│       ├── errors/
│       ├── pagination/
│       ├── validation/
│       └── clock/
├── migrations/
├── api/openapi.yaml
├── docs/
│   ├── architecture.md
│   ├── security.md
│   ├── runbook.md
│   └── decisions/
├── scripts/
├── test/
│   ├── integration/
│   └── e2e/
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── .env.example
```

### 10.1 Module internal layout

Example `internal/modules/inventory/`:

```text
inventory/
├── handler.go       # Fiber handlers; HTTP only
├── dto.go           # Request/response structures
├── service.go       # Business rules and transactions
├── repository.go    # GORM/database access
├── model.go         # Persistence models
├── policy.go        # Domain authorization checks
├── errors.go
└── service_test.go
```

Keep handlers thin. Business rules belong in services/domain logic, not in Fiber handlers or GORM hooks. Repositories should accept a transaction handle where operations must share a transaction. Avoid circular dependencies between modules; use narrow interfaces for cross-module calls.

### 10.2 Configuration

Example `.env.example` (placeholders only):

```dotenv
APP_ENV=development
APP_ADDR=127.0.0.1:8080
APP_PUBLIC_URL=http://localhost:3000
DATABASE_URL=postgres://dental_app:CHANGE_ME@127.0.0.1:5432/dental_lab?sslmode=disable
AUTH_ACCESS_TTL=15m
AUTH_REFRESH_TTL=168h
UPLOAD_MAX_BYTES=10485760
UPLOAD_DIR=./private_uploads
LOG_LEVEL=info
```

For production, do not copy development `sslmode=disable` blindly. Require TLS for remote database connections, restrict configuration file permissions, validate required settings at startup, and never commit actual `.env` files.

---

## 11. GORM and Transaction Guidelines

- Use GORM for CRUD and relationship mapping, but use explicit SQL for migrations, complex reporting, and database features that need precise control.
- Production schema changes must use versioned migrations. `AutoMigrate` may be useful in local prototypes but must not silently define production schema evolution.
- Avoid implicit N+1 queries; use explicit preloading or query joins after measuring query behavior.
- Select only required columns for list endpoints.
- Use context cancellation and request deadlines for database operations.
- Use transactions for multi-step workflows:
  - Receive inventory: receipt + batch + movement + balance update.
  - Consume materials: validate stock + movement + cost entry + balance update.
  - Create invoice: invoice + lines + sequence allocation + audit event.
  - Record payment: payment + invoice balance/status update + audit event.
  - Transition production stage: state update + stage history + audit event.
- Handle unique-constraint violations as domain conflicts, not generic server errors.
- Do not depend on hooks for critical financial/inventory behavior unless thoroughly tested and clearly documented.
- Use `FOR UPDATE`/guarded updates for contention-sensitive stock balances and invoice state changes.
- Keep transactions short; never perform network calls while holding database locks.
- Set pool limits and connection lifetime intentionally; monitor pool saturation.

### 11.1 Example transaction pattern

```go
func (s *Service) RecordStockIssue(ctx context.Context, input IssueInput) error {
    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 1. Load and lock the relevant balance/batch rows.
        // 2. Validate permission, quantity, expiry and available stock.
        // 3. Insert immutable stock movement.
        // 4. Update a derived balance row if the design maintains one.
        // 5. Insert audit/outbox event in the same transaction.
        // Return an error to roll back every step on failure.
        return nil
    })
}
```

This is a structural example only; implement the locking and SQL according to the final schema.

---

## 12. Migration and Data Integrity Plan

1. Create a migration baseline with extensions only when explicitly required.
2. Add tables and constraints in dependency order.
3. Seed permissions, default roles, production stages and required system settings using repeatable/controlled seed logic.
4. Keep schema migrations separate from application startup.
5. For new required columns: add nullable/defaulted column, deploy compatible code, backfill, verify, then add `NOT NULL` in a later migration.
6. For renames or type changes: use expand-and-contract, not a destructive one-step change.
7. Take a verified backup before high-risk production migrations.
8. Run migrations once as a controlled deployment step, not concurrently from every API replica.
9. Use migration checksums/version tracking and fail deployment if migration state is inconsistent.
10. Test both a fresh database install and upgrade from the previous release.
11. Never use destructive migration behavior without an approved recovery plan.
12. Reconcile inventory ledger/balances and invoice totals after migration.

---

## 13. Background Jobs and Reliability

Use a PostgreSQL-backed queue/outbox or another explicitly selected job system for the first release. Jobs should have:
- Unique job/event ID.
- Job type and payload version.
- Status (`pending`, `processing`, `succeeded`, `failed`).
- Attempt count, next retry time, last error category and timestamps.
- Idempotent handler behavior.
- Exponential backoff with jitter and a maximum retry policy.
- Dead-letter/manual retry handling for persistent failures.

Use the transactional outbox pattern when a database change must reliably trigger a notification: save the business change and outbox event in one transaction, then let the worker deliver it. External network calls must not be made inside database transactions.

---

## 14. Testing and Quality Gates

### 14.1 Test layers

- **Unit tests:** pricing, discounts, inventory rules, stage transitions, invoice/payment state rules, permission policies.
- **Repository/integration tests:** real PostgreSQL, foreign keys, constraints, transactions, locks, migrations.
- **API tests:** authentication, validation, authorization, pagination, error envelopes.
- **End-to-end tests:** order intake to production/QC/delivery/invoice/payment.
- **Security tests:** IDOR/BOLA, role bypass, SQL injection probes, CSRF/CORS, rate limits, upload validation, secret leakage.
- **Concurrency tests:** two users consuming the same remaining stock; duplicate invoice/payment requests.
- **Recovery tests:** restore database and file backups, then verify cross-references and application startup.
- **Performance tests:** dashboard, stock ledger, production list and reports with representative data.

### 14.2 CI pipeline

1. Format (`gofmt`) and static checks (`go vet`).
2. Unit tests and race detector where practical.
3. Dependency and secret scanning.
4. Build API and worker.
5. Validate OpenAPI and migration ordering.
6. Run integration tests against PostgreSQL.
7. Build frontend and run lint/type checks.
8. Produce versioned artifacts and deployment notes.
9. Require review for migrations, authorization and financial/inventory logic.

### 14.3 Release acceptance gates

- No critical/high unresolved security findings.
- All role/permission tests pass.
- Inventory ledger and balance reconciliation passes.
- Invoice/payment totals reconcile.
- Backup restore test passes for the release.
- Rollback or forward-fix plan exists for migrations.
- Monitoring and alerting are active.

---

## 15. Backup, Disaster Recovery and Operations

### 15.1 Backup policy

A starting policy to approve with the owner:
- Daily encrypted PostgreSQL logical backup.
- More frequent backups or WAL archiving if the business requires a lower recovery point objective.
- Separate backup of uploaded files and configuration metadata.
- Off-site copy in a separate account/location; use immutable/versioned retention where available.
- Retention example: daily copies for 14 days and monthly copies for 3–6 months, adjusted to legal, business and storage requirements.
- Encrypt backup files and protect encryption keys separately.
- Never count a backup as successful until the job verifies its output and reports status.

### 15.2 Define recovery objectives

Business owner must approve:
- **RPO (Recovery Point Objective):** maximum tolerable data loss.
- **RTO (Recovery Time Objective):** maximum tolerable service downtime.

A daily backup alone may be insufficient if losing a full day of stock movements, production updates or payments is unacceptable.

### 15.3 Restore drill

At least periodically and after major infrastructure changes:
1. Restore PostgreSQL into an isolated environment.
2. Restore corresponding file backup.
3. Verify migration version and application compatibility.
4. Run integrity checks for orders, invoices, payments, stock movements and attachment references.
5. Start API/worker against restored data.
6. Record actual RPO/RTO and any failures.
7. Fix the process and repeat until successful.

### 15.4 Operations checklist

- Patch OS, Go runtime/build image, PostgreSQL and dependencies.
- Monitor disk, memory, CPU, database connections, query latency and backup status.
- Rotate secrets and certificates.
- Restrict production shell and database access.
- Maintain runbooks for deployment, rollback, restore, compromised credentials and data incident response.
- Keep staging data synthetic or masked.
- Alert on disk exhaustion, database unavailable, repeated 5xx, job backlog, failed backups and certificate expiry.

---

## 16. Sprint-by-Sprint Implementation Plan

Assumption: one small team; sprint length 1–2 weeks. Estimates must be recalibrated after discovery.

### Sprint 0 — Discovery and architecture decisions

**Deliverables**
- Confirm workflows, users/roles, production stages, inventory costing, invoice rules, tax/rounding, payment methods, stock locations, file retention, and business timezone.
- Confirm single-lab deployment versus multi-branch/multi-tenant.
- Define RPO/RTO, data migration needs, and reporting definitions.
- Approve wireframes and OpenAPI conventions.

**Acceptance**
- Business rules documented and signed off.
- All critical unknowns have owners and due dates.
- Initial threat model and data classification approved.

### Sprint 1 — Foundation and secure deployment

**Deliverables**
- Go project layout, configuration validation, structured logs, request IDs.
- PostgreSQL connection pool and migration runner.
- Fiber middleware: recovery, request limits, timeouts, CORS allowlist, security headers.
- CI pipeline, health/readiness checks, local development environment.
- Deployment skeleton and secret-handling policy.

**Acceptance**
- Fresh database can be migrated deterministically.
- Secrets are absent from repository.
- API fails safely when required configuration is missing.
- PostgreSQL is not publicly exposed.

### Sprint 2 — Authentication, users, roles and audit

**Deliverables**
- Login/logout/session lifecycle, password hashing/reset.
- User CRUD, role/permission tables and permission middleware.
- Audit service and security-event logging.
- Owner/admin protections and optional MFA design.

**Acceptance**
- Unauthenticated requests cannot access protected APIs.
- Each role has automated allow/deny tests.
- Session revocation works.
- Sensitive fields are not written to logs.

### Sprint 3 — Customer, catalog and supplier master data

**Deliverables**
- Doctor/clinic customers and contact data.
- Material/product catalog, units and suppliers.
- Search, filtering, pagination and validation.
- Initial import template if existing data needs migration.

**Acceptance**
- Duplicate business identifiers are rejected appropriately.
- Invalid references are blocked by database constraints.
- Users cannot access unauthorized customer records.

### Sprint 4 — Inventory ledger and stock alerts

**Deliverables**
- Stock receipts, issues, adjustments and returns.
- Batch/expiry tracking and minimum stock thresholds.
- Stock balance queries and low-stock/expiry alerts.
- Stock movement audit and reconciliation report.

**Acceptance**
- Every balance change has a movement record.
- Concurrent issue tests cannot overspend stock.
- Expired batch policy is enforced.
- Ledger and balance reconciliation passes.

### Sprint 5 — Production order and workflow

**Deliverables**
- Production order intake, due dates, product/specification and customer links.
- Configurable stages and transition history.
- Technician assignment and status tracking.
- Delay list and production board.

**Acceptance**
- Illegal stage transitions are rejected server-side.
- Every transition records actor and timestamp.
- Due-date and delayed-order reports match test fixtures.

### Sprint 6 — Material consumption, QC and delivery

**Deliverables**
- Material consumption tied to order and stock batch.
- QC checklist/report and outcome.
- Protected file upload/download.
- Delivery tracking and completion rules.

**Acceptance**
- Consumption and stock movement commit atomically.
- Failed QC can follow an approved rework path.
- Unauthorized users cannot upload or download attachments.
- Oversized or invalid files are rejected.

### Sprint 7 — Invoices, payments and expenses

**Deliverables**
- Invoice generation from eligible production orders.
- Invoice line items, due date and status.
- Payment posting, reversal/void workflow and idempotency.
- Expense and production-cost records.

**Acceptance**
- Duplicate invoice generation is prevented.
- Duplicate payment request does not double-post.
- Money uses decimal arithmetic.
- Invoice totals, payments and outstanding balances reconcile.

### Sprint 8 — Dashboard and reporting

**Deliverables**
- Production weekly/monthly metrics.
- Inventory valuation/usage reports based on approved costing policy.
- Sales, outstanding invoices, expenses and gross margin.
- Top customers/products and overdue alerts.

**Acceptance**
- KPI definitions and timezone boundaries are documented.
- Reports reconcile with source records.
- Large queries are paginated or bounded and use measured indexes.

### Sprint 9 — Staff/task management and CRM enhancements

**Deliverables**
- Staff profiles and skills.
- Task assignment and technician KPI.
- Customer preferences, notes, feedback and order history.
- Discount/loyalty rules if approved.

**Acceptance**
- KPI calculations are reproducible from defined source data.
- Discount rules cannot bypass invoice controls.
- Sensitive notes are permission-protected.

### Sprint 10 — Notifications, UAT and production launch

**Deliverables**
- Background job/outbox worker.
- Optional email/WhatsApp adapter after provider selection.
- User acceptance testing, security review, performance baseline.
- Backup restore drill, runbooks and production deployment.

**Acceptance**
- Failed jobs retry safely without duplicate side effects.
- Backup restoration is demonstrated.
- Critical/high security issues are resolved or explicitly accepted by accountable owner.
- UAT sign-off and rollback plan exist.

**Indicative scope:** 11 sprints including discovery. Calendar duration depends on team size, UI complexity, legacy data migration, and accounting decisions.

---

## 17. User Stories and Acceptance Criteria

### Inventory

**As an inventory operator,** I want to record incoming material by batch and expiry date so that stock can be traced.

- Given a valid receipt, when it is posted, then a stock movement and batch record are created in one transaction.
- Given a duplicate receipt reference, when submitted again, then the API rejects it or returns the idempotent result.
- Given an expired batch, when consumption is attempted, then the operation is blocked unless an approved exception policy exists.

### Production

**As a production manager,** I want to see every order's current stage, PIC and due date so that delays can be addressed.

- Every transition is validated against the configured workflow.
- Every transition records the actor, time, previous state and next state.
- A user without permission cannot assign a technician or change a stage.
- A completed order cannot be delivered twice without an explicit correction flow.

### Finance

**As a cashier,** I want to record payments against invoices so that outstanding balances remain accurate.

- Payment posting is atomic and idempotent.
- A payment cannot exceed the outstanding amount unless an approved overpayment rule exists.
- Posted financial history is corrected by reversal/adjustment, not silent deletion.
- Only authorized roles can void invoices or reverse payments.

### Security

**As an owner,** I want sensitive operations restricted and audited.

- Role changes require the proper permission and are audited.
- Direct object IDs do not bypass record-level authorization.
- Passwords, tokens and secrets never appear in logs.
- Backup restoration is tested and documented.

---

## 18. Environment and Deployment Plan

### 18.1 Environments

- **Local:** developer machines, local PostgreSQL, synthetic data.
- **Staging:** production-like configuration, separate credentials and database, masked/synthetic data.
- **Production:** restricted access, managed secrets, monitoring and off-site backups.

### 18.2 Deployment sequence

1. Build immutable, versioned API/worker/frontend artifacts in CI.
2. Back up production database and files before risky changes.
3. Verify migration compatibility and run migrations once.
4. Deploy API/worker version compatible with both old and new schema where expand-and-contract is used.
5. Run smoke tests and verify health, login, inventory, production and finance paths.
6. Monitor error rate, latency, job backlog and database health.
7. Roll back application artifacts if needed; use a forward migration/fix when schema rollback would risk data loss.
8. Record release version and migration version.

### 18.3 Production hardening checklist

- [ ] HTTPS and certificate renewal verified.
- [ ] PostgreSQL private-only access and least-privilege database user.
- [ ] Production secrets injected securely and file permissions restricted.
- [ ] Debug mode and verbose SQL logging disabled.
- [ ] Upload directory private and outside web root.
- [ ] CORS allowlist and security headers verified.
- [ ] Login and sensitive endpoints rate-limited.
- [ ] Backups encrypted, off-site and monitored.
- [ ] Restore drill completed.
- [ ] Admin accounts protected with MFA where available.
- [ ] Logs redact tokens and sensitive data.
- [ ] Dependency and OS patch process assigned.
- [ ] Incident response and rollback runbooks available.

---

## 19. Performance and Scalability

Start with a single API service and PostgreSQL. Optimize from measurements rather than introducing distributed infrastructure prematurely.

- Index production order status/due date, customer/date, invoice status/due date, stock item/location/batch, and movement timestamp.
- Use pagination and bounded date ranges for lists/reports.
- Avoid loading all movement history for a simple current-stock screen.
- Use connection pooling and set sane limits relative to PostgreSQL capacity.
- Cache only read-heavy data where invalidation is understood; never use cache as the financial or stock source of truth.
- Move slow report generation and notifications to background jobs.
- Store large files in object storage instead of PostgreSQL bytea unless there is a specific, justified requirement.
- Add read replicas or separate reporting infrastructure only when actual load warrants them.
- Define and test a realistic performance target after expected users, orders/day, file volume and retention are known.

---

## 20. Confirmed Business Decisions & Specifications

The 15 baseline discovery questions have been confirmed by the business owner. These decisions govern database schema, business logic constraints, and API design:

| # | Question | Confirmed Decision | System & Technical Implications |
|---|---|---|---|
| 1 | Single lab vs multi-branch vs SaaS | **Single Laboratory (1 Lab)** | Standalone single-tenant deployment. No complex multi-tenant partitioning or branch isolation layers needed, keeping schema lean and performant. |
| 2 | Roles & Approval operations | **Owner, Lab Assistant/Admin, Technician, Finance/Cashier**.<br>**Owner Approval required** for purchasing production materials. | Implement Purchase Requisition/PO workflow: Material purchase requests require explicit Owner approval (`status = APPROVED_BY_OWNER`) before stock receipt is permitted. |
| 3 | Warehouse locations | **Single Location (1 Gudang/Warehouse)** | Single warehouse inventory model (`Main Lab Store`). No inter-branch or cross-warehouse transfers needed in v1. |
| 4 | Batch & Expiry tracking | **Optional batch/expiry tracking** (`not required`) | Add `lot_number` and `expiry_date` fields to item receipt/input forms as nullable/optional fields. If provided, generate near-expiry alerts in inventory dashboards. |
| 5 | Costing / Valuation method | **Pencatatan manual (Manual Cost Recording)** | Store manual purchase unit price and manual cost allocation per production order. No automatic weighted average or complex FIFO cost tracking engine needed. |
| 6 | Production stages & QC reopen | **Varied stages by product type**;<br>**Orders CAN be reopened after QC failure** | Stage templates are configurable per denture type (e.g., Crown/Bridge, Acrylic, Valplast, Orthodontic). QC failure transitions order to `REOPENED_FOR_REWORK` back to designated technical stage with mandatory defect notes. |
| 7 | Invoicing trigger point | **Production Completion** | Invoices are issued automatically or on-demand when production stages and QC pass successfully (`status = COMPLETED`), prior to or during delivery. |
| 8 | Payment types supported | **Yes**: Partial payments, deposits (DP), refunds, credit terms (tempo), and overpayments | Payment ledger supports down payments at order intake, multiple partial payment records against an invoice, track outstanding balances (`balance_due`), store credit for overpayments, and audit-logged refunds/voids. |
| 9 | Tax & Legal invoice rules | **No special tax rules currently** | Standard subtotal, discount, and total without complex VAT/e-Faktur integration for v1. |
| 10 | Ledger vs Operational Finance | **Laporan keuangan operasional only** | Implement operational cash flow, order revenue, material & operational expenses, gross margin, and accounts receivable (aging piutang dokter). Full double-entry general ledger is out of scope. |
| 11 | Patient Info & Retention | **Reference data**: Patient Name, Doctor/Clinic Name, Target Teeth (Odontogram/tooth numbers), Denture Type, Shade/Color, Photos/Impression Scans.<br>**Permanent retention** (do not delete). | Custom dental work order schema with odontogram/teeth selection, shade guide reference, and secure attachment storage. Customer and order records must never be hard-deleted (`deleted_at` soft-delete or permanent retention for warranty/repeat orders). |
| 12 | Notification provider | **Manual WhatsApp order / Lab Assistant communication** | Add automated "Click to Chat" WhatsApp action buttons (`https://wa.me/{phone}?text={template}`) with dynamic message templates (Order Received, Ready for Delivery, Invoice summary). No paid 3rd-party API needed for v1. |
| 13 | RPO / RTO & Backup | **RPO: 24 Jam** (daily EOD backup)<br>**RTO: 24-48 Jam** (ready in max 2 days on machine failure/virus) | Implement automated End-of-Day (EOD) database dump (`pg_dump`) and encrypted asset sync to external/secondary cloud drive. Fast containerized restore runbook (`docker-compose up` / restore script). |
| 14 | Scale & Volume | **Around 30–50 clinic/doctor customers** | Estimated ~5–20 orders/day. Single lightweight VPS or local server with PostgreSQL + Go Fiber handles this load comfortably with sub-second response times and minimal hardware cost. |
| 15 | Legacy data migration | **No legacy data import needed** (previously manual) | Fresh start without legacy database ETL. System launches directly with clean master data setup (materials catalog, denture types, doctor/clinic contacts). |

---

## 21. Definition of Done

A feature is complete only when:
- Business rules and edge cases are documented.
- Authorization is enforced server-side and tested.
- Input validation and safe error handling are implemented.
- Database constraints and migration are included.
- Audit logging exists for sensitive mutations.
- Unit and integration tests pass.
- OpenAPI is updated.
- Logs contain correlation IDs without secrets.
- Performance is acceptable for representative data.
- User-facing acceptance criteria pass.
- Deployment and rollback notes are available.

A release is production-ready only after UAT, security review, backup restoration test, monitoring verification, and accountable business sign-off.

---

## 22. Final Recommendation

Implement the platform as a **Go Fiber + GORM + PostgreSQL modular monolith**, with a responsive web frontend and a separate background worker. Prioritize the High modules—inventory, production, finance, dashboard, and secure online access—while laying the security and audit foundations in Sprint 1–2 rather than postponing them.

The most important design safeguards are: immutable stock and financial history, transaction-safe stock consumption and payment posting, server-side permission checks, private file storage, versioned SQL migrations, and verified off-site backups. Finalize the open business decisions before freezing the database schema or implementing financial calculations.
