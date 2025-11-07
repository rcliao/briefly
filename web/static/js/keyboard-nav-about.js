/**
 * AboutPageNav - Keyboard navigation for the about page
 *
 * Contexts:
 * - NAV: Header navigation (inherited from BasePageNav)
 * - CONTENT: Section navigation (default)
 */

class AboutPageNav extends BasePageNav {
  constructor() {
    super('about')

    // CONTENT context state
    this.sectionIndex = 0
    this.sections = []

    // Initialize after parent setup
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.initAboutPage())
    } else {
      this.initAboutPage()
    }
  }

  // Default context for about page is CONTENT
  getDefaultContext() {
    return 'CONTENT'
  }

  // Initialize about page-specific features
  initAboutPage() {
    // Cache DOM references - find all sections
    this.sections = Array.from(document.querySelectorAll('.about-section'))

    // Initialize contexts
    this.initContentContext()

    // Set initial state - highlight first section
    if (this.sections.length > 0) {
      this.highlightSection(0)
    }

    // Initialize swipe gestures for mobile
    if (typeof SwipeGestureHandler !== 'undefined') {
      new SwipeGestureHandler(
        () => this.nextSection(),  // Swipe up = next
        () => this.prevSection()   // Swipe down = previous
      )
    }
  }

  // ============================================================
  // CONTENT Context
  // ============================================================

  initContentContext() {
    // Add click/touch handlers to sections
    this.sections.forEach((section, index) => {
      const updateState = () => {
        this.sectionIndex = index
        this.setContext('CONTENT')
        this.highlightSection(index)
      }

      section.addEventListener('click', (e) => {
        // Don't hijack clicks on links
        if (e.target.closest('a')) return
        updateState()
      })

      // Use smart tap handler for mobile - only triggers on taps, not scrolls
      if (typeof addTapHandler !== 'undefined') {
        addTapHandler(section, (e) => {
          if (e.target.closest('a')) return
          e.preventDefault()
          updateState()
        })
      }
    })
  }

  handleContentContext(e) {
    const key = e.key

    if (key === 'j' || key === 'ArrowDown') {
      e.preventDefault()
      this.nextSection()
    } else if (key === 'k' || key === 'ArrowUp') {
      e.preventDefault()
      // If at first section, move to NAV context
      if (this.sectionIndex === 0) {
        this.setContext('NAV')
        this.highlightNav(this.navIndex)
      } else {
        this.prevSection()
      }
    } else if (key === 'g') {
      e.preventDefault()
      this.handleGGShortcut()
    } else if (key === 'G') {
      e.preventDefault()
      // Go to last section
      this.sectionIndex = this.sections.length - 1
      this.highlightSection(this.sectionIndex)
    } else if (key === 'y') {
      e.preventDefault()
      this.copySectionText()
    }
  }

  handleGGShortcut() {
    // Wait for second 'g' key
    this.waitForSecondKey((second) => {
      if (second === 'g') {
        // Go to first section
        this.sectionIndex = 0
        this.highlightSection(0)
      }
    })
  }

  nextSection() {
    if (this.sections.length === 0) return
    this.sectionIndex = (this.sectionIndex + 1) % this.sections.length
    this.highlightSection(this.sectionIndex)
  }

  prevSection() {
    if (this.sections.length === 0) return
    this.sectionIndex = (this.sectionIndex - 1 + this.sections.length) % this.sections.length
    this.highlightSection(this.sectionIndex)
  }

  highlightSection(index) {
    // Remove previous highlights
    this.sections.forEach(section => {
      section.classList.remove('highlighted')
      section.removeAttribute('aria-current')
    })

    // Add highlight to current
    if (this.sections[index]) {
      this.sections[index].classList.add('highlighted')
      this.sections[index].setAttribute('aria-current', 'true')

      // Scroll into view
      if (typeof smoothScrollTo !== 'undefined') {
        smoothScrollTo(this.sections[index])
      } else {
        this.sections[index].scrollIntoView({ behavior: 'smooth', block: 'center' })
      }

      // Announce to screen reader
      const sectionTitle = this.sections[index].querySelector('h2')?.textContent || `Section ${index + 1}`
      this.announceToScreenReader(`${sectionTitle} - Section ${index + 1} of ${this.sections.length}`)
    }
  }

  async copySectionText() {
    const section = this.sections[this.sectionIndex]
    if (!section) return

    // Get section title and text
    const title = section.querySelector('h2')?.textContent || ''
    const paragraphs = Array.from(section.querySelectorAll('p, li'))
      .map(el => el.textContent.trim())
      .join('\n\n')

    const text = `${title}\n\n${paragraphs}`

    if (text && typeof copyToClipboard !== 'undefined') {
      const success = await copyToClipboard(text.trim())
      if (success && typeof showNotification !== 'undefined') {
        showNotification('Section copied to clipboard!')
      }
    }
  }

  // ============================================================
  // Context Routing
  // ============================================================

  handleContextKey(e) {
    // Route to appropriate context handler
    if (this.context === 'NAV') {
      this.handleNavContext(e)
    } else if (this.context === 'CONTENT') {
      this.handleContentContext(e)
    }
  }

  // ============================================================
  // Help Content
  // ============================================================

  getHelpContent() {
    const baseHelp = super.getHelpContent()
    return baseHelp + `
      <section class="shortcut-section ${this.context === 'CONTENT' ? 'current-context' : ''}">
        <h3>CONTENT Context</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>j</kbd> / <kbd>k</kbd> or <kbd>↓</kbd> / <kbd>↑</kbd></dt>
            <dd>Navigate sections</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>g</kbd> <kbd>g</kbd></dt>
            <dd>Go to first section</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>G</kbd></dt>
            <dd>Go to last section</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>y</kbd></dt>
            <dd>Copy section text</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>k</kbd> at top</dt>
            <dd>Move to navigation</dd>
          </div>
        </dl>
      </section>
    `
  }
}

// Initialize on page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    new AboutPageNav()
  })
} else {
  new AboutPageNav()
}
