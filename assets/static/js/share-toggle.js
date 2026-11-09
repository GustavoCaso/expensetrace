/**
 * Category share toggle (reports page)
 *
 * Switches expense category percentages between "share of spending" and
 * "share of income". The choice is stored on <html data-share-basis>, which
 * HTMX never swaps, so it survives loading another month's card. CSS uses
 * that attribute to show the matching values; this module only records the
 * choice and keeps the buttons' aria-pressed in sync.
 *
 * Usage:
 *   <button class="share-toggle-btn" data-share-basis="income">income</button>
 *   <span class="share-of-spending">..</span><span class="share-of-income">..</span>
 */

const STORAGE_KEY = 'expensetrace.shareBasis';
const BASES = ['spending', 'income'];

function readStoredBasis() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return BASES.includes(stored) ? stored : 'spending';
  } catch {
    return 'spending';
  }
}

function storeBasis(basis) {
  try {
    localStorage.setItem(STORAGE_KEY, basis);
  } catch {
    // Storage unavailable (private mode, blocked); the choice just won't persist.
  }
}

function applyBasis(basis) {
  document.documentElement.dataset.shareBasis = basis;
  document.querySelectorAll('.share-toggle-btn').forEach(btn => {
    btn.setAttribute('aria-pressed', String(btn.dataset.shareBasis === basis));
  });
}

export function initializeShareToggle() {
  applyBasis(readStoredBasis());

  document.body.addEventListener('click', function (e) {
    const btn = e.target.closest('.share-toggle-btn');
    if (!btn || !BASES.includes(btn.dataset.shareBasis)) return;

    storeBasis(btn.dataset.shareBasis);
    applyBasis(btn.dataset.shareBasis);
  });

  // A swapped-in report card renders its buttons in the default state.
  document.body.addEventListener('htmx:afterSwap', function () {
    applyBasis(document.documentElement.dataset.shareBasis || 'spending');
  });
}
