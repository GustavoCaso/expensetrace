/**
 * Utility functions for formatting data
 */

/**
 * Format currency amount in cents to display format
 * @param {number} amount - Amount in cents (e.g., 12345 = €123.45)
 * @returns {string} Formatted currency string (e.g., "123,45€")
 */
export function formatMoney(amount) {
  const absAmount = Math.abs(amount);
  const sign = amount < 0 ? '-' : '';

  let intPart = Math.floor(absAmount / 100).toString();
  const decPart = (absAmount % 100).toString().padStart(2, '0');

  // Add thousands separator
  if (intPart.length > 3) {
    intPart = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
  }

  return `${sign}${intPart},${decPart}€`;
}

/**
 * Format date string to display format
 * @param {string} dateString - ISO date string
 * @returns {string} Formatted date (e.g., "Jan 15, 2024")
 */
export function formatDate(dateString) {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}
