# Keyboard Navigation Architecture

**Version:** 2.0
**Status:** Design
**Last Updated:** 2025-01-05

## Problem Statement

The current keyboard navigation has several issues:

1. **Ambiguous key mappings**: `l` moves to digest detail when in theme filter, but user expects it to change themes
2. **No clear focus model**: User doesn't know if they're navigating themes or digests
3. **Lack of visual feedback**: No indication of what's currently focused or active
4. **Mixed mental models**: Trying to support both "quick actions" and "navigation" with same keys

**Example of current problem:**
```
User state: Theme filter focused, "AI/ML" theme selected
User presses: l (expecting to go to next theme)
Actual result: Navigates to digest detail page
Expected result: Move to "Cloud" theme
```

## Design Goals

1. **Clear mental model**: Vim-inspired contexts make it obvious what keys do based on what you're interacting with
2. **Visual clarity**: Always show what's focused and what context you're in
3. **Predictable behavior**: Same keys do consistent things within a context
4. **Content-first approach**: Start in DIGEST context (news first), navigate up to themes/nav as needed
5. **Keyboard-first**: Fully functional without mouse, but doesn't break mouse usage

## Architecture Overview

### Focus Contexts (Vim-inspired)

The navigation system operates in different **focus contexts** based on what section the user is interacting with:

| Context | Purpose | Entry | Exit |
|---------|---------|-------|------|
| **DIGEST** | Navigate and interact with digest cards | Default state on page load | Esc, h |
| **THEME** | Navigate and multi-select themes | `h` from DIGEST or `k` from DIGEST at top | Esc, Enter |
| **NAV** | Navigate header links (Home, About) | `k` from THEME or Esc+`k` | Esc, Enter |

**Context Transitions:**
```
                     NAV
                      ↓
                   (k/j)
                      ↓
                    THEME
                      ↓
                   (k/j)
                      ↓
                   DIGEST (default)


         Esc from any → Returns to DIGEST
```

### Focus Hierarchy

Each page has a tree structure that keyboard navigation follows:

```
Homepage
├── Header [FOCUSABLE - NAV context]
│   ├── Logo (/) [0]
│   ├── Nav: Home (/) [1]
│   └── Nav: About (/about) [2]
├── Main Content
│   ├── Theme Filter Section [FOCUSABLE - THEME context]
│   │   ├── All [0]
│   │   ├── AI/ML [1]
│   │   ├── Cloud [2]
│   │   └── ... [n]
│   └── Digest List Section [FOCUSABLE - DIGEST context]
│       ├── Digest Card [0]
│       ├── Digest Card [1]
│       └── ... [n]
└── Footer
```

**Focusable Sections:**
- `nav` - Header navigation links (Logo, Home, About)
- `theme-filter` - The theme tabs/pills (multi-select)
- `digest-list` - The list of digest cards (default focus on page load)

## Key Bindings

### DIGEST Context (Default)

**Purpose:** Navigate and interact with digest cards

| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Next digest | Highlight next digest card |
| `k` / `↑` | Previous digest | Highlight previous digest card (or move to THEME context if at top) |
| `Enter` / `o` | Open digest | Navigate to digest detail page |
| `h` | Go to themes | Move to THEME context |
| `y` | Copy summary | Copy digest summary to clipboard |
| `Esc` | Reset | Return to first digest, stay in DIGEST |
| `?` | Show help modal | Display keyboard shortcuts for DIGEST context |

**Auto-behaviors:**
- **Default state**: Page loads in DIGEST context with first digest highlighted
- Scroll digest into view when navigating
- `k` at index 0 moves to THEME context (upward navigation)
- `j` at last digest wraps to first digest
- Smooth scroll animation when moving between digests

**Visual State:**
- Context indicator shows "DIGEST"
- Current digest has bold outline + background highlight + subtle elevation
- Smooth scroll animation when moving between digests

**Example:**
```
State: Page loads → DIGEST context, index=0 (first digest highlighted)
User presses: j
Result: index=1, second digest highlighted and scrolled into view
User presses: Enter
Result: Navigate to /digests/{id}

Alternative flow:
User presses: k (while at index 0)
Result: Move to THEME context with current theme highlighted
```

### NAV Context

**Purpose:** Navigate header links (Home, About, etc.)

| Key | Action | Description |
|-----|--------|-------------|
| `h` / `←` | Previous link | Move to previous nav link |
| `l` / `→` | Next link | Move to next nav link |
| `Enter` / `o` | Follow link | Navigate to selected page |
| `j` / `↓` | Move down | Go to THEME context |
| `Esc` | Exit | Return to DIGEST context |

**Auto-behaviors:**
- Wraps around (last link → first link)
- Starts at Logo (index 0)

**Visual State:**
- Context indicator shows "NAV"
- Current link has subtle outline
- No background change (header styling maintained)

**Example:**
```
State: DIGEST context, user at top
User presses: k (while in DIGEST at index 0)
Result: Move up to THEME context
User presses: k (while in THEME)
Result: Move to NAV context, Logo highlighted
User presses: l
Result: "Home" link highlighted
User presses: l
Result: "About" link highlighted
User presses: Enter
Result: Navigate to /about
```

### THEME Context

**Purpose:** Navigate and multi-select themes for filtering

| Key | Action | Description |
|-----|--------|-------------|
| `h` / `←` | Previous theme | Move to previous theme tab |
| `l` / `→` | Next theme | Move to next theme tab |
| `Space` | Toggle selection | Select/deselect current theme (multi-select) |
| `Enter` | Apply filters | Apply selected themes and move to DIGEST context |
| `j` / `↓` | Move down | Go to DIGEST context (without applying changes) |
| `k` / `↑` | Move up | Go to NAV context |
| `Esc` | Cancel | Discard changes and return to DIGEST context |

**Multi-select Behavior:**
- **Space bar** toggles current theme selection (checkmark appears)
- Can select multiple themes (OR filter: show digests matching ANY selected theme)
- "All" is mutually exclusive - selecting "All" deselects other themes
- Selecting any specific theme deselects "All"
- **Enter** applies selections and filters digests, then moves to DIGEST context
- Visual feedback: selected themes have checkmark icon

**Auto-behaviors:**
- Wraps around (last theme → first theme)
- `j` exits to DIGEST without applying (preserves old filters)
- `Esc` cancels changes and returns to DIGEST with old filters
- Enter applies and moves to DIGEST with new filters

**Visual State:**
- Context indicator shows "THEME"
- Highlighted theme (navigation cursor) has bold outline
- Selected themes (for filtering) have checkmark + background
- Smooth CSS transition when changing themes

**Example:**
```
State: DIGEST context, user presses h
Result: Enter THEME context, current active theme highlighted

User presses: Space (on "AI/ML")
Result: "AI/ML" selected (checkmark appears), "All" deselected

User presses: l (navigate to next)
Result: Cursor moves to "Cloud" theme

User presses: Space (on "Cloud")
Result: "Cloud" also selected (both AI/ML and Cloud have checkmarks)

User presses: Enter
Result: Digests filter to show AI/ML OR Cloud articles, move to DIGEST context

Alternative - cancel:
User presses: Esc (instead of Enter)
Result: Discard selections, return to DIGEST context with previous filters
```

## State Management

### JavaScript State Object

```javascript
const KeyboardNav = {
  // Current state
  context: 'DIGEST',        // 'NAV', 'THEME', 'DIGEST' (default: DIGEST)

  // Indices for each context
  navIndex: 0,              // Current nav link index (0 = Logo)
  themeIndex: 0,            // Current theme cursor index (0 = "All")
  digestIndex: 0,           // Current digest index (default: 0)

  // Theme selection state (for multi-select)
  selectedThemes: ['all'],  // Array of selected theme IDs (default: ['all'])
  pendingThemeSelection: [], // Temporary selection during THEME context

  // DOM references (cached)
  navLinks: [],             // Array of nav link elements
  themeTabs: [],            // Array of theme tab elements
  digestCards: [],          // Array of digest card elements

  // State management
  init() { },
  setContext(newContext) { },

  // Navigation handlers
  handleNavContext(key) { },
  handleThemeContext(key) { },
  handleDigestContext(key) { },

  // Theme multi-select
  toggleThemeSelection(themeId) { },
  applyThemeFilters() { },
  cancelThemeSelection() { },

  // Highlighting
  highlightNav(index) { },
  highlightTheme(index) { },
  highlightDigest(index) { },

  // Utilities
  updateVisualState() { },
  scrollIntoView(element) { },
  showContextIndicator() { },
  updateThemeCheckmarks() { }
}
```

### State Transitions

**Context transitions:**
```javascript
// Page load → DIGEST (default)
window.addEventListener('DOMContentLoaded', () => {
  KeyboardNav.init()
  KeyboardNav.setContext('DIGEST')
  KeyboardNav.highlightDigest(0) // Highlight first digest
})

// DIGEST → THEME (h key)
if (context === 'DIGEST' && key === 'h') {
  setContext('THEME')
  pendingThemeSelection = [...selectedThemes] // Clone current selection
  highlightTheme(themeIndex)
}

// DIGEST → THEME (k at index 0)
if (context === 'DIGEST' && key === 'k' && digestIndex === 0) {
  setContext('THEME')
  pendingThemeSelection = [...selectedThemes]
  highlightTheme(themeIndex)
}

// THEME → NAV (k key)
if (context === 'THEME' && key === 'k') {
  setContext('NAV')
  highlightNav(0) // Start at Logo
}

// THEME → DIGEST (Enter - apply)
if (context === 'THEME' && key === 'Enter') {
  selectedThemes = [...pendingThemeSelection]
  applyThemeFilters() // Trigger HTMX request
  setContext('DIGEST')
  highlightDigest(0)
}

// THEME → DIGEST (j - discard)
if (context === 'THEME' && key === 'j') {
  pendingThemeSelection = [] // Discard changes
  setContext('DIGEST')
  highlightDigest(0)
}

// NAV → THEME (j key)
if (context === 'NAV' && key === 'j') {
  setContext('THEME')
  highlightTheme(themeIndex)
}

// Any context → DIGEST (Esc)
if (key === 'Esc') {
  if (context === 'THEME') {
    pendingThemeSelection = [] // Discard changes
  }
  setContext('DIGEST')
  highlightDigest(digestIndex) // Maintain current position
}
```

## Mouse Integration

### Design Philosophy

**Principle**: Keyboard state should always reflect user's actual focus, whether they're using mouse or keyboard.

When a user clicks on an element, the keyboard navigation state should update to match, so that pressing a keyboard shortcut acts on the clicked element.

### Click Handlers

**Click on Nav Link:**
```javascript
navLinks.forEach((link, index) => {
  link.addEventListener('click', (e) => {
    // Update state to match click
    navIndex = index
    setContext('NAV')
    highlightNav(index)

    // Let default click behavior proceed (navigate to page)
  })
})
```

**Click on Theme Tab:**
```javascript
themeTabs.forEach((tab, index) => {
  tab.addEventListener('click', (e) => {
    // Update keyboard cursor position
    themeIndex = index
    setContext('THEME')
    highlightTheme(index)

    // Toggle selection (like Space bar)
    const themeId = tab.dataset.themeId
    toggleThemeSelection(themeId)

    // Auto-apply single theme click (instant filter)
    // This differs from keyboard (Space to select, Enter to apply)
    selectedThemes = [themeId]
    applyThemeFilters()
  })
})
```

**Click on Digest Card:**
```javascript
digestCards.forEach((card, index) => {
  card.addEventListener('click', (e) => {
    // Don't navigate if clicking a link inside the card
    if (e.target.tagName === 'A') return

    // Update state to match click
    digestIndex = index
    setContext('DIGEST')
    highlightDigest(index)

    // Option 1: Just update focus (user can press Enter to open)
    // Option 2: Auto-open on click (navigate to detail page)
    // Recommended: Option 1 for consistency
  })
})
```

### Hybrid Usage Patterns

**Pattern 1: Mouse to Keyboard**
```
User clicks digest #3 → digestIndex = 3, DIGEST context, highlight #3
User presses j → digestIndex = 4, highlight #4 (continues from click)
User presses Enter → Navigate to digest #4 detail
```

**Pattern 2: Keyboard to Mouse**
```
User presses j j j → digestIndex = 3, DIGEST context, highlight #3
User clicks digest #7 → digestIndex = 7, highlight #7
User presses Enter → Navigate to digest #7 detail (not #3)
```

**Pattern 3: Theme Click then Keyboard**
```
User clicks "AI/ML" theme → themeIndex = 1, THEME context, filter to AI/ML
User presses l → themeIndex = 2, "Cloud" theme cursor
User presses Space → "Cloud" selected
User presses Enter → Apply both AI/ML and Cloud filters, move to DIGEST
```

### Visual Consistency

**Both mouse and keyboard should show same highlights:**
- Click a digest → Same outline/shadow as keyboard `j`/`k` navigation
- Click a theme → Same cursor highlight as keyboard `h`/`l` navigation
- Hover should be visually distinct from selection/cursor

**CSS Pseudo-classes:**
```css
/* Hover (mouse intent) - subtle */
.digest-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
}

/* Keyboard/click highlight (focus/selection) - bold */
.digest-card.highlighted {
  outline: 3px solid var(--nord8);
  outline-offset: 2px;
  background: var(--nord1);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

/* Both can coexist */
.digest-card.highlighted:hover {
  /* Highlighted wins, hover adds subtle extra feedback */
  box-shadow: 0 5px 14px rgba(0, 0, 0, 0.35);
}
```

### Context Switching on Click

**Rules:**
1. Clicking any element **updates context** to match that element's context
2. Clicking within current context **preserves context**, updates index
3. Context indicator updates to reflect clicks

**Example:**
```
State: DIGEST context, digestIndex = 2

User clicks theme tab "Cloud"
→ setContext('THEME'), themeIndex = 2, apply filter

User clicks digest #5
→ setContext('DIGEST'), digestIndex = 5, highlight #5

User clicks nav "About" link
→ setContext('NAV'), navIndex = 2, navigate to /about
```

### Accessibility Considerations

**Focus vs Highlight:**
- **Browser focus** (`:focus`) - for accessibility, follows Tab key
- **Highlight** (`.highlighted`) - for keyboard nav, follows j/k/h/l

Both can coexist:
```javascript
function highlightDigest(index) {
  // Visual highlight
  digestCards[index].classList.add('highlighted')

  // Also set browser focus for screen readers
  digestCards[index].focus()

  // ARIA
  digestCards[index].setAttribute('aria-current', 'true')
}

// On click, sync both
digestCards[index].addEventListener('click', () => {
  highlightDigest(index) // Updates both visual + a11y
  setContext('DIGEST')
})
```

### Touch Devices

**Philosophy**: Touch events should work just like click events - update state and show highlights. Mobile devices can have keyboards too!

**Behavior:**
- Touch on element → Update state, show highlight (same as click)
- Visual highlights always visible (mobile users benefit from seeing selection)
- Context indicator auto-hides on touch (shown on keyboard input)
- Swipe gestures for navigation (optional enhancement)

**Touch Event Integration:**

```javascript
// Touch works exactly like click - updates keyboard state
digestCards.forEach((card, index) => {
  // Both touch and click update state
  const updateState = () => {
    digestIndex = index
    setContext('DIGEST')
    highlightDigest(index)
  }

  card.addEventListener('click', updateState)
  card.addEventListener('touchend', (e) => {
    e.preventDefault() // Prevent duplicate click event
    updateState()
  })
})

// Swipe gestures (optional - adds j/k navigation via swipe)
let touchStartY = 0

document.addEventListener('touchstart', (e) => {
  touchStartY = e.touches[0].clientY
})

document.addEventListener('touchend', (e) => {
  const touchEndY = e.changedTouches[0].clientY
  const deltaY = touchStartY - touchEndY

  // Swipe up = next (j)
  if (deltaY > 50 && context === 'DIGEST') {
    nextDigest()
  }
  // Swipe down = previous (k)
  if (deltaY < -50 && context === 'DIGEST') {
    prevDigest()
  }
})
```

**Context Indicator Visibility:**

```javascript
let lastInputType = 'unknown' // 'keyboard', 'mouse', 'touch'

// Show indicator on keyboard input
document.addEventListener('keydown', (e) => {
  if (lastInputType !== 'keyboard') {
    lastInputType = 'keyboard'
    document.body.classList.add('keyboard-mode')
    showContextIndicator()
  }
})

// Hide indicator on touch (but keep highlights!)
document.addEventListener('touchstart', () => {
  if (lastInputType !== 'touch') {
    lastInputType = 'touch'
    document.body.classList.remove('keyboard-mode')
    hideContextIndicator() // Hide indicator only, not highlights
  }
})

// Mouse/click - show indicator if they seem to be using keyboard workflow
// (e.g., clicking then pressing keys)
document.addEventListener('click', () => {
  if (lastInputType === 'keyboard') {
    // Keep indicator visible if user is mixing mouse + keyboard
  } else if (lastInputType !== 'mouse') {
    lastInputType = 'mouse'
    // Don't show indicator for pure mouse users
  }
})
```

**CSS - Highlights Always Visible:**

```css
/* Highlights visible on ALL devices */
.digest-card.highlighted {
  outline: 3px solid var(--nord8);
  outline-offset: 2px;
  background: var(--nord1);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  /* No @media query - always shown */
}

/* Context indicator only hidden when not in keyboard mode */
.context-indicator {
  position: fixed;
  bottom: 1rem;
  right: 1rem;
  opacity: 1;
  transition: opacity 0.3s ease;
}

/* Hide indicator when not in keyboard mode */
body:not(.keyboard-mode) .context-indicator {
  opacity: 0;
  pointer-events: none;
}
```

**Mobile-Specific Enhancements:**

```javascript
// Detect if device has physical keyboard
function hasPhysicalKeyboard() {
  // If any keyboard event has been received, assume keyboard exists
  return document.body.classList.contains('keyboard-mode')
}

// Long press for context menu (future enhancement)
let longPressTimer
digestCards.forEach((card, index) => {
  card.addEventListener('touchstart', (e) => {
    longPressTimer = setTimeout(() => {
      // Show context menu: Copy, Open, etc.
      showDigestContextMenu(index)
    }, 500)
  })

  card.addEventListener('touchend', () => {
    clearTimeout(longPressTimer)
  })

  card.addEventListener('touchmove', () => {
    clearTimeout(longPressTimer)
  })
})
```

**Why This Approach:**
1. ✅ **iPad with keyboard** - Full keyboard nav works, highlights visible
2. ✅ **Phone with Bluetooth keyboard** - Full keyboard nav works
3. ✅ **Pure touch users** - Tap to select, highlights show selection, no keyboard clutter
4. ✅ **Hybrid users** - System adapts based on input method
5. ✅ **Accessibility** - Highlights always visible for all users

### Edge Cases

**Rapid clicking:**
- Each click updates state immediately
- No debouncing needed (user intent is clear)

**Click during keyboard navigation:**
- Click wins, keyboard state updates to match
- Next keyboard press continues from clicked position

**Click on already-highlighted element:**
- No state change, but event still fires
- For digest: Could toggle expand/collapse (future enhancement)

**Click link inside digest card:**
```javascript
card.addEventListener('click', (e) => {
  // If clicking article link, don't hijack
  if (e.target.closest('a.article-link')) {
    return // Let link navigate
  }

  // Otherwise, update keyboard focus
  highlightDigest(index)
  setContext('DIGEST')
})
```

## Visual Feedback

### Context Indicator

**Position:** Bottom-right corner of screen (fixed)

**Design:**
```
┌────────────────────┐
│  [Icon] CONTEXT    │
└────────────────────┘
```

**Styles:**
- NAV: Purple background, "🔗 NAV"
- THEME: Blue background, "🎨 THEME (multi-select)"
- DIGEST: Green background, "📰 DIGEST" (default)

**CSS:**
```css
.context-indicator {
  position: fixed;
  bottom: 1rem;
  right: 1rem;
  padding: 0.5rem 1rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  font-weight: 600;
  opacity: 0.9;
  transition: all 0.2s ease;
  z-index: 100;
}

.context-nav { background: var(--nord15); color: var(--nord0); }
.context-theme { background: var(--nord10); color: var(--nord0); }
.context-digest { background: var(--nord14); color: var(--nord0); }
```

### Focus Indicators

**Nav Link Highlight (NAV context):**
```css
.nav-link.highlighted {
  outline: 2px solid var(--nord15);
  outline-offset: 2px;
  border-radius: 0.25rem;
  background: rgba(180, 142, 173, 0.1); /* Subtle purple tint */
}
```

**Theme Tab Highlight (THEME context):**
```css
/* Navigation cursor */
.theme-tab.cursor {
  outline: 3px solid var(--nord8);
  outline-offset: 2px;
  transform: scale(1.05);
  transition: all 0.2s ease;
}

/* Selected for filtering (checkmark) */
.theme-tab.selected {
  background: var(--nord10);
  color: var(--nord0);
  font-weight: 600;
}

.theme-tab.selected::after {
  content: "✓";
  margin-left: 0.5rem;
  font-weight: bold;
}
```

**Digest Card Highlight (DIGEST context):**
```css
.digest-card.highlighted {
  outline: 3px solid var(--nord8);
  outline-offset: 2px;
  background: var(--nord1);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transition: all 0.2s ease;
}
```

### Accessibility

- All visual states must also update ARIA attributes
- `aria-current="true"` on highlighted elements
- `role="navigation"` on header and theme-filter
- `role="list"` on digest-list
- Announce context changes to screen readers

```javascript
function highlightDigest(index) {
  // Visual update
  digestCards[index].classList.add('highlighted')

  // Accessibility update
  digestCards[index].setAttribute('aria-current', 'true')
  digestCards[index].focus() // Sync keyboard and screen reader focus

  // Announce to screen reader
  announceToScreenReader(`DIGEST context: Digest ${index + 1} of ${digestCards.length}`)
}

function setContext(newContext) {
  const previousContext = this.context
  this.context = newContext

  // Update context indicator
  showContextIndicator()

  // Announce to screen reader
  const announcements = {
    'NAV': 'Navigation context: Use h/l to navigate links, Enter to follow',
    'THEME': 'Theme context: Use h/l to navigate, Space to select, Enter to apply filters',
    'DIGEST': 'Digest context: Use j/k to navigate digests, Enter to open'
  }
  announceToScreenReader(announcements[newContext])
}
```

## Implementation Plan

### Phase 1: Core State Management (Day 1-2)
- [ ] Implement state management object with context tracking
- [ ] Add context indicator UI component (bottom-right, auto-hide on touch)
- [ ] Add input type detection (keyboard/mouse/touch)
- [ ] Add DIGEST context (default) with key handlers
- [ ] Add digest highlight styling and smooth scroll
- [ ] **Add click handlers for digest cards** (update state on click)
- [ ] **Add touch handlers for digest cards** (same behavior as click)
- [ ] Test: Page loads in DIGEST, j/k navigation works
- [ ] Test: Click digest updates digestIndex, keyboard continues from click
- [ ] Test: Touch digest shows highlight, context indicator hides

### Phase 2: THEME Context with Multi-select (Day 3-4)
- [ ] Implement THEME context key handlers (h/l navigation, Space toggle)
- [ ] Add pending selection state management
- [ ] Add theme highlight (cursor) and selected (checkmark) styling
- [ ] **Add click handlers for theme tabs** (auto-apply on click)
- [ ] Integrate Enter to apply filters via HTMX
- [ ] Test context transitions (DIGEST ↔ THEME, apply/cancel)
- [ ] Test: Click theme applies filter, keyboard can multi-select afterward

### Phase 3: NAV Context (Day 5)
- [ ] Implement NAV context key handlers (h/l for links)
- [ ] Add nav link highlight styling
- [ ] **Add click handlers for nav links** (update state before navigation)
- [ ] Test upward navigation (DIGEST → THEME → NAV)
- [ ] Test Enter to follow links
- [ ] Test: Click nav link updates navIndex

### Phase 4: Help Modal & Polish (Day 6-7)
- [ ] Update `?` help modal to be context-aware
- [ ] Add smooth CSS transitions for all highlights
- [ ] Add accessibility attributes and screen reader announcements
- [ ] Add "copy summary" functionality (y key in DIGEST)
- [ ] **Add swipe gesture navigation** (optional: swipe up/down for j/k)
- [ ] **Add long-press context menu** (optional: future enhancement)
- [ ] Test on mobile devices (iOS Safari, Chrome Android)
- [ ] Test with iPad + keyboard (hybrid input)
- [ ] Write integration tests

### Files to Create/Modify

**New files:**
- `web/static/js/keyboard-nav-v2.js` - New context-based navigation implementation
- `web/static/css/keyboard-nav.css` - Navigation-specific styles (context indicator, highlights)

**Modified files:**
- `web/templates/pages/home.html` - Add context indicator, data attributes for nav/theme/digest elements
- `web/static/js/keyboard-nav.js` - Deprecate for homepage (keep for digest-detail page)
- `web/templates/layouts/base.html` - Update keyboard shortcuts modal to be context-aware

## Edge Cases

### Empty States
- **No digests**: DIGEST context shows message "No digests available. Press `h` to filter themes"
- **Single theme**: THEME context `h`/`l` should do nothing (only "All" exists)
- **No themes selected**: If user deselects all themes in THEME context, "All" is auto-selected

### HTMX Updates
- **Digest list reload**: Stay in DIGEST context, reset digestIndex to 0, highlight first digest
- **Theme filter applied**: Update selectedThemes array, reload digest list, stay in DIGEST context
- **Pending selections**: Applying theme filter clears pendingThemeSelection state

### Multiple Tabs
- **Focus loss**: Context state is page-scoped, not persisted across sessions
- **Page reload**: Always starts in DIGEST context (default)

### Mobile/Touch
- **Touch events**: Touch updates state just like clicks - highlights always visible
- **Context indicator**: Auto-hides on touch input, shows on keyboard input
- **iPad/tablet with keyboard**: Full keyboard nav + highlights work perfectly
- **Swipe gestures**: Optional swipe up/down for j/k navigation
- **Long press**: Future enhancement for context menus (copy, open, etc.)
- **Hybrid users**: System detects input type and adapts indicator visibility

### Navigation Conflicts
- **k at DIGEST index 0**: Moves to THEME context (upward navigation)
- **k at THEME (any index)**: Moves to NAV context (upward navigation)
- **k at NAV**: Stays in NAV, wraps to last link (no upward context available)

## Future Enhancements

### Phase 5: Advanced Features
- [ ] Command palette (`:` key) for quick actions
- [ ] Search mode (`/` key) for filtering digests
- [ ] Bookmarks (`m` key) to mark digests for later
- [ ] History (`Ctrl+O` / `Ctrl+I`) for navigation history

### Phase 6: Cross-Page Navigation
- [ ] Global keyboard shortcuts (work on all pages)
- [ ] Digest detail page uses same navigation model
- [ ] `g` + `h` → go home, `g` + `a` → go about

## Design Decisions

1. **Persist context across page reloads?**
   - ✅ **Decision**: No, always start in DIGEST context (default)
   - Rationale: Consistent, predictable experience for all users

2. **Enter behavior in THEME context?**
   - ✅ **Decision**: Enter confirms multi-selection and moves to DIGEST context
   - Rationale: Enables multi-select workflow (Space to select, Enter to apply)
   - Alternative considered: Auto-filter on selection (rejected - doesn't support multi-select)

3. **Visual hints on first visit?**
   - ✅ **Decision**: No additional hints
   - Rationale: Footer already shows "Press ? for keyboard shortcuts"

4. **Keyboard shortcuts help (`?`)?**
   - ✅ **Decision**: Modal overlay with context-aware sections
   - Rationale: Shows all shortcuts but highlights relevant ones for current context
   - Modal sections: NAV shortcuts, THEME shortcuts, DIGEST shortcuts

## Testing Strategy

### Unit Tests
- State transitions (context changes)
- Index bounds checking (wrap-around, navigation between contexts)
- Multi-select theme state (pending vs applied)
- DOM element highlighting

### Integration Tests
- Full user flows (DIGEST → THEME → NAV)
- Theme multi-select and apply (HTMX interaction)
- Upward navigation (k from DIGEST index 0 → THEME)
- Cancel vs apply in THEME context
- Multiple digests/themes

### Manual Testing Checklist

**Keyboard Navigation:**
- [ ] All key bindings work in each context
- [ ] Context indicator updates correctly
- [ ] Page loads in DIGEST context with first digest highlighted
- [ ] Visual highlights are clear and consistent (cursor vs selected in THEME)
- [ ] Smooth scrolling works for digest navigation
- [ ] Multi-select themes work (Space to toggle, Enter to apply)
- [ ] Esc properly cancels pending theme selections

**Mouse Integration:**
- [ ] Click digest updates digestIndex and shows highlight
- [ ] Click theme applies filter and updates themeIndex
- [ ] Click nav link updates navIndex before navigation
- [ ] Hover shows subtle feedback (distinct from highlight)
- [ ] Click then keyboard: next key press continues from clicked position
- [ ] Keyboard then click: click updates state, keyboard continues from new position

**Touch Integration:**
- [ ] Touch on digest shows highlight (same as click)
- [ ] Touch on theme applies filter and shows highlight
- [ ] Context indicator hides on touch input
- [ ] Context indicator shows on keyboard input (iPad with keyboard)
- [ ] Swipe up/down navigates digests (optional)
- [ ] Highlights always visible on mobile (no media query hiding)

**Hybrid & General:**
- [ ] No keyboard shortcuts break mouse interaction
- [ ] Mouse clicks don't break keyboard navigation state
- [ ] Touch events don't break keyboard navigation state
- [ ] Mixing touch + keyboard works seamlessly (iPad)
- [ ] Accessible with screen reader (context announcements)
- [ ] Works in all supported browsers (desktop + mobile)

## Multi-Page Navigation Architecture

### Design Philosophy

**Goal**: Create a consistent keyboard navigation model that works across all pages in the application.

**Principles:**
1. **Consistent base keys** - `?` for help, `Esc` to reset, `g` for "go to" navigation
2. **Page-specific contexts** - Each page defines its own focusable sections
3. **Global context** - NAV context available on all pages (header navigation)
4. **Shared state management** - Base navigation system inherited by all pages

### Global Contexts (All Pages)

These contexts are available on every page:

#### NAV Context
**Available on**: All pages
**Purpose**: Navigate header links (Logo, Home, About)
**Entry**: `k` from top of page content, or `g` + `n`
**Keys**:
- `h` / `l` - Navigate between links
- `Enter` - Follow link
- `j` - Move down to page content

#### GLOBAL Keys (Any Context)
**Available on**: All pages, all contexts
**Keys**:
- `?` - Show keyboard help modal (context-aware)
- `Esc` - Reset to default context for current page
- `g` + `h` - Go home (navigate to `/`)
- `g` + `a` - Go about (navigate to `/about`)
- `g` + `n` - Go to NAV context

### Page-Specific Contexts

Each page defines its own content contexts:

---

### Homepage (`/`)

**Default Context**: `DIGEST`

**Contexts:**
1. **NAV** - Header navigation (global)
2. **THEME** - Theme filter tabs
3. **DIGEST** - Digest card list (default)

**Context Flow:**
```
NAV ← k
 ↓ j
THEME ← k/h
 ↓ j/Enter
DIGEST (default)
```

**Page-Specific Keys:**
- `h` - From DIGEST, go to THEME context
- `Space` - In THEME, toggle selection
- `Enter` - In THEME, apply filters
- `y` - In DIGEST, copy summary

**Full Key Reference**: See sections above

---

### About Page (`/about`)

**Default Context**: `CONTENT`

**Contexts:**
1. **NAV** - Header navigation (global)
2. **CONTENT** - Article sections (default)

**Context Flow:**
```
NAV ← k (from top of content)
 ↓ j
CONTENT (default)
```

**CONTENT Context Keys:**
| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Next section | Scroll to next section (Problem, Solution, etc.) |
| `k` / `↑` | Previous section | Scroll to previous section (or NAV if at top) |
| `g` + `g` | Go to top | Jump to first section |
| `G` (shift+g) | Go to bottom | Jump to last section (footer) |
| `Esc` | Reset | Return to first section |

**Section Navigation:**
```javascript
// About page sections (detected by h2 headers)
const sections = [
  { id: 'problem', title: 'The Problem' },
  { id: 'solution', title: 'The Solution' },
  { id: 'how-it-works', title: 'How It Works' },
  { id: 'open-source', title: 'Open Source' },
  { id: 'built-by', title: 'Built By Engineers, For Engineers' }
]

// State
const AboutPageNav = {
  context: 'CONTENT',
  sectionIndex: 0,

  handleContentContext(key) {
    if (key === 'j' || key === 'ArrowDown') {
      this.nextSection()
    } else if (key === 'k' || key === 'ArrowUp') {
      if (sectionIndex === 0) {
        setContext('NAV') // Move to header
      } else {
        this.prevSection()
      }
    } else if (key === 'g') {
      waitForSecondKey((second) => {
        if (second === 'g') this.goToSection(0)
      })
    } else if (key === 'G') {
      this.goToSection(sections.length - 1)
    }
  },

  nextSection() {
    sectionIndex = (sectionIndex + 1) % sections.length
    scrollToSection(sectionIndex)
  }
}
```

**Visual Feedback:**
- Current section has subtle left border highlight
- Smooth scroll animation between sections
- Context indicator shows "CONTENT"

---

### Digest Detail Page (`/digests/{id}`)

**Default Context**: `ARTICLE`

**Contexts:**
1. **NAV** - Header navigation (global)
2. **SECTION** - Page sections (TL;DR, Key Moments, Articles)
3. **ARTICLE** - Individual articles within current section (default)

**Context Flow:**
```
NAV ← k (from top)
 ↓ j
SECTION ← h (exit article list)
 ↓ Enter (enter article list)
ARTICLE (default)
```

**SECTION Context Keys:**
| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Next section | Move to next section (TL;DR → Key Moments → Article Groups) |
| `k` / `↑` | Previous section | Move to previous section (or NAV if at top) |
| `Enter` / `l` | Enter section | Dive into ARTICLE context for current section |
| `g` + `g` | Go to top | Jump to TL;DR section |
| `G` | Go to bottom | Jump to last article group |

**ARTICLE Context Keys:**
| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Next article | Highlight next article in current section |
| `k` / `↑` | Previous article | Highlight previous article |
| `Enter` / `o` | Open article | Navigate to article URL (external link) |
| `h` / `Esc` | Exit to SECTION | Return to section navigation |
| `y` | Copy link | Copy article URL to clipboard |
| `e` | Expand/collapse | Toggle article summary expansion |

**Page Sections:**
```javascript
const DigestDetailNav = {
  context: 'ARTICLE',
  sectionIndex: 0,  // 0=TLDR, 1=KeyMoments, 2=ArticleGroup[0], ...
  articleIndex: 0,  // Within current section

  sections: [
    { type: 'tldr', title: 'TL;DR', articles: [] },
    { type: 'keyMoments', title: 'Key Moments', items: [...] },
    { type: 'articleGroup', title: 'AI Applications', articles: [...] },
    { type: 'articleGroup', title: 'Development Tools', articles: [...] }
  ],

  handleSectionContext(key) {
    if (key === 'j') this.nextSection()
    if (key === 'k') {
      if (sectionIndex === 0) {
        setContext('NAV')
      } else {
        this.prevSection()
      }
    }
    if (key === 'Enter' || key === 'l') {
      setContext('ARTICLE')
      highlightArticle(0) // Start at first article
    }
  },

  handleArticleContext(key) {
    if (key === 'j') this.nextArticle()
    if (key === 'k') this.prevArticle()
    if (key === 'h' || key === 'Esc') {
      setContext('SECTION')
      highlightSection(sectionIndex)
    }
    if (key === 'Enter' || key === 'o') {
      window.open(currentArticle.url, '_blank')
    }
  }
}
```

**Visual Feedback:**
- Section highlight: Left border + background tint
- Article highlight: Bold outline + elevation
- Context indicator: "SECTION" or "ARTICLE"

---

### Base Navigation System

All pages inherit from a base navigation class:

```javascript
class BasePageNav {
  constructor(pageType) {
    this.pageType = pageType  // 'home', 'about', 'digest-detail'
    this.context = this.getDefaultContext()
    this.navIndex = 0
    this.navLinks = document.querySelectorAll('header nav a')

    this.initGlobalKeys()
    this.initNavContext()
  }

  // Override in subclasses
  getDefaultContext() {
    return 'CONTENT'
  }

  // Global keys work everywhere
  initGlobalKeys() {
    document.addEventListener('keydown', (e) => {
      // Help modal
      if (e.key === '?') {
        this.showHelpModal()
        return
      }

      // Esc always resets to default
      if (e.key === 'Escape') {
        this.setContext(this.getDefaultContext())
        return
      }

      // Go-to shortcuts (g + second key)
      if (e.key === 'g') {
        this.handleGoToShortcut()
        return
      }

      // Route to context handler
      this.handleContextKey(e.key)
    })
  }

  handleGoToShortcut() {
    this.waitForSecondKey((second) => {
      if (second === 'h') window.location.href = '/'
      if (second === 'a') window.location.href = '/about'
      if (second === 'n') this.setContext('NAV')
    })
  }

  // NAV context available on all pages
  initNavContext() {
    this.navLinks.forEach((link, index) => {
      link.addEventListener('click', () => {
        this.navIndex = index
        this.setContext('NAV')
      })
    })
  }

  handleNavContext(key) {
    if (key === 'h' || key === 'ArrowLeft') {
      this.navIndex = (this.navIndex - 1 + this.navLinks.length) % this.navLinks.length
      this.highlightNav(this.navIndex)
    }
    if (key === 'l' || key === 'ArrowRight') {
      this.navIndex = (this.navIndex + 1) % this.navLinks.length
      this.highlightNav(this.navIndex)
    }
    if (key === 'Enter' || key === 'o') {
      this.navLinks[this.navIndex].click()
    }
    if (key === 'j' || key === 'ArrowDown') {
      this.setContext(this.getDefaultContext())
    }
  }

  // Override in subclasses
  handleContextKey(key) {
    if (this.context === 'NAV') {
      this.handleNavContext(key)
    }
    // Subclass handles page-specific contexts
  }

  setContext(newContext) {
    this.context = newContext
    this.updateContextIndicator()
    this.announceContextChange()
  }

  showHelpModal() {
    // Show context-aware help based on current page + context
    const helpContent = this.getHelpContent()
    // ... render modal
  }

  getHelpContent() {
    // Override in subclasses to provide page-specific help
    return {
      global: [ /* ? Esc, g+h, g+a, g+n */ ],
      contexts: { /* NAV, page-specific contexts */ }
    }
  }
}
```

**Page-Specific Implementations:**

```javascript
// Homepage
class HomePageNav extends BasePageNav {
  getDefaultContext() { return 'DIGEST' }

  handleContextKey(key) {
    if (this.context === 'NAV') {
      this.handleNavContext(key)
    } else if (this.context === 'THEME') {
      this.handleThemeContext(key)
    } else if (this.context === 'DIGEST') {
      this.handleDigestContext(key)
    }
  }
}

// About Page
class AboutPageNav extends BasePageNav {
  getDefaultContext() { return 'CONTENT' }

  handleContextKey(key) {
    if (this.context === 'NAV') {
      this.handleNavContext(key)
    } else if (this.context === 'CONTENT') {
      this.handleContentContext(key)
    }
  }
}

// Digest Detail Page
class DigestDetailPageNav extends BasePageNav {
  getDefaultContext() { return 'ARTICLE' }

  handleContextKey(key) {
    if (this.context === 'NAV') {
      this.handleNavContext(key)
    } else if (this.context === 'SECTION') {
      this.handleSectionContext(key)
    } else if (this.context === 'ARTICLE') {
      this.handleArticleContext(key)
    }
  }
}
```

### Shared Components

**Context Indicator Component:**
- Position: Bottom-right (all pages)
- Shows current context name
- Context-specific color coding
- Hides on touch devices

**Help Modal Component:**
- Press `?` on any page
- Shows global shortcuts (top section)
- Shows page-specific shortcuts (main section)
- Highlights shortcuts for current context
- Organized by context

**Example Help Modal (Digest Detail Page):**
```
┌─────────────────────────────────────────┐
│  Keyboard Shortcuts - Digest Detail     │
├─────────────────────────────────────────┤
│  Global (any page)                      │
│  ?          Show this help              │
│  Esc        Reset to default context    │
│  g + h      Go home                     │
│  g + a      Go to About                 │
│  g + n      Go to header navigation     │
├─────────────────────────────────────────┤
│  NAV Context                            │
│  h / l      Navigate links              │
│  Enter      Follow link                 │
├─────────────────────────────────────────┤
│  SECTION Context                        │
│  j / k      Next/previous section       │
│  Enter / l  Enter article list          │
│  g g        Jump to top                 │
│  G          Jump to bottom              │
├─────────────────────────────────────────┤
│  ⭐ ARTICLE Context (current)           │
│  j / k      Next/previous article       │
│  Enter / o  Open article                │
│  h / Esc    Exit to sections            │
│  y          Copy article link           │
│  e          Expand/collapse             │
└─────────────────────────────────────────┘
```

### Future Pages

When adding new pages, follow this pattern:

1. **Define default context** - What should be focused on page load?
2. **Define page contexts** - What sections can user navigate?
3. **Extend BasePageNav** - Implement `handleContextKey()`
4. **Add help content** - Define `getHelpContent()`
5. **Add to global shortcuts** - Update `g + ?` shortcuts if needed

**Example - Future Settings Page:**
```javascript
class SettingsPageNav extends BasePageNav {
  getDefaultContext() { return 'FORM' }

  contexts: {
    NAV: 'Header navigation',
    SIDEBAR: 'Settings sections',
    FORM: 'Form fields in current section'
  }

  handleContextKey(key) {
    if (this.context === 'NAV') this.handleNavContext(key)
    if (this.context === 'SIDEBAR') this.handleSidebarContext(key)
    if (this.context === 'FORM') this.handleFormContext(key)
  }
}
```

### Implementation Files

**Shared/Base:**
- `web/static/js/keyboard-nav-base.js` - BasePageNav class
- `web/static/js/keyboard-nav-components.js` - Context indicator, help modal
- `web/static/css/keyboard-nav.css` - Shared styles

**Page-Specific:**
- `web/static/js/keyboard-nav-home.js` - HomePageNav class
- `web/static/js/keyboard-nav-about.js` - AboutPageNav class
- `web/static/js/keyboard-nav-digest.js` - DigestDetailPageNav class

**Usage in Templates:**
```html
<!-- home.html -->
<script src="/static/js/keyboard-nav-base.js"></script>
<script src="/static/js/keyboard-nav-home.js"></script>
<script>
  const nav = new HomePageNav()
</script>

<!-- about.html -->
<script src="/static/js/keyboard-nav-base.js"></script>
<script src="/static/js/keyboard-nav-about.js"></script>
<script>
  const nav = new AboutPageNav()
</script>
```

## References

- [Vim modes documentation](https://vim.fandom.com/wiki/Vim_modes)
- [WAI-ARIA keyboard patterns](https://www.w3.org/WAI/ARIA/apg/practices/keyboard-interface/)
- [Nord theme colors](https://www.nordtheme.com/docs/colors-and-palettes)
