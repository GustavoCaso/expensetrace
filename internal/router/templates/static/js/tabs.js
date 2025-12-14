/**
 * Tab switching system
 *
 * Usage:
 *   <button class="tab-btn" data-tab="basics">Basics</button>
 *   <button class="tab-btn" data-tab="examples">Examples</button>
 *   <div id="basics-tab" class="tab-content active">...</div>
 *   <div id="examples-tab" class="tab-content">...</div>
 *
 * Behavior:
 *   - Clicking a tab button with data-tab="X" activates the element with id="X-tab"
 *   - Removes 'active' class from all other tabs and content
 *   - Adds 'active' class to clicked tab and corresponding content
 */

export function initializeTabSystem() {
  document.body.addEventListener('click', function (e) {
    const tabBtn = e.target.closest('.tab-btn');
    if (!tabBtn) return;

    e.preventDefault();

    // Get all tab buttons and contents
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');

    // Remove active class from all
    tabBtns.forEach(btn => btn.classList.remove('active'));
    tabContents.forEach(content => content.classList.remove('active'));

    // Add active class to clicked tab and its content
    tabBtn.classList.add('active');
    const tabId = tabBtn.getAttribute('data-tab');
    const targetContent = document.getElementById(tabId + '-tab');

    if (targetContent) {
      targetContent.classList.add('active');
    } else {
      console.warn('Tab content not found for tab:', tabId);
    }
  });
}
