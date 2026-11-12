/**
 * Expense form category filter
 *
 * Limits the category dropdown to the categories that accept the selected
 * expense type: the exclude category (type "any") plus either the expense
 * or the income categories. Without JS every category stays visible.
 *
 * Usage:
 *   <input type="radio" name="type" value="0">  (0 = expense, 1 = income)
 *   <select data-category-filter>
 *     <option value="">None</option>
 *     <option data-category-type="any">..</option>
 *     <optgroup data-category-type="charge">..</optgroup>
 *     <optgroup data-category-type="income">..</optgroup>
 *   </select>
 *
 * Optgroups are detached rather than hidden because hidden <option>s are
 * still selectable in Safari. The detached nodes are kept per select so
 * switching back restores them.
 */

const EXPENSE_TYPE_TO_CATEGORY_TYPE = { '0': 'charge', '1': 'income' };

// select element -> { charge: HTMLOptGroupElement, income: HTMLOptGroupElement }
const groupsBySelect = new WeakMap();

function groupsFor(select) {
  let groups = groupsBySelect.get(select);
  if (!groups) {
    groups = {};
    select.querySelectorAll('optgroup[data-category-type]').forEach(group => {
      groups[group.dataset.categoryType] = group;
    });
    groupsBySelect.set(select, groups);
  }
  return groups;
}

function filterSelect(select) {
  const form = select.closest('form');
  if (!form) return;

  const checked = form.querySelector('input[name="type"]:checked');
  const wanted = checked ? EXPENSE_TYPE_TO_CATEGORY_TYPE[checked.value] : null;
  if (!wanted) return;

  const groups = groupsFor(select);
  const selected = select.selectedOptions[0];

  Object.entries(groups).forEach(([type, group]) => {
    if (type === wanted) {
      if (!group.isConnected) select.appendChild(group);
    } else if (group.isConnected) {
      group.remove();
    }
  });

  // Drop a selection that no longer fits the chosen type.
  if (!selected || !selected.isConnected) {
    select.value = '';
  }
}

function filterAll(root) {
  root.querySelectorAll('select[data-category-filter]').forEach(filterSelect);
}

export function initializeCategoryFilter() {
  filterAll(document);

  document.body.addEventListener('change', function (e) {
    if (!e.target.matches('input[name="type"]')) return;
    const form = e.target.closest('form');
    if (form) filterAll(form);
  });

  // The expense form is re-rendered by HTMX after a failed submit.
  document.body.addEventListener('htmx:afterSwap', function () {
    filterAll(document);
  });
}
