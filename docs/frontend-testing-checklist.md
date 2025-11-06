# Frontend Testing Checklist - Text-Based UI

## Test Environment Setup

1. **Start the server:**
   ```bash
   ./briefly server
   ```

2. **Access the application:**
   - Homepage: `http://localhost:8080/`
   - Theme management: `http://localhost:8080/themes`
   - Manual submission: `http://localhost:8080/submit`

## Phase 5: Manual Testing

### Homepage Tests

#### Visual/Layout
- [ ] Page loads with Nord color palette (dark background #2E3440)
- [ ] Theme filter appears at top with "All" and theme links
- [ ] Digest list shows up to 5 recent digests
- [ ] Each digest shows: title, time ago, theme badge, article count, preview
- [ ] Footer shows keyboard hints (j/k navigate, ? shortcuts)
- [ ] Empty state shows if no digests for selected theme

#### Keyboard Navigation
- [ ] **j** - Navigate down through digest list
- [ ] **k** - Navigate up through digest list
- [ ] **Enter** or **l** - Open selected digest detail page
- [ ] **g g** - Jump to top of page
- [ ] **G** - Jump to bottom of page
- [ ] **?** - Open keyboard shortcuts modal
- [ ] **Esc** - Close keyboard shortcuts modal

#### Mouse Interaction
- [ ] Click digest title to open detail page
- [ ] Click theme filter links to filter by theme
- [ ] Selected digest has visual highlight (blue left border, grey background)
- [ ] Theme filter partial update works (HTMX)

### Digest Detail Page Tests

#### Visual/Layout
- [ ] Page shows digest title and date
- [ ] Summary section displays with key moments (if available)
- [ ] Perspectives section shows topic clusters with articles
- [ ] All Articles section lists all articles
- [ ] Each article shows title, source domain, time ago
- [ ] Article links open in new tab

#### Keyboard Navigation
- [ ] **s** - Jump to Summary section
- [ ] **k** - Jump to Key Moments (if present)
- [ ] **p** - Jump to Perspectives section
- [ ] **a** - Jump to All Articles section
- [ ] **j/k** - Navigate through articles in current section
- [ ] **Enter** or **o** - Open selected article in new tab
- [ ] **y** - Copy summary to clipboard (check confirmation message)
- [ ] **h** - Return to homepage
- [ ] **?** - Open keyboard shortcuts modal

#### Content Rendering
- [ ] Markdown summary renders correctly (bold, lists, links)
- [ ] Key moments extracted from summary (bullet points)
- [ ] Perspectives grouped by topic cluster
- [ ] No duplicate articles between Perspectives and All Articles

### Responsive Design Tests

#### Desktop (1920x1080)
- [ ] Base font size: 16px
- [ ] Line height: 1.6
- [ ] Content max-width: 900px, centered
- [ ] Digest items have adequate spacing
- [ ] Keyboard shortcuts modal displays centered

#### Tablet (768x1024)
- [ ] Base font size: 15px
- [ ] Content adjusts to viewport width
- [ ] Touch-friendly target sizes (44px minimum)
- [ ] Modal overlays properly

#### Mobile (375x667)
- [ ] Base font size: 14px
- [ ] Single column layout
- [ ] Text wraps properly
- [ ] Footer keyboard hints remain visible
- [ ] Modal takes full viewport width

### Cross-Browser Tests

#### Chrome/Chromium
- [ ] All keyboard shortcuts work
- [ ] Nord colors render correctly
- [ ] HTMX partial updates work
- [ ] PostHog analytics fires (check console)

#### Firefox
- [ ] All keyboard shortcuts work
- [ ] Nord colors render correctly
- [ ] HTMX partial updates work

#### Safari (macOS/iOS)
- [ ] All keyboard shortcuts work
- [ ] Nord colors render correctly
- [ ] HTMX partial updates work
- [ ] Touch navigation works on iOS

### Accessibility Tests

- [ ] All interactive elements keyboard accessible
- [ ] Tab navigation works logically
- [ ] Focus states visible (blue outline)
- [ ] Links have clear hover states
- [ ] Modal has proper ARIA attributes
- [ ] Modal close button accessible
- [ ] Semantic HTML used throughout

### Performance Tests

- [ ] Initial page load < 1 second
- [ ] Theme filter updates instantaneous
- [ ] Digest detail page loads quickly
- [ ] No console errors
- [ ] No layout shift during load

## Known Issues / Future Work

- [ ] Citation links [1], [2], [3] not yet implemented
- [ ] Banner images not yet implemented
- [ ] PostHog events may need verification
- [ ] Consider adding loading states for HTMX updates

## Browser Developer Tools Checks

### Console
- [ ] No JavaScript errors
- [ ] PostHog tracking events visible (if enabled)
- [ ] HTMX requests logged

### Network
- [ ] CSS loaded successfully
- [ ] JS loaded successfully
- [ ] HTMX requests return 200 status
- [ ] No 404 errors

### Elements/Inspector
- [ ] Nord CSS classes applied correctly
- [ ] `data-digest-id` attributes present on digest items
- [ ] `data-url` attributes present on article items
- [ ] Modal display toggles correctly

## Sign-off

- [ ] All homepage tests pass
- [ ] All digest detail tests pass
- [ ] Responsive design verified on all breakpoints
- [ ] Cross-browser compatibility confirmed
- [ ] No critical bugs found
- [ ] Ready for production deployment

**Tested by:** _______________
**Date:** _______________
**Browser versions tested:** _______________
