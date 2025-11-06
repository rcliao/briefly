/**
 * Keyboard Navigation Components
 *
 * Shared UI components for keyboard navigation:
 * - Context indicator (bottom-right badge)
 * - Help modal (enhanced version)
 * - Utility functions
 */

// ============================================================
// Utility Functions
// ============================================================

/**
 * Scroll element into view with smooth animation
 */
function smoothScrollTo(element, offset = 80) {
  if (!element) return

  const elementPosition = element.getBoundingClientRect().top
  const offsetPosition = elementPosition + window.pageYOffset - offset

  window.scrollTo({
    top: offsetPosition,
    behavior: 'smooth'
  })
}

/**
 * Copy text to clipboard
 */
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch (err) {
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'absolute'
    textarea.style.left = '-9999px'
    document.body.appendChild(textarea)
    textarea.select()
    const success = document.execCommand('copy')
    document.body.removeChild(textarea)
    return success
  }
}

/**
 * Show temporary notification
 */
function showNotification(message, duration = 2000) {
  // Remove existing notification
  const existing = document.getElementById('keyboard-notification')
  if (existing) {
    existing.remove()
  }

  // Create notification
  const notification = document.createElement('div')
  notification.id = 'keyboard-notification'
  notification.className = 'keyboard-notification'
  notification.textContent = message

  document.body.appendChild(notification)

  // Trigger animation
  setTimeout(() => {
    notification.classList.add('show')
  }, 10)

  // Remove after duration
  setTimeout(() => {
    notification.classList.remove('show')
    setTimeout(() => {
      notification.remove()
    }, 300)
  }, duration)
}

// ============================================================
// Swipe Gesture Detection (Mobile)
// ============================================================

class SwipeGestureHandler {
  constructor(onSwipeUp, onSwipeDown, threshold = 50) {
    this.onSwipeUp = onSwipeUp
    this.onSwipeDown = onSwipeDown
    this.threshold = threshold
    this.touchStartY = 0
    this.touchStartX = 0

    this.init()
  }

  init() {
    document.addEventListener('touchstart', (e) => {
      this.touchStartY = e.touches[0].clientY
      this.touchStartX = e.touches[0].clientX
    }, { passive: true })

    document.addEventListener('touchend', (e) => {
      const touchEndY = e.changedTouches[0].clientY
      const touchEndX = e.changedTouches[0].clientX

      const deltaY = this.touchStartY - touchEndY
      const deltaX = Math.abs(this.touchStartX - touchEndX)

      // Only trigger if vertical swipe (not horizontal)
      if (deltaX < 50) {
        // Swipe up = next
        if (deltaY > this.threshold && this.onSwipeUp) {
          this.onSwipeUp()
        }
        // Swipe down = previous
        else if (deltaY < -this.threshold && this.onSwipeDown) {
          this.onSwipeDown()
        }
      }
    }, { passive: true })
  }
}

// ============================================================
// Long Press Detection (Mobile Context Menu)
// ============================================================

class LongPressHandler {
  constructor(element, onLongPress, delay = 500) {
    this.element = element
    this.onLongPress = onLongPress
    this.delay = delay
    this.timer = null

    this.init()
  }

  init() {
    this.element.addEventListener('touchstart', (e) => {
      this.timer = setTimeout(() => {
        if (this.onLongPress) {
          this.onLongPress(e)
        }
      }, this.delay)
    })

    this.element.addEventListener('touchend', () => {
      clearTimeout(this.timer)
    })

    this.element.addEventListener('touchmove', () => {
      clearTimeout(this.timer)
    })
  }
}

// ============================================================
// Context Menu (For Long Press)
// ============================================================

class ContextMenu {
  constructor() {
    this.menu = null
  }

  show(x, y, items) {
    // Remove existing menu
    this.hide()

    // Create menu
    this.menu = document.createElement('div')
    this.menu.className = 'context-menu'
    this.menu.style.left = `${x}px`
    this.menu.style.top = `${y}px`

    // Add items
    items.forEach(item => {
      const menuItem = document.createElement('button')
      menuItem.className = 'context-menu-item'
      menuItem.textContent = item.label
      menuItem.onclick = () => {
        item.action()
        this.hide()
      }
      this.menu.appendChild(menuItem)
    })

    document.body.appendChild(this.menu)

    // Close on click outside
    setTimeout(() => {
      document.addEventListener('click', this.handleClickOutside)
    }, 0)
  }

  hide() {
    if (this.menu) {
      this.menu.remove()
      this.menu = null
      document.removeEventListener('click', this.handleClickOutside)
    }
  }

  handleClickOutside = (e) => {
    if (this.menu && !this.menu.contains(e.target)) {
      this.hide()
    }
  }
}

// Export for use in other files
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    smoothScrollTo,
    copyToClipboard,
    showNotification,
    SwipeGestureHandler,
    LongPressHandler,
    ContextMenu
  }
}
