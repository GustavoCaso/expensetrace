/**
 * Unified data-attribute-driven toggle system
 *
 * Usage:
 *   <button data-toggle data-toggle-target="element-id" data-toggle-text="Alt text">
 *     Show Details
 *   </button>
 *   <div id="element-id" class="collapsed">Content</div>
 *
 * Attributes:
 *   - data-toggle: Marks element as a toggle button
 *   - data-toggle-target: ID of element to toggle (optional, defaults to next sibling)
 *   - data-toggle-text: Alternate text to swap when toggling (optional)
 *
 * Behavior:
 *   - Toggles 'collapsed' class on target element
 *   - Toggles 'active' class on button for visual feedback
 *   - Swaps button text if data-toggle-text is provided
 */

export function initializeToggleSystem() {
  document.body.addEventListener('click', function (e) {
    const toggleBtn = e.target.closest('[data-toggle]');
    if (!toggleBtn) return;

    // Don't interfere with form submissions or other button behaviors
    if (toggleBtn.tagName === 'BUTTON' && toggleBtn.type !== 'button') {
      return;
    }

    e.preventDefault();
    e.stopPropagation();

    // Get target element
    const targetId = toggleBtn.getAttribute('data-toggle-target');
    let targetElement;

    if (targetId) {
      // Explicit target via data-toggle-target
      targetElement = document.getElementById(targetId);
    } else {
      // Default: next sibling element
      targetElement = toggleBtn.nextElementSibling;
    }

    if (!targetElement) {
      console.warn('Toggle target not found', {
        button: toggleBtn,
        targetId: targetId || '(next sibling)'
      });
      return;
    }

    // Toggle collapsed class on target
    const isCollapsed = targetElement.classList.contains('collapsed');
    if (isCollapsed) {
      targetElement.classList.remove('collapsed');
    } else {
      targetElement.classList.add('collapsed');
    }

    // Toggle active class on button for visual feedback
    toggleBtn.classList.toggle('active', !isCollapsed);

    // Swap button text if data-toggle-text is provided
    const alternateText = toggleBtn.getAttribute('data-toggle-text');
    if (alternateText) {
      const currentText = toggleBtn.textContent.trim();
      toggleBtn.textContent = alternateText;
      toggleBtn.setAttribute('data-toggle-text', currentText);
    }
  });
}
