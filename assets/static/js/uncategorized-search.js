/**
 * Uncategorized page search (client-side)
 *
 * The page renders every uncategorized group, so searching only hides cards
 * whose description doesn't contain the query (case-insensitive). The type
 * filter radios hide cards via CSS; this module keeps the filter counts and
 * the "no matches" message in sync with both.
 *
 * The server is only hit when assigning a category, and that swaps only the
 * results (the search box and filter stay put), so after the swap this module
 * re-applies the search to the new cards.
 *
 * Usage:
 *   <input id="uncategorized-search">
 *   <div class="uncategorized-group" data-type="charge|income" data-description="...">
 *   <span data-count="all|charge|income"></span>
 *   <p class="uncategorized-no-match" hidden></p>
 */

function applySearch() {
  const input = document.getElementById('uncategorized-search');
  if (!input) return;

  const query = input.value.trim().toLowerCase();
  const filter = document.querySelector('.uncategorized-filter input:checked')?.value || 'all';
  const counts = { all: 0, charge: 0, income: 0 };
  const cards = document.querySelectorAll('.uncategorized-group');
  let visible = 0;

  cards.forEach(card => {
    const matches = (card.dataset.description || '').toLowerCase().includes(query);
    card.hidden = !matches;
    if (!matches) return;

    counts.all++;
    counts[card.dataset.type]++;
    if (filter === 'all' || filter === card.dataset.type) visible++;
  });

  document.querySelectorAll('.uncategorized-filter [data-count]').forEach(el => {
    el.textContent = counts[el.dataset.count];
  });

  const noMatch = document.querySelector('.uncategorized-no-match');
  // With no cards at all the results show the "all categorized" state instead.
  if (noMatch) noMatch.hidden = visible > 0 || cards.length === 0;
}

export function initializeUncategorizedSearch() {
  applySearch();

  document.body.addEventListener('input', function (e) {
    if (e.target.id === 'uncategorized-search') applySearch();
  });

  document.body.addEventListener('change', function (e) {
    if (e.target.closest('.uncategorized-filter')) applySearch();
  });

  // Enter in the search box must not submit anything.
  document.body.addEventListener('keydown', function (e) {
    if (e.key === 'Enter' && e.target.id === 'uncategorized-search') e.preventDefault();
  });

  // The page is re-rendered after assigning a category.
  document.body.addEventListener('htmx:afterSwap', applySearch);
}
