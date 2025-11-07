/**
 * DigestDetailPageNav - Keyboard navigation for digest detail page
 *
 * Contexts:
 * - NAV: Header navigation (inherited from BasePageNav)
 * - SIDEBAR: Recent digests navigation (sidebar)
 * - SECTION: Section navigation (Summary, Key Moments, Articles) - default
 * - ARTICLE: Article list navigation (when in Articles section)
 */

class DigestDetailPageNav extends BasePageNav {
  constructor() {
    super('digest-detail')

    // SIDEBAR context state
    this.recentDigestIndex = 0
    this.recentDigests = []

    // SECTION context state
    this.sectionIndex = 0
    this.sections = []

    // ARTICLE context state
    this.articleIndex = 0
    this.articles = []

    // Initialize after parent setup
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.initDigestPage())
    } else {
      this.initDigestPage()
    }
  }

  // Default context for digest detail page is SECTION
  getDefaultContext() {
    return 'SECTION'
  }

  // Initialize digest page-specific features
  initDigestPage() {
    // Cache DOM references
    this.recentDigests = Array.from(document.querySelectorAll('.digest-nav-item'))
    this.sections = Array.from(document.querySelectorAll('.navigable-section'))
    this.articles = Array.from(document.querySelectorAll('.article-item'))

    // Initialize contexts
    this.initSidebarContext()
    this.initSectionContext()
    this.initArticleContext()

    // Set initial state - highlight first section
    if (this.sections.length > 0) {
      this.highlightSection(0)
    }

    // Initialize swipe gestures for mobile
    if (typeof SwipeGestureHandler !== 'undefined') {
      new SwipeGestureHandler(
        () => {
          if (this.context === 'SIDEBAR') {
            this.nextRecentDigest()
          } else if (this.context === 'SECTION') {
            this.nextSection()
          } else if (this.context === 'ARTICLE') {
            this.nextArticle()
          }
        },
        () => {
          if (this.context === 'SIDEBAR') {
            this.prevRecentDigest()
          } else if (this.context === 'SECTION') {
            this.prevSection()
          } else if (this.context === 'ARTICLE') {
            this.prevArticle()
          }
        }
      )
    }
  }

  // ============================================================
  // SIDEBAR Context (Recent Digests)
  // ============================================================

  initSidebarContext() {
    // Add click/touch handlers to recent digest items
    this.recentDigests.forEach((item, index) => {
      const updateState = () => {
        this.recentDigestIndex = index
        this.setContext('SIDEBAR')
        this.highlightRecentDigest(index)
      }

      item.addEventListener('click', (e) => {
        // Don't hijack clicks on links
        if (e.target.closest('a')) return
        updateState()
      })

      // Use smart tap handler for mobile - only triggers on taps, not scrolls
      if (typeof addTapHandler !== 'undefined') {
        addTapHandler(item, (e) => {
          if (e.target.closest('a')) return
          e.preventDefault()
          updateState()
        })
      }
    })
  }

  handleSidebarContext(e) {
    const key = e.key

    if (key === 'j' || key === 'ArrowDown') {
      e.preventDefault()
      this.nextRecentDigest()
    } else if (key === 'k' || key === 'ArrowUp') {
      e.preventDefault()
      this.prevRecentDigest()
    } else if (key === 'Enter' || key === 'o') {
      e.preventDefault()
      this.openRecentDigest()
    } else if (key === 'Escape' || key === 'l') {
      e.preventDefault()
      // Exit to SECTION context
      this.setContext('SECTION')
      this.highlightSection(this.sectionIndex)
    } else if (key === 'g') {
      e.preventDefault()
      this.handleSidebarGGShortcut()
    } else if (key === 'G') {
      e.preventDefault()
      // Go to last recent digest
      this.recentDigestIndex = this.recentDigests.length - 1
      this.highlightRecentDigest(this.recentDigestIndex)
    }
  }

  handleSidebarGGShortcut() {
    // Wait for second 'g' key
    this.waitForSecondKey((second) => {
      if (second === 'g') {
        // Go to first recent digest
        this.recentDigestIndex = 0
        this.highlightRecentDigest(0)
      }
    })
  }

  nextRecentDigest() {
    if (this.recentDigests.length === 0) return
    this.recentDigestIndex = (this.recentDigestIndex + 1) % this.recentDigests.length
    this.highlightRecentDigest(this.recentDigestIndex)
  }

  prevRecentDigest() {
    if (this.recentDigests.length === 0) return
    this.recentDigestIndex = (this.recentDigestIndex - 1 + this.recentDigests.length) % this.recentDigests.length
    this.highlightRecentDigest(this.recentDigestIndex)
  }

  highlightRecentDigest(index) {
    // Remove previous highlights
    this.recentDigests.forEach(item => {
      item.classList.remove('highlighted')
      item.removeAttribute('aria-current')
    })

    // Add highlight to current
    if (this.recentDigests[index]) {
      this.recentDigests[index].classList.add('highlighted')
      this.recentDigests[index].setAttribute('aria-current', 'true')

      // Scroll into view
      if (typeof smoothScrollTo !== 'undefined') {
        smoothScrollTo(this.recentDigests[index])
      } else {
        this.recentDigests[index].scrollIntoView({ behavior: 'smooth', block: 'nearest' })
      }

      // Announce to screen reader
      const title = this.recentDigests[index].querySelector('.digest-nav-title')?.textContent || `Digest ${index + 1}`
      this.announceToScreenReader(`${title} - Recent digest ${index + 1} of ${this.recentDigests.length}`)
    }
  }

  openRecentDigest() {
    const item = this.recentDigests[this.recentDigestIndex]
    if (!item) return

    // Find the link and navigate
    const link = item.querySelector('a')
    if (link) {
      window.location.href = link.href

      // Track with analytics
      if (window.posthog) {
        posthog.capture('digest_opened_keyboard', {
          from: 'sidebar',
          href: link.href
        })
      }
    }
  }

  // ============================================================
  // SECTION Context
  // ============================================================

  initSectionContext() {
    // Add click/touch handlers to sections
    this.sections.forEach((section, index) => {
      const updateState = () => {
        this.sectionIndex = index
        this.setContext('SECTION')
        this.highlightSection(index)
      }

      section.addEventListener('click', (e) => {
        // Don't hijack clicks on links or article items
        if (e.target.closest('a') || e.target.closest('.article-item')) return
        updateState()
      })

      // Use smart tap handler for mobile - only triggers on taps, not scrolls
      if (typeof addTapHandler !== 'undefined') {
        addTapHandler(section, (e) => {
          if (e.target.closest('a') || e.target.closest('.article-item')) return
          e.preventDefault()
          updateState()
        })
      }
    })
  }

  handleSectionContext(e) {
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
    } else if (key === 'Enter' || key === 'o') {
      e.preventDefault()
      // If on "articles" section, enter ARTICLE context
      const currentSection = this.sections[this.sectionIndex]
      if (currentSection && currentSection.dataset.section === 'articles') {
        this.enterArticleContext()
      }
    } else if (key === 'g') {
      e.preventDefault()
      this.handleGGShortcut()
    } else if (key === 'G') {
      e.preventDefault()
      // Go to last section
      this.sectionIndex = this.sections.length - 1
      this.highlightSection(this.sectionIndex)
    } else if (key === 'h') {
      e.preventDefault()
      window.location.href = '/'
    } else if (key === 'r') {
      e.preventDefault()
      // Enter SIDEBAR context (recent digests)
      if (this.recentDigests.length > 0) {
        this.setContext('SIDEBAR')
        this.highlightRecentDigest(this.recentDigestIndex)
      }
    } else if (key === 'y') {
      e.preventDefault()
      this.copySectionContent()
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

  async copySectionContent() {
    const section = this.sections[this.sectionIndex]
    if (!section) return

    // Get section title and text
    const title = section.querySelector('h2')?.textContent || ''
    const content = section.textContent.trim()

    const text = `${title}\n\n${content}`

    if (text && typeof copyToClipboard !== 'undefined') {
      const success = await copyToClipboard(text.trim())
      if (success && typeof showNotification !== 'undefined') {
        showNotification('Section copied to clipboard!')
      }
    }
  }

  // ============================================================
  // ARTICLE Context
  // ============================================================

  initArticleContext() {
    // Add click/touch handlers to articles
    this.articles.forEach((article, index) => {
      const updateState = () => {
        this.articleIndex = index
        this.setContext('ARTICLE')
        this.highlightArticle(index)
      }

      article.addEventListener('click', (e) => {
        // Don't hijack clicks on links
        if (e.target.closest('a')) return
        updateState()
      })

      // Use smart tap handler for mobile - only triggers on taps, not scrolls
      if (typeof addTapHandler !== 'undefined') {
        addTapHandler(article, (e) => {
          if (e.target.closest('a')) return
          e.preventDefault()
          updateState()
        })
      }
    })
  }

  enterArticleContext() {
    if (this.articles.length === 0) return

    // Start at first article
    this.articleIndex = 0
    this.setContext('ARTICLE')
    this.highlightArticle(0)
  }

  handleArticleContext(e) {
    const key = e.key

    if (key === 'j' || key === 'ArrowDown') {
      e.preventDefault()
      this.nextArticle()
    } else if (key === 'k' || key === 'ArrowUp') {
      e.preventDefault()
      // If at first article, exit to SECTION context
      if (this.articleIndex === 0) {
        this.exitArticleContext()
      } else {
        this.prevArticle()
      }
    } else if (key === 'Enter' || key === 'o') {
      e.preventDefault()
      this.openArticle()
    } else if (key === 'Escape') {
      e.preventDefault()
      this.exitArticleContext()
    } else if (key === 'g') {
      e.preventDefault()
      this.handleArticleGGShortcut()
    } else if (key === 'G') {
      e.preventDefault()
      // Go to last article
      this.articleIndex = this.articles.length - 1
      this.highlightArticle(this.articleIndex)
    } else if (key === 'y') {
      e.preventDefault()
      this.copyArticleInfo()
    }
  }

  handleArticleGGShortcut() {
    // Wait for second 'g' key
    this.waitForSecondKey((second) => {
      if (second === 'g') {
        // Go to first article
        this.articleIndex = 0
        this.highlightArticle(0)
      }
    })
  }

  exitArticleContext() {
    // Go back to SECTION context, focused on articles section
    this.setContext('SECTION')

    // Find articles section
    const articlesIndex = this.sections.findIndex(s => s.dataset.section === 'articles')
    if (articlesIndex !== -1) {
      this.sectionIndex = articlesIndex
      this.highlightSection(articlesIndex)
    }
  }

  nextArticle() {
    if (this.articles.length === 0) return
    this.articleIndex = (this.articleIndex + 1) % this.articles.length
    this.highlightArticle(this.articleIndex)
  }

  prevArticle() {
    if (this.articles.length === 0) return
    this.articleIndex = (this.articleIndex - 1 + this.articles.length) % this.articles.length
    this.highlightArticle(this.articleIndex)
  }

  highlightArticle(index) {
    // Remove previous highlights
    this.articles.forEach(article => {
      article.classList.remove('highlighted')
      article.removeAttribute('aria-current')
    })

    // Add highlight to current
    if (this.articles[index]) {
      this.articles[index].classList.add('highlighted')
      this.articles[index].setAttribute('aria-current', 'true')

      // Scroll into view
      if (typeof smoothScrollTo !== 'undefined') {
        smoothScrollTo(this.articles[index])
      } else {
        this.articles[index].scrollIntoView({ behavior: 'smooth', block: 'center' })
      }

      // Announce to screen reader
      const title = this.articles[index].querySelector('.article-link')?.textContent || `Article ${index + 1}`
      this.announceToScreenReader(`${title} - Article ${index + 1} of ${this.articles.length}`)
    }
  }

  openArticle() {
    const article = this.articles[this.articleIndex]
    if (!article) return

    // Get article URL and open in new tab
    const url = article.dataset.url
    if (url) {
      window.open(url, '_blank', 'noopener,noreferrer')

      // Track with analytics
      if (window.posthog) {
        posthog.capture('article_opened_keyboard', { url })
      }
    }
  }

  async copyArticleInfo() {
    const article = this.articles[this.articleIndex]
    if (!article) return

    // Get article title and URL
    const title = article.querySelector('.article-link')?.textContent || ''
    const url = article.dataset.url || ''

    const text = `${title}\n${url}`

    if (text && typeof copyToClipboard !== 'undefined') {
      const success = await copyToClipboard(text.trim())
      if (success && typeof showNotification !== 'undefined') {
        showNotification('Article info copied to clipboard!')
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
    } else if (this.context === 'SIDEBAR') {
      this.handleSidebarContext(e)
    } else if (this.context === 'SECTION') {
      this.handleSectionContext(e)
    } else if (this.context === 'ARTICLE') {
      this.handleArticleContext(e)
    }
  }

  // ============================================================
  // Help Content
  // ============================================================

  getHelpContent() {
    const baseHelp = super.getHelpContent()
    return baseHelp + `
      <section class="shortcut-section ${this.context === 'SIDEBAR' ? 'current-context' : ''}">
        <h3>SIDEBAR Context (Recent Digests)</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>j</kbd> / <kbd>k</kbd> or <kbd>↓</kbd> / <kbd>↑</kbd></dt>
            <dd>Navigate recent digests</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Enter</kbd> / <kbd>o</kbd></dt>
            <dd>Open selected digest</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>g</kbd> <kbd>g</kbd></dt>
            <dd>Go to first digest</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>G</kbd></dt>
            <dd>Go to last digest</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Esc</kbd> / <kbd>l</kbd></dt>
            <dd>Exit to sections</dd>
          </div>
        </dl>
      </section>

      <section class="shortcut-section ${this.context === 'SECTION' ? 'current-context' : ''}">
        <h3>SECTION Context</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>j</kbd> / <kbd>k</kbd> or <kbd>↓</kbd> / <kbd>↑</kbd></dt>
            <dd>Navigate sections</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Enter</kbd> / <kbd>o</kbd></dt>
            <dd>Enter articles (when on Articles section)</dd>
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
            <dt><kbd>r</kbd></dt>
            <dd>Go to recent digests (sidebar)</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>h</kbd></dt>
            <dd>Back to home</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>y</kbd></dt>
            <dd>Copy section content</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>k</kbd> at top</dt>
            <dd>Move to navigation</dd>
          </div>
        </dl>
      </section>

      <section class="shortcut-section ${this.context === 'ARTICLE' ? 'current-context' : ''}">
        <h3>ARTICLE Context</h3>
        <dl class="shortcut-list">
          <div class="shortcut-item">
            <dt><kbd>j</kbd> / <kbd>k</kbd> or <kbd>↓</kbd> / <kbd>↑</kbd></dt>
            <dd>Navigate articles</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Enter</kbd> / <kbd>o</kbd></dt>
            <dd>Open article in new tab</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>g</kbd> <kbd>g</kbd></dt>
            <dd>Go to first article</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>G</kbd></dt>
            <dd>Go to last article</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>Esc</kbd></dt>
            <dd>Exit to sections</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>y</kbd></dt>
            <dd>Copy article title and URL</dd>
          </div>
          <div class="shortcut-item">
            <dt><kbd>k</kbd> at top</dt>
            <dd>Exit to sections</dd>
          </div>
        </dl>
      </section>
    `
  }
}

// Initialize on page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    new DigestDetailPageNav()
  })
} else {
  new DigestDetailPageNav()
}
