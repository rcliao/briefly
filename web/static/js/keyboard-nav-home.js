/**
 * HomePageNav - Keyboard navigation for the homepage
 *
 * Contexts:
 * - NAV: Header navigation (inherited from BasePageNav)
 * - THEME: Theme filter tabs (multi-select)
 * - DIGEST: Digest card list (default)
 */

class HomePageNav extends BasePageNav {
  constructor() {
    super('home')

    // DIGEST context state
    this.digestIndex = 0
    this.digestCards = []

    // THEME context state
    this.themeIndex = 0
    this.themeTabs = []
    this.selectedThemes = ['all']  // Currently active filters
    this.pendingThemeSelection = [] // Temporary during THEME context

    // Initialize after parent setup
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.initHomePage())
    } else {
      this.initHomePage()
    }
  }

  // Default context for homepage is DIGEST
  getDefaultContext() {
    return 'DIGEST'
  }

  // Initialize homepage-specific features
  initHomePage() {
    // Cache DOM references
    this.digestCards = Array.from(document.querySelectorAll('.digest-item'))
    this.themeTabs = Array.from(document.querySelectorAll('.theme-tab'))

    // Initialize contexts
    this.initDigestContext()
    this.initThemeContext()

    // Set initial state - highlight first digest
    if (this.digestCards.length > 0) {
      this.highlightDigest(0)
    }

    // Initialize swipe gestures for mobile
    if (typeof SwipeGestureHandler !== 'undefined') {
      new SwipeGestureHandler(
        () => this.nextDigest(),  // Swipe up = next
        () => this.prevDigest()   // Swipe down = previous
      )
    }
  }

  // ============================================================
  // DIGEST Context
  // ============================================================

  initDigestContext() {
    // Add click/touch handlers to digest cards
    this.digestCards.forEach((card, index) => {
      const updateState = () => {
        this.digestIndex = index
        this.setContext('DIGEST')
        this.highlightDigest(index)
      }

      card.addEventListener('click', (e) => {
        // Don't hijack clicks on links
        if (e.target.closest('a')) return
        updateState()
      })

      card.addEventListener('touchend', (e) => {
        if (e.target.closest('a')) return
        e.preventDefault()
        updateState()
      })
    })
  }

  handleDigestContext(e) {
    const key = e.key

    if (key === 'j' || key === 'ArrowDown') {
      e.preventDefault()
      this.nextDigest()
    } else if (key === 'k' || key === 'ArrowUp') {
      e.preventDefault()
      // If at first digest, move to THEME context
      if (this.digestIndex === 0) {
        this.setContext('THEME')
        this.highlightTheme(this.themeIndex)
      } else {
        this.prevDigest()
      }
    } else if (key === 'Enter' || key === 'o') {
      e.preventDefault()
      this.openDigest()
    } else if (key === 'h') {
      e.preventDefault()
      this.setContext('THEME')
      this.highlightTheme(this.themeIndex)
    } else if (key === 'y') {
      e.preventDefault()
      this.copyDigestSummary()
    }
  }

  nextDigest() {
    if (this.digestCards.length === 0) return
    this.digestIndex = (this.digestIndex + 1) % this.digestCards.length
    this.highlightDigest(this.digestIndex)
  }

  prevDigest() {
    if (this.digestCards.length === 0) return
    this.digestIndex = (this.digestIndex - 1 + this.digestCards.length) % this.digestCards.length
    this.highlightDigest(this.digestIndex)
  }

  highlightDigest(index) {
    // Remove previous highlights
    this.digestCards.forEach(card => {
      card.classList.remove('highlighted')
      card.removeAttribute('aria-current')
    })

    // Add highlight to current
    if (this.digestCards[index]) {
      this.digestCards[index].classList.add('highlighted')
      this.digestCards[index].setAttribute('aria-current', 'true')

      // Scroll into view
      if (typeof smoothScrollTo !== 'undefined') {
        smoothScrollTo(this.digestCards[index])
      } else {
        this.digestCards[index].scrollIntoView({ behavior: 'smooth', block: 'center' })
      }

      // Announce to screen reader
      this.announceToScreenReader(`Digest ${index + 1} of ${this.digestCards.length}`)
    }
  }

  openDigest() {
    const card = this.digestCards[this.digestIndex]
    if (!card) return

    // Get digest link
    const link = card.querySelector('a[href^="/digests/"]')
    if (link) {
      link.click()
    }
  }

  async copyDigestSummary() {
    const card = this.digestCards[this.digestIndex]
    if (!card) return

    // Get summary text
    const summary = card.querySelector('.summary-preview')?.textContent ||
                   card.querySelector('.digest-summary')?.textContent ||
                   ''

    if (summary && typeof copyToClipboard !== 'undefined') {
      const success = await copyToClipboard(summary.trim())
      if (success && typeof showNotification !== 'undefined') {
        showNotification('Summary copied to clipboard!')
      }
    }
  }

  // ============================================================
  // THEME Context (Multi-select)
  // ============================================================

  initThemeContext() {
    // Get currently active theme from URL or DOM
    const urlParams = new URLSearchParams(window.location.search)
    const activeTheme = urlParams.get('theme') || 'all'
    this.selectedThemes = [activeTheme]

    // Find index of active theme
    this.themeTabs.forEach((tab, index) => {
      const themeId = tab.dataset.themeId || tab.textContent.toLowerCase()
      if (themeId === activeTheme) {
        this.themeIndex = index
      }

      // Mark selected themes
      if (this.selectedThemes.includes(themeId)) {
        tab.classList.add('selected')
      }
    })

    // Add click/touch handlers to theme tabs
    this.themeTabs.forEach((tab, index) => {
      const updateState = () => {
        this.themeIndex = index
        this.setContext('THEME')
        this.highlightTheme(index)

        // Auto-apply on click (instant filter)
        const themeId = tab.dataset.themeId || tab.textContent.toLowerCase()
        this.selectedThemes = [themeId]
        this.applyThemeFilters()
      }

      tab.addEventListener('click', updateState)
      tab.addEventListener('touchend', (e) => {
        e.preventDefault()
        updateState()
      })
    })
  }

  handleThemeContext(e) {
    const key = e.key

    if (key === 'h' || key === 'ArrowLeft') {
      e.preventDefault()
      this.prevTheme()
    } else if (key === 'l' || key === 'ArrowRight') {
      e.preventDefault()
      this.nextTheme()
    } else if (key === ' ') {
      e.preventDefault()
      this.toggleThemeSelection()
    } else if (key === 'Enter') {
      e.preventDefault()
      this.applyThemeFilters()
      this.setContext('DIGEST')
      this.highlightDigest(0)
    } else if (key === 'j' || key === 'ArrowDown') {
      e.preventDefault()
      // Exit to DIGEST without applying changes
      this.pendingThemeSelection = []
      this.setContext('DIGEST')
      this.highlightDigest(0)
    } else if (key === 'k' || key === 'ArrowUp') {
      e.preventDefault()
      // Move to NAV context
      this.setContext('NAV')
      this.highlightNav(this.navIndex)
    }
  }

  nextTheme() {
    if (this.themeTabs.length === 0) return
    this.themeIndex = (this.themeIndex + 1) % this.themeTabs.length
    this.highlightTheme(this.themeIndex)
  }

  prevTheme() {
    if (this.themeTabs.length === 0) return
    this.themeIndex = (this.themeIndex - 1 + this.themeTabs.length) % this.themeTabs.length
    this.highlightTheme(this.themeIndex)
  }

  highlightTheme(index) {
    // Remove previous cursor
    this.themeTabs.forEach(tab => {
      tab.classList.remove('cursor')
      tab.removeAttribute('aria-current')
    })

    // Add cursor to current
    if (this.themeTabs[index]) {
      this.themeTabs[index].classList.add('cursor')
      this.themeTabs[index].setAttribute('aria-current', 'location')
      this.themeTabs[index].focus({ preventScroll: true })

      // Announce to screen reader
      const themeText = this.themeTabs[index].textContent.trim()
      this.announceToScreenReader(`Theme: ${themeText}`)
    }
  }

  toggleThemeSelection() {
    const tab = this.themeTabs[this.themeIndex]
    if (!tab) return

    const themeId = tab.dataset.themeId || tab.textContent.toLowerCase()

    // Initialize pending selection from current if empty
    if (this.pendingThemeSelection.length === 0) {
      this.pendingThemeSelection = [...this.selectedThemes]
    }

    // Toggle selection
    const index = this.pendingThemeSelection.indexOf(themeId)
    if (index === -1) {
      // Add to selection
      this.pendingThemeSelection.push(themeId)
      tab.classList.add('selected')

      // If selecting any specific theme, remove "all"
      if (themeId !== 'all') {
        const allIndex = this.pendingThemeSelection.indexOf('all')
        if (allIndex !== -1) {
          this.pendingThemeSelection.splice(allIndex, 1)
          this.themeTabs.forEach(t => {
            if ((t.dataset.themeId || t.textContent.toLowerCase()) === 'all') {
              t.classList.remove('selected')
            }
          })
        }
      }
    } else {
      // Remove from selection
      this.pendingThemeSelection.splice(index, 1)
      tab.classList.remove('selected')

      // If no themes selected, select "all"
      if (this.pendingThemeSelection.length === 0) {
        this.pendingThemeSelection = ['all']
        this.themeTabs.forEach(t => {
          if ((t.dataset.themeId || t.textContent.toLowerCase()) === 'all') {
            t.classList.add('selected')
          }
        })
      }
    }

    // Announce change
    const action = index === -1 ? 'selected' : 'deselected'
    const themeText = tab.textContent.trim()
    if (typeof showNotification !== 'undefined') {
      showNotification(`${themeText} ${action}`)
    }
  }

  applyThemeFilters() {
    // Apply pending selection if any
    if (this.pendingThemeSelection.length > 0) {
      this.selectedThemes = [...this.pendingThemeSelection]
    } else {
      // If no pending selection, apply the currently highlighted theme
      const currentTab = this.themeTabs[this.themeIndex]
      if (currentTab) {
        const themeId = currentTab.dataset.themeId || currentTab.textContent.toLowerCase()
        this.selectedThemes = [themeId]
      }
    }

    // Clear pending
    this.pendingThemeSelection = []

    // Build URL with theme filter
    const url = new URL(window.location)
    if (this.selectedThemes.length === 1 && this.selectedThemes[0] === 'all') {
      url.searchParams.delete('theme')
    } else {
      url.searchParams.set('theme', this.selectedThemes.join(','))
    }

    // Navigate to filtered URL (HTMX will handle the update)
    window.location.href = url.toString()
  }

  // ============================================================
  // NAV Context Override
  // ============================================================

  handleNavContext(e) {
    const key = e.key

    if (key === 'h' || key === 'ArrowLeft') {
      e.preventDefault()
      this.navIndex = (this.navIndex - 1 + this.navLinks.length) % this.navLinks.length
      this.highlightNav(this.navIndex)
    } else if (key === 'l' || key === 'ArrowRight') {
      e.preventDefault()
      this.navIndex = (this.navIndex + 1) % this.navLinks.length
      this.highlightNav(this.navIndex)
    } else if (key === 'Enter' || key === 'o') {
      e.preventDefault()
      this.navLinks[this.navIndex].click()
    } else if (key === 'j' || key === 'ArrowDown') {
      e.preventDefault()
      // On homepage, go to THEME context (not directly to DIGEST)
      this.setContext('THEME')
      this.highlightTheme(this.themeIndex)
    }
  }

  // ============================================================
  // Context Routing
  // ============================================================

  handleContextKey(e) {
    // Route to appropriate context handler
    if (this.context === 'NAV') {
      this.handleNavContext(e)
    } else if (this.context === 'THEME') {
      this.handleThemeContext(e)
    } else if (this.context === 'DIGEST') {
      this.handleDigestContext(e)
    }
  }

  // ============================================================
  // Help Content
  // ============================================================

  getHelpContent() {
    const baseHelp = super.getHelpContent()
    return baseHelp + `
      <section class="shortcut-section ${this.context === 'THEME' ? 'current-context' : ''}">
        <h3>THEME Context</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>h</kbd> / <kbd>l</kbd> or <kbd>←</kbd> / <kbd>→</kbd></dt>
            <dd>Navigate themes</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Space</kbd></dt>
            <dd>Toggle theme selection (multi-select)</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Enter</kbd></dt>
            <dd>Apply filters and move to digests</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>j</kbd> or <kbd>↓</kbd></dt>
            <dd>Exit without applying</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>k</kbd> or <kbd>↑</kbd></dt>
            <dd>Move to navigation</dd>
          </div>
        </dl>
      </section>

      <section class="shortcut-section ${this.context === 'DIGEST' ? 'current-context' : ''}">
        <h3>DIGEST Context</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>j</kbd> / <kbd>k</kbd> or <kbd>↓</kbd> / <kbd>↑</kbd></dt>
            <dd>Navigate digests</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Enter</kbd> / <kbd>o</kbd></dt>
            <dd>Open digest</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>h</kbd></dt>
            <dd>Go to theme filters</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>y</kbd></dt>
            <dd>Copy digest summary</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>k</kbd> at top</dt>
            <dd>Move to themes</dd>
          </div>
        </dl>
      </section>
    `
  }
}

// Initialize on page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    new HomePageNav()
  })
} else {
  new HomePageNav()
}
