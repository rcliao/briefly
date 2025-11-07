/**
 * BasePageNav - Foundation for keyboard navigation across all pages
 *
 * Features:
 * - Global shortcuts (?, Esc, g+h, g+a, g+n)
 * - NAV context for header navigation
 * - Input type detection (keyboard/mouse/touch)
 * - Context indicator management
 */

class BasePageNav {
  constructor(pageType) {
    this.pageType = pageType  // 'home', 'about', 'digest-detail'
    this.context = this.getDefaultContext()

    // Navigation state
    this.navIndex = 0
    this.navLinks = []

    // Input type tracking
    this.lastInputType = 'unknown' // 'keyboard', 'mouse', 'touch'

    // For multi-key shortcuts (g+h, g+a, etc.)
    this.waitingForSecondKey = false
    this.secondKeyCallback = null
    this.secondKeyTimeout = null

    // DOM references
    this.contextIndicator = null
    this.modalKeyHandler = null

    // Initialize
    this.init()
  }

  // Override in subclasses to set default context
  getDefaultContext() {
    return 'CONTENT'
  }

  // Initialize navigation system
  init() {
    // Wait for DOM to be ready
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.setup())
    } else {
      this.setup()
    }
  }

  setup() {
    // Cache DOM references
    this.navLinks = Array.from(document.querySelectorAll('header nav a'))

    // Initialize components
    this.initGlobalKeys()
    this.initNavContext()
    this.initInputTypeDetection()
    this.initContextIndicator()

    // Set initial context
    this.setContext(this.getDefaultContext())
  }

  // ============================================================
  // Global Key Handlers
  // ============================================================

  initGlobalKeys() {
    document.addEventListener('keydown', (e) => {
      // Skip if typing in input/textarea
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        return
      }

      // Handle second key in multi-key shortcuts
      if (this.waitingForSecondKey) {
        clearTimeout(this.secondKeyTimeout)
        this.waitingForSecondKey = false
        if (this.secondKeyCallback) {
          this.secondKeyCallback(e.key)
          this.secondKeyCallback = null
        }
        return
      }

      // Help modal
      if (e.key === '?') {
        e.preventDefault()
        this.showHelpModal()
        return
      }

      // Esc always resets to default context
      if (e.key === 'Escape') {
        e.preventDefault()
        this.setContext(this.getDefaultContext())
        return
      }

      // Go-to shortcuts (g + second key)
      if (e.key === 'g') {
        e.preventDefault()
        this.handleGoToShortcut()
        return
      }

      // Route to context-specific handler
      this.handleContextKey(e)
    })
  }

  handleGoToShortcut() {
    this.waitForSecondKey((second) => {
      if (second === 'h') {
        window.location.href = '/'
      } else if (second === 'a') {
        window.location.href = '/about'
      } else if (second === 'n') {
        this.setContext('NAV')
        this.highlightNav(this.navIndex)
      }
    })
  }

  waitForSecondKey(callback) {
    this.waitingForSecondKey = true
    this.secondKeyCallback = callback

    // Timeout after 1 second
    this.secondKeyTimeout = setTimeout(() => {
      this.waitingForSecondKey = false
      this.secondKeyCallback = null
    }, 1000)
  }

  // ============================================================
  // NAV Context (Header Navigation)
  // ============================================================

  initNavContext() {
    // Add click/touch handlers to nav links
    this.navLinks.forEach((link, index) => {
      const updateState = () => {
        this.navIndex = index
        this.setContext('NAV')
        this.highlightNav(index)
      }

      link.addEventListener('click', updateState)

      // Use smart tap handler for mobile - only triggers on taps, not scrolls
      if (typeof addTapHandler !== 'undefined') {
        addTapHandler(link, (e) => {
          e.preventDefault()
          updateState()
        })
      }
    })
  }

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
      this.setContext(this.getDefaultContext())
    }
  }

  highlightNav(index) {
    // Remove previous highlights
    this.navLinks.forEach(link => {
      link.classList.remove('highlighted')
      link.removeAttribute('aria-current')
    })

    // Add highlight to current
    if (this.navLinks[index]) {
      this.navLinks[index].classList.add('highlighted')
      this.navLinks[index].setAttribute('aria-current', 'true')

      // Scroll to show the header when entering NAV context
      const header = document.querySelector('header')
      if (header) {
        // Scroll to top of page to show header
        if (typeof smoothScrollTo !== 'undefined') {
          smoothScrollTo(header, 0)
        } else {
          window.scrollTo({ top: 0, behavior: 'smooth' })
        }
      }

      this.navLinks[index].focus({ preventScroll: true })
    }
  }

  // ============================================================
  // Input Type Detection
  // ============================================================

  initInputTypeDetection() {
    // Keyboard input - show context indicator
    document.addEventListener('keydown', (e) => {
      if (this.lastInputType !== 'keyboard') {
        this.lastInputType = 'keyboard'
        document.body.classList.add('keyboard-mode')
        this.showContextIndicator()
      }
    })

    // Touch input - hide context indicator
    document.addEventListener('touchstart', () => {
      if (this.lastInputType !== 'touch') {
        this.lastInputType = 'touch'
        document.body.classList.remove('keyboard-mode')
        this.hideContextIndicator()
      }
    })

    // Mouse click - don't change indicator visibility
    document.addEventListener('click', () => {
      if (this.lastInputType !== 'keyboard' && this.lastInputType !== 'mouse') {
        this.lastInputType = 'mouse'
      }
    })
  }

  // ============================================================
  // Context Management
  // ============================================================

  setContext(newContext) {
    const previousContext = this.context
    this.context = newContext

    // Update body class to reflect current context
    if (previousContext) {
      document.body.classList.remove(`context-${previousContext.toLowerCase()}`)
    }
    document.body.classList.add(`context-${newContext.toLowerCase()}`)

    // Update context indicator
    this.updateContextIndicator()

    // Announce to screen readers
    this.announceContextChange(previousContext, newContext)
  }

  // Override in subclasses to handle page-specific contexts
  handleContextKey(e) {
    if (this.context === 'NAV') {
      this.handleNavContext(e)
    }
    // Subclasses handle other contexts
  }

  // ============================================================
  // Context Indicator
  // ============================================================

  initContextIndicator() {
    // Create indicator if it doesn't exist
    if (!document.getElementById('context-indicator')) {
      const indicator = document.createElement('div')
      indicator.id = 'context-indicator'
      indicator.className = 'context-indicator'
      document.body.appendChild(indicator)
    }

    this.contextIndicator = document.getElementById('context-indicator')
  }

  updateContextIndicator() {
    if (!this.contextIndicator) return

    // Remove all context classes
    this.contextIndicator.className = 'context-indicator'

    // Add current context class
    this.contextIndicator.classList.add(`context-${this.context.toLowerCase()}`)

    // Update text with icon
    const icons = {
      'NAV': '🔗',
      'THEME': '🎨',
      'DIGEST': '📰',
      'CONTENT': '📄',
      'SECTION': '📑',
      'SIDEBAR': '📋',
      'ARTICLE': '📖'
    }

    const icon = icons[this.context] || '📄'
    const contextName = this.context === 'THEME' ? 'THEME (multi-select)' : this.context

    this.contextIndicator.textContent = `${icon} ${contextName}`
  }

  showContextIndicator() {
    if (this.contextIndicator) {
      this.contextIndicator.style.opacity = '1'
      this.contextIndicator.style.pointerEvents = 'auto'
    }
  }

  hideContextIndicator() {
    if (this.contextIndicator) {
      this.contextIndicator.style.opacity = '0'
      this.contextIndicator.style.pointerEvents = 'none'
    }
  }

  // ============================================================
  // Accessibility
  // ============================================================

  announceContextChange(previousContext, newContext) {
    const announcements = {
      'NAV': 'Navigation context: Use h/l to navigate links, Enter to follow',
      'THEME': 'Theme context: Use h/l to navigate, Space to select, Enter to apply filters',
      'DIGEST': 'Digest context: Use j/k to navigate digests, Enter to open',
      'CONTENT': 'Content context: Use j/k to navigate sections',
      'SECTION': 'Section context: Use j/k to navigate sections, Enter to view articles',
      'SIDEBAR': 'Sidebar context: Use j/k to navigate recent digests, Enter to open',
      'ARTICLE': 'Article context: Use j/k to navigate articles, Enter to open'
    }

    const announcement = announcements[newContext] || `${newContext} context`
    this.announceToScreenReader(announcement)
  }

  announceToScreenReader(message) {
    // Create or get announcement element
    let announcer = document.getElementById('sr-announcer')
    if (!announcer) {
      announcer = document.createElement('div')
      announcer.id = 'sr-announcer'
      announcer.setAttribute('role', 'status')
      announcer.setAttribute('aria-live', 'polite')
      announcer.setAttribute('aria-atomic', 'true')
      announcer.style.position = 'absolute'
      announcer.style.left = '-10000px'
      announcer.style.width = '1px'
      announcer.style.height = '1px'
      announcer.style.overflow = 'hidden'
      document.body.appendChild(announcer)
    }

    // Clear and set new message
    announcer.textContent = ''
    setTimeout(() => {
      announcer.textContent = message
    }, 100)
  }

  // ============================================================
  // Help Modal
  // ============================================================

  showHelpModal() {
    // Check if modal already exists
    let modal = document.getElementById('keyboard-help-modal')
    if (modal) {
      // Update content with current context highlighting
      const modalBody = modal.querySelector('.modal-body')
      if (modalBody) {
        modalBody.innerHTML = this.getHelpContent()
      }
      modal.style.display = 'flex'
      this.addModalKeyboardHandlers()
      return
    }

    // Get help content from subclass
    const helpContent = this.getHelpContent()

    // Create modal
    modal = document.createElement('div')
    modal.id = 'keyboard-help-modal'
    modal.className = 'modal'
    modal.style.display = 'flex'

    modal.innerHTML = `
      <div class="modal-overlay"></div>
      <div class="modal-content">
        <div class="modal-header">
          <h2>Keyboard Shortcuts - ${this.pageType}</h2>
          <button class="modal-close" aria-label="Close modal">×</button>
        </div>
        <div class="modal-body">
          ${helpContent}
        </div>
      </div>
    `

    document.body.appendChild(modal)

    // Add event listeners
    const closeBtn = modal.querySelector('.modal-close')
    const overlay = modal.querySelector('.modal-overlay')

    const closeModal = () => this.closeHelpModal()

    closeBtn.addEventListener('click', closeModal)
    overlay.addEventListener('click', closeModal)

    this.addModalKeyboardHandlers()
  }

  closeHelpModal() {
    const modal = document.getElementById('keyboard-help-modal')
    if (modal) {
      modal.style.display = 'none'
      this.removeModalKeyboardHandlers()
    }
  }

  addModalKeyboardHandlers() {
    // Add escape key handler
    this.modalKeyHandler = (e) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        this.closeHelpModal()
      }
    }
    document.addEventListener('keydown', this.modalKeyHandler)
  }

  removeModalKeyboardHandlers() {
    if (this.modalKeyHandler) {
      document.removeEventListener('keydown', this.modalKeyHandler)
      this.modalKeyHandler = null
    }
  }

  // Override in subclasses to provide page-specific help
  getHelpContent() {
    return `
      <section class="shortcut-section">
        <h3>Global (any page)</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>?</kbd></dt>
            <dd>Show this help</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Esc</kbd></dt>
            <dd>Reset to default context</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>g</kbd> + <kbd>h</kbd></dt>
            <dd>Go home</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>g</kbd> + <kbd>a</kbd></dt>
            <dd>Go to About</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>g</kbd> + <kbd>n</kbd></dt>
            <dd>Go to header navigation</dd>
          </div>
        </dl>
      </section>

      <section class="shortcut-section ${this.context === 'NAV' ? 'current-context' : ''}">
        <h3>NAV Context</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>h</kbd> / <kbd>l</kbd> or <kbd>←</kbd> / <kbd>→</kbd></dt>
            <dd>Navigate links</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Enter</kbd></dt>
            <dd>Follow link</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>j</kbd> or <kbd>↓</kbd></dt>
            <dd>Move to page content</dd>
          </div>
        </dl>
      </section>
    `
  }
}
