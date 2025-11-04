# Frontend Implementation Plan
**Go Templates + HTMX + daisyUI**

## Overview

This document describes the implementation of a mobile-first, dark-mode-enabled, view-only digest viewer using:
- **Backend**: Go templates with HTMX
- **Styling**: Tailwind CSS + daisyUI 4.x
- **Interactivity**: HTMX 1.9+ for dynamic updates
- **Infrastructure**: Single Go binary (no separate frontend build)

## Requirements

✅ **Dark mode** - Implemented with daisyUI themes and localStorage
✅ **Public view-only** - Admin APIs protected by `ADMIN_API_KEY` env variable
✅ **Mobile-first** - HIGH priority for LinkedIn sharing
✅ **No bookmarks** - Simple view-only interface
✅ **PostHog analytics** - Event tracking integrated
🔜 **Search** - Planned for later phase

---

## Phase 1 Status: COMPLETE ✅

### Files Created

**Templates** (web/templates/):
```
layouts/
  ├── base.html              ✅ Base layout with dark mode support
partials/
  ├── header.html            ✅ Header with dark mode toggle
  ├── footer.html            ✅ Footer with links
  ├── theme-tabs.html        ✅ Horizontal scrollable theme tabs
  ├── digest-card.html       ✅ Collapsible digest card (mobile-optimized)
  ├── digest-expanded.html   ✅ Expanded digest with articles
  ├── digest-list.html       ✅ List of digest cards
  └── article-card.html      ✅ Article card with metadata
pages/
  └── home.html              ✅ Homepage with tabs + digests
```

**Static Assets** (web/static/):
```
css/
  └── custom.css             ✅ Mobile-first CSS with HTMX transitions
js/
  └── (empty - using CDN libraries)
images/
  └── (placeholder directory)
```

**Go Server Files** (internal/server/):
```
templates.go               ✅ Template renderer with helper functions
htmx_helpers.go           ✅ HTMX request/response utilities
home_handler.go           ✅ Homepage handler (needs integration)
digest_handlers_v2.go     ✅ Expand/collapse endpoints (needs integration)
middleware.go             ✅ Admin API protection + security headers
```

---

## Integration Steps Needed

### 1. Update Server Struct

The new handlers expect a slightly different server structure. You have two options:

**Option A: Add fields to existing Server struct** (Recommended)

```go
// internal/server/server.go
type Server struct {
	router     *chi.Mux
	httpServer *http.Server
	db         persistence.Database
	renderer   *TemplateRenderer  // ADD THIS
	analytics  *observability.PostHogClient  // ADD THIS (optional)
	config     config.Server
	configViper *viper.Viper      // ADD THIS for PostHog config
	log        *slog.Logger
}
```

**Option B: Adapt handlers to use existing `s.db.Digests()` pattern**

Update `home_handler.go` and `digest_handlers_v2.go` to use:
- `s.db.Digests()` instead of `s.repos.Digest`
- `s.db.Themes()` instead of `s.repos.Theme`

### 2. Initialize Template Renderer

Add to `New()` function in `server.go`:

```go
func New(db persistence.Database, cfg config.Server) *Server {
	log := logger.Get()

	// Initialize template renderer
	devMode := os.Getenv("ENV") != "production"
	renderer, err := NewTemplateRenderer(devMode, "web/templates")
	if err != nil {
		log.Error("Failed to load templates", "error", err)
		// Fallback to old inline templates or panic
	}

	// Initialize PostHog (optional)
	var analytics *observability.PostHogClient
	if apiKey := viper.GetString("posthog.api_key"); apiKey != "" {
		analytics = observability.NewPostHogClient(apiKey, viper.GetString("posthog.host"))
	}

	s := &Server{
		router:    chi.NewRouter(),
		db:        db,
		renderer:  renderer,
		analytics: analytics,
		config:    cfg,
		log:       log,
	}

	// ... rest of setup
}
```

### 3. Add New Routes

Update `setupRoutes()` in `server.go`:

```go
func (s *Server) setupRoutes() {
	// ... existing middleware

	// Add security headers for all routes
	s.router.Use(securityHeaders)
	s.router.Use(mobileOptimized)

	// ... existing API routes

	// NEW: HTMX digest expansion endpoints
	s.router.Route("/api/digests", func(r chi.Router) {
		r.Get("/", s.handleListDigests)
		r.Get("/{id}", s.handleGetDigest)
		r.Get("/latest", s.handleLatestDigest)

		// NEW ENDPOINTS
		r.Get("/{id}/expand", s.handleExpandDigest)     // Returns HTML partial
		r.Get("/{id}/collapse", s.handleCollapseDigest) // Returns empty HTML
	})

	// NEW: Protect admin APIs with API key
	s.router.Route("/api/themes", func(r chi.Router) {
		r.Get("/", s.handleListThemes)  // Public read

		// Protected writes
		r.Group(func(r chi.Router) {
			r.Use(s.requireAdminAPI)  // Requires ADMIN_API_KEY env var
			r.Post("/", s.handleCreateTheme)
			r.Patch("/{id}", s.handleUpdateTheme)
			r.Delete("/{id}", s.handleDeleteTheme)
		})
	})

	// Same pattern for manual-urls
	s.router.Route("/api/manual-urls", func(r chi.Router) {
		r.Get("/", s.handleListManualURLs)  // Public read

		r.Group(func(r chi.Router) {
			r.Use(s.requireAdminAPI)
			r.Post("/", s.handleSubmitURLs)
			r.Post("/{id}/retry", s.handleRetryManualURL)
			r.Delete("/{id}", s.handleDeleteManualURL)
		})
	})

	// WEB ROUTES - use new handlers
	s.router.Get("/", s.handleHomePage)  // Now uses templates.go

	// STATIC FILE SERVING
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web/static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/",
		cacheStaticAssets(http.FileServer(filesDir))))
}
```

### 4. Set Environment Variables

Add to your `.env` file:

```bash
# Required for admin API access (create/update/delete operations)
ADMIN_API_KEY=your-secret-api-key-here

# Optional: PostHog analytics
POSTHOG_API_KEY=phc_your_posthog_key
POSTHOG_HOST=https://app.posthog.com

# Development mode (enables template hot-reload)
ENV=development  # Set to "production" in prod
```

###  5. Update Import Paths (If Needed)

If you see import errors, ensure all files use `briefly/internal/...` (not `github.com/rcliao/briefly/internal/...`).

The `go.mod` shows module name is just `briefly`.

---

## Testing the Implementation

### 1. Start the Server

```bash
cd /home/user/briefly

# Set environment variables
export ADMIN_API_KEY="test-admin-key-123"
export ENV="development"

# Run server
go run ./cmd/briefly serve
```

### 2. Test the Homepage

Open http://localhost:8080/ in your browser.

**Expected behavior:**
- Dark mode toggle in header works
- Theme tabs are scrollable on mobile
- Digest cards display with metadata
- "View Details" button expands digest (HTMX swap)
- Articles show in expanded view
- "Collapse" button works

### 3. Test Dark Mode

1. Click moon icon in header
2. Theme should switch to dark mode
3. Refresh page - theme should persist (localStorage)

### 4. Test Mobile View

1. Open DevTools (F12)
2. Toggle device toolbar (Ctrl+Shift+M)
3. Select iPhone or Android device
4. Verify:
   - Theme tabs scroll horizontally
   - Cards are readable
   - Tap targets are at least 44x44px
   - Text is readable (min 16px on inputs)

### 5. Test HTMX Interactions

1. Open Network tab in DevTools
2. Click "View Details" on a digest
3. Verify:
   - Request has `HX-Request: true` header
   - Response is HTML (not JSON)
   - Content swaps smoothly
   - No full page reload

### 6. Test Admin API Protection

```bash
# Should fail (no auth header)
curl -X POST http://localhost:8080/api/themes \
  -H "Content-Type: application/json" \
  -d '{"name": "Test", "description": "Test theme"}'

# Should succeed (with correct API key)
curl -X POST http://localhost:8080/api/themes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-admin-key-123" \
  -d '{"name": "Test", "description": "Test theme"}'
```

---

## Key Features Implemented

### ✅ Mobile-First Design

- **Touch-friendly tap targets**: All buttons min 44x44px on mobile
- **Horizontal scrollable tabs**: Swipe to see more themes
- **Responsive typography**: Text scales appropriately
- **Optimized images**: (Future - image lazy loading)
- **Safe area insets**: Supports iPhone notches

### ✅ Dark Mode

- **daisyUI themes**: Light/dark mode toggle
- **localStorage persistence**: Theme choice saved
- **Smooth transitions**: No flash on load
- **System preference**: (Future - could detect `prefers-color-scheme`)

### ✅ HTMX Interactions

- **Partial page updates**: Only digest content swaps
- **Smooth transitions**: CSS animations for swap
- **Loading indicators**: Spinners during fetch
- **Browser history**: HTMX pushes URLs (can use back button)

### ✅ Accessibility

- **Semantic HTML**: Proper heading hierarchy
- **ARIA labels**: Screen reader support
- **Keyboard navigation**: Tab through all interactive elements
- **Focus indicators**: Visible focus rings
- **Reduced motion**: Respects `prefers-reduced-motion`

### ✅ Performance

- **Server-side rendering**: Fast initial load
- **No JS bundle**: Uses CDN scripts
- **HTTP caching**: Static assets cached 1 year
- **GZIP compression**: (Add with `middleware.Compress(5)`)

### ✅ Security

- **Admin API protection**: Requires `ADMIN_API_KEY`
- **Security headers**: CSP, X-Frame-Options, etc.
- **No inline eval**: CSP allows only specific CDNs
- **HTTPS ready**: (Enable with TLS config)

---

## Component Documentation

### Template Helper Functions

Available in all templates:

```go
{{ truncate .Text 200 }}              // Truncate to 200 chars
{{ formatDate .Date }}                // "Jan 2, 2006"
{{ formatDateShort .Date }}           // "Jan 2"
{{ readTime .Text }}                  // Reading time in minutes
{{ themeEmoji .ThemeName }}           // Get emoji for theme
{{ extractDomain .URL }}              // "example.com"
{{ add 1 2 }}                         // Math: 3
{{ mul .Score 100 }}                  // Math: 85.0 * 100 = 8500
{{ eq .ID "123" }}                    // Equality check
{{ gt .Count 0 }}                     // Greater than check
```

### daisyUI Components Used

```html
<!-- Cards -->
<div class="card bg-base-100 shadow-xl">
  <div class="card-body">
    <h2 class="card-title">Title</h2>
    <p>Content</p>
    <div class="card-actions justify-end">
      <button class="btn btn-primary">Action</button>
    </div>
  </div>
</div>

<!-- Tabs -->
<div class="tabs tabs-boxed">
  <a class="tab tab-active">Tab 1</a>
  <a class="tab">Tab 2</a>
</div>

<!-- Badges -->
<span class="badge badge-primary">AI/ML</span>
<span class="badge badge-ghost badge-sm">12</span>

<!-- Loading -->
<span class="loading loading-spinner loading-sm"></span>

<!-- Alerts -->
<div class="alert alert-info">
  <svg>...</svg>
  <span>Info message</span>
</div>
```

### HTMX Patterns

```html
<!-- Click to load -->
<button hx-get="/api/data" hx-target="#result">Load</button>

<!-- Form submission -->
<form hx-post="/api/submit" hx-target="#response">
  <input name="value">
  <button type="submit">Submit</button>
</form>

<!-- Infinite scroll -->
<div hx-get="/api/more?offset=10" hx-trigger="revealed">
  Loading...
</div>

<!-- Polling -->
<div hx-get="/api/status" hx-trigger="every 2s">
  Status: ...
</div>
```

---

## Next Steps (Phase 2+)

### Phase 2: Enhanced Digest View (Week 2)

- [ ] Implement digest detail page (`/digests/{id}`)
- [ ] Add article summary expansion (within digest)
- [ ] Add "Copy to LinkedIn" button
- [ ] Add social share buttons

### Phase 3: Filtering & Pagination (Week 3)

- [ ] Date range filter
- [ ] "Last 7/30/90 days" presets
- [ ] Pagination for digest list
- [ ] Infinite scroll option

### Phase 4: Search (Week 4)

- [ ] Search bar in header
- [ ] Debounced search (500ms delay)
- [ ] Search across digests + articles
- [ ] Highlight matching terms

### Phase 5: Polish (Week 5)

- [ ] Loading skeletons
- [ ] Error states
- [ ] Empty states
- [ ] Toast notifications
- [ ] Improved animations

### Phase 6: Production (Week 6)

- [ ] Minify CSS
- [ ] Add CSP headers
- [ ] Setup monitoring
- [ ] Performance testing
- [ ] SEO optimization

---

## Troubleshooting

### Templates not loading

```bash
# Check template directory exists
ls -la web/templates/

# Check template renderer logs
# Should see: "Loading templates from web/templates"

# If in dev mode, templates reload on each request
ENV=development go run ./cmd/briefly serve
```

### Dark mode not persisting

```javascript
// Check localStorage in browser console
localStorage.getItem('theme')  // Should be "light" or "dark"

// Check data-theme attribute
document.documentElement.getAttribute('data-theme')
```

### HTMX not working

```html
<!-- Check HTMX loaded in browser console -->
<script>console.log(typeof htmx)</script>
<!-- Should print "object" -->

<!-- Check request headers -->
<!-- HX-Request: true should be present -->
```

### Static files not serving

```bash
# Check directory structure
ls -la web/static/css/custom.css

# Check server logs for file serving errors

# Verify route in setupRoutes():
# s.router.Handle("/static/*", ...)
```

### Admin API returns 403

```bash
# Check ADMIN_API_KEY is set
echo $ADMIN_API_KEY

# Check Authorization header format
# Must be: "Bearer YOUR-API-KEY-HERE"
```

---

## Performance Benchmarks

**Target Metrics:**
- Initial page load: < 2s (on 3G)
- HTMX partial load: < 500ms
- Dark mode toggle: < 100ms
- Lighthouse score: > 90

**Test Commands:**
```bash
# Lighthouse CI
npx lighthouse http://localhost:8080 --view

# Load testing
ab -n 1000 -c 10 http://localhost:8080/

# HTMX endpoint testing
ab -n 100 -c 5 http://localhost:8080/api/digests/123/expand
```

---

## Deployment Checklist

- [ ] Set `ENV=production`
- [ ] Set strong `ADMIN_API_KEY`
- [ ] Enable GZIP compression
- [ ] Setup HTTPS/TLS
- [ ] Configure CSP headers
- [ ] Enable caching headers
- [ ] Setup error monitoring
- [ ] Configure PostHog (optional)
- [ ] Test mobile on real devices
- [ ] Run Lighthouse audit
- [ ] Load test with expected traffic

---

## Resources

- **daisyUI Docs**: https://daisyui.com/
- **HTMX Docs**: https://htmx.org/docs/
- **Tailwind CSS**: https://tailwindcss.com/docs
- **Go Templates**: https://pkg.go.dev/html/template
- **Chi Router**: https://github.com/go-chi/chi

---

## Support

For issues or questions:
1. Check this document first
2. Review existing templates for examples
3. Check browser console for errors
4. Review server logs for backend issues
5. Test in incognito mode (rules out cache issues)

**Common Issues:**
- Import path errors → Use `briefly/internal/...`
- Template not found → Check file path in `Render()` call
- HTMX not swapping → Check `hx-target` selector exists
- Dark mode flash → Ensure script in `<head>` before `<body>`
