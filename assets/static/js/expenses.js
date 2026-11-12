/**
 * Main entry point for ExpenseTrace frontend functionality
 *
 * This file imports and initializes all application modules:
 * - Toggle system for collapsible UI elements
 * - Tab switching for category patterns
 * - Financial bar chart with savings visualization
 * - Category share toggle (spending vs income) on reports
 * - Client-side search on the uncategorized page
 */

// Import all modules
import { initializeToggleSystem } from './toggle.js';
import { initializeTabSystem } from './tabs.js';
import { initializeBarChart } from './charts/bar-chart.js';
import { initializeShareToggle } from './share-toggle.js';
import { initializeUncategorizedSearch } from './uncategorized-search.js';


// Initialize all systems when DOM is ready
document.addEventListener('DOMContentLoaded', function () {
  // Core UI systems
  initializeToggleSystem();
  initializeTabSystem();
  initializeShareToggle();
  initializeUncategorizedSearch();

  // Charts (these check for their containers before initializing)
  initializeBarChart();
});
